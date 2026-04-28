package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	firebase "firebase.google.com/go/v4"
	"github.com/cinema-booking/backend/config"
	"github.com/cinema-booking/backend/handlers"
	"github.com/cinema-booking/backend/middleware"
	"github.com/cinema-booking/backend/mq"
	"github.com/cinema-booking/backend/repository"
	"github.com/cinema-booking/backend/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/api/option"
)

func main() {
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	ctx := context.Background()

	// --- MongoDB ---
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("MongoDB connect failed: %v", err)
	}
	defer mongoClient.Disconnect(ctx)
	db := mongoClient.Database("cinema")

	// --- Redis ---
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis connect failed: %v", err)
	}

	// --- Firebase ---
	var credJSON map[string]interface{}
	if err := json.Unmarshal([]byte(cfg.FirebaseCredJSON), &credJSON); err != nil {
		log.Fatalf("Invalid FIREBASE_CREDENTIALS_JSON: %v", err)
	}
	credBytes, _ := json.Marshal(credJSON)
	firebaseApp, err := firebase.NewApp(ctx, nil, option.WithCredentialsJSON(credBytes))
	if err != nil {
		log.Fatalf("Firebase init failed: %v", err)
	}
	firebaseAuth, err := firebaseApp.Auth(ctx)
	if err != nil {
		log.Fatalf("Firebase auth init failed: %v", err)
	}

	// --- Repositories ---
	userRepo := repository.NewUserRepository(db)
	showtimeRepo := repository.NewShowtimeRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)

	// Seed initial data
	if err := showtimeRepo.Seed(ctx); err != nil {
		log.Printf("Seed warning: %v", err)
	}

	// --- RabbitMQ ---
	producer, err := mq.NewProducer(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("RabbitMQ producer failed: %v", err)
	}
	defer producer.Close()

	consumer, err := mq.NewConsumer(cfg.RabbitMQURL, auditRepo)
	if err != nil {
		log.Fatalf("RabbitMQ consumer failed: %v", err)
	}
	defer consumer.Close()
	consumer.Start()

	// --- WebSocket Hub ---
	hub := services.NewWSHub()

	// --- Services ---
	bookingSvc := services.NewBookingService(rdb, showtimeRepo, bookingRepo, producer, hub)
	go bookingSvc.WatchExpiredLocks(ctx)

	// --- Middleware ---
	authMW := middleware.NewAuthMiddleware(firebaseAuth, userRepo)

	// --- Handlers ---
	authHandler := handlers.NewAuthHandler(userRepo)
	showtimeHandler := handlers.NewShowtimeHandler(showtimeRepo)
	bookingHandler := handlers.NewBookingHandler(bookingSvc, bookingRepo)
	wsHandler := handlers.NewWSHandler(hub)
	adminHandler := handlers.NewAdminHandler(bookingRepo, auditRepo, showtimeRepo)

	// --- Router ---
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// WebSocket (no auth required — showtime ID is public)
	r.GET("/ws/:showtimeId", wsHandler.Handle)

	// User routes (auth required)
	user := r.Group("/api", authMW.Authenticate())
	{
		user.GET("/me", authHandler.Me)
		user.GET("/showtimes", showtimeHandler.List)
		user.GET("/showtimes/:id", showtimeHandler.Get)
		user.POST("/bookings/lock", bookingHandler.LockSeat)
		user.POST("/bookings/confirm", bookingHandler.ConfirmBooking)
		user.GET("/bookings/mine", bookingHandler.MyBookings)
	}

	// Admin routes (auth + admin role required)
	admin := r.Group("/api/admin", authMW.Authenticate(), middleware.RequireRole("admin"))
	{
		admin.GET("/bookings", adminHandler.ListBookings)
		admin.GET("/audit-logs", adminHandler.ListAuditLogs)
	}

	log.Printf("🎬 Cinema backend running on :%s", cfg.Port)
	r.Run(":" + cfg.Port)
}
