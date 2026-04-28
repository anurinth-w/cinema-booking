package handlers

import (
	"context"
	"net/http"

	"github.com/cinema-booking/backend/models"
	"github.com/cinema-booking/backend/repository"
	"github.com/cinema-booking/backend/services"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// --- Auth Handler ---

type AuthHandler struct {
	userRepo *repository.UserRepository
}

func NewAuthHandler(userRepo *repository.UserRepository) *AuthHandler {
	return &AuthHandler{userRepo: userRepo}
}

func (h *AuthHandler) Me(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	c.JSON(http.StatusOK, user)
}

// --- Showtime Handler ---

type ShowtimeHandler struct {
	showtimeRepo *repository.ShowtimeRepository
}

func NewShowtimeHandler(repo *repository.ShowtimeRepository) *ShowtimeHandler {
	return &ShowtimeHandler{showtimeRepo: repo}
}

func (h *ShowtimeHandler) List(c *gin.Context) {
	showtimes, err := h.showtimeRepo.FindAll(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, showtimes)
}

func (h *ShowtimeHandler) Get(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	showtime, err := h.showtimeRepo.FindByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "showtime not found"})
		return
	}
	c.JSON(http.StatusOK, showtime)
}

// --- Booking Handler ---

type BookingHandler struct {
	bookingSvc  *services.BookingService
	bookingRepo *repository.BookingRepository
}

func NewBookingHandler(svc *services.BookingService, repo *repository.BookingRepository) *BookingHandler {
	return &BookingHandler{bookingSvc: svc, bookingRepo: repo}
}

type LockRequest struct {
	ShowtimeID string `json:"showtime_id" binding:"required"`
	SeatID     string `json:"seat_id" binding:"required"`
}

func (h *BookingHandler) LockSeat(c *gin.Context) {
	var req LockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := c.MustGet("user").(*models.User)

	err := h.bookingSvc.LockSeat(context.Background(), req.ShowtimeID, req.SeatID, user.ID.Hex())
	if err != nil {
		switch err {
		case services.ErrSeatLocked:
			c.JSON(http.StatusConflict, gin.H{"error": "seat is already locked"})
		case services.ErrSeatBooked:
			c.JSON(http.StatusConflict, gin.H{"error": "seat is already booked"})
		case services.ErrShowtimeNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "showtime not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "seat locked", "ttl_seconds": 300})
}

type ConfirmRequest struct {
	ShowtimeID string `json:"showtime_id" binding:"required"`
	SeatID     string `json:"seat_id" binding:"required"`
}

func (h *BookingHandler) ConfirmBooking(c *gin.Context) {
	var req ConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := c.MustGet("user").(*models.User)

	booking, err := h.bookingSvc.ConfirmBooking(
		context.Background(),
		req.ShowtimeID, req.SeatID,
		user.ID.Hex(), user.Email,
	)
	if err != nil {
		switch err {
		case services.ErrLockExpired:
			c.JSON(http.StatusGone, gin.H{"error": "lock expired, please select seat again"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, booking)
}

func (h *BookingHandler) MyBookings(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	bookings, err := h.bookingRepo.FindAll(context.Background(), bson.M{"user_id": user.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bookings)
}

// --- WebSocket Handler ---

type WSHandler struct {
	hub *services.WSHub
}

func NewWSHandler(hub *services.WSHub) *WSHandler {
	return &WSHandler{hub: hub}
}

func (h *WSHandler) Handle(c *gin.Context) {
	showtimeID := c.Param("showtimeId")
	if showtimeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "showtimeId required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &services.Client{
		Conn:       conn,
		ShowtimeID: showtimeID,
		Send:       make(chan []byte, 64),
	}

	h.hub.Register(client)
	go client.WritePump()
	client.ReadPump(h.hub)
}

// --- Admin Handler ---

type AdminHandler struct {
	bookingRepo  *repository.BookingRepository
	auditRepo    *repository.AuditLogRepository
	showtimeRepo *repository.ShowtimeRepository
}

func NewAdminHandler(
	bookingRepo *repository.BookingRepository,
	auditRepo *repository.AuditLogRepository,
	showtimeRepo *repository.ShowtimeRepository,
) *AdminHandler {
	return &AdminHandler{
		bookingRepo:  bookingRepo,
		auditRepo:    auditRepo,
		showtimeRepo: showtimeRepo,
	}
}

func (h *AdminHandler) ListBookings(c *gin.Context) {
	filter := bson.M{}

	if movieTitle := c.Query("movie"); movieTitle != "" {
		filter["movie_title"] = bson.M{"$regex": movieTitle, "$options": "i"}
	}
	if userEmail := c.Query("user"); userEmail != "" {
		filter["user_email"] = bson.M{"$regex": userEmail, "$options": "i"}
	}
	if date := c.Query("date"); date != "" {
		// filter by date prefix on booked_at (simple approach)
		filter["$expr"] = bson.M{
			"$eq": bson.A{
				bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$booked_at"}},
				date,
			},
		}
	}

	bookings, err := h.bookingRepo.FindAll(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bookings)
}

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	logs, err := h.auditRepo.FindAll(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}
