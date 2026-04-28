package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
	"strings"

	"github.com/cinema-booking/backend/models"
	"github.com/cinema-booking/backend/mq"
	"github.com/cinema-booking/backend/repository"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const lockTTL = 5 * time.Minute

var (
	ErrSeatLocked    = errors.New("seat is already locked")
	ErrSeatBooked    = errors.New("seat is already booked")
	ErrLockExpired   = errors.New("lock expired or not owned by user")
	ErrShowtimeNotFound = errors.New("showtime not found")
)

// lockKey returns a Redis key for a seat lock
func lockKey(showtimeID, seatID string) string {
	return fmt.Sprintf("seat_lock:%s:%s", showtimeID, seatID)
}

type BookingService struct {
	redis       *redis.Client
	showtimeRepo *repository.ShowtimeRepository
	bookingRepo  *repository.BookingRepository
	producer     *mq.Producer
	hub          *WSHub
}

func NewBookingService(
	rdb *redis.Client,
	showtimeRepo *repository.ShowtimeRepository,
	bookingRepo *repository.BookingRepository,
	producer *mq.Producer,
	hub *WSHub,
) *BookingService {
	return &BookingService{
		redis:        rdb,
		showtimeRepo: showtimeRepo,
		bookingRepo:  bookingRepo,
		producer:     producer,
		hub:          hub,
	}
}

// LockSeat attempts to acquire a distributed lock for a seat.
// Uses SET NX EX for atomicity — only one user wins even under heavy concurrent load.
func (s *BookingService) LockSeat(ctx context.Context, showtimeID, seatID, userID string) error {
	// Verify showtime and seat exist
	stID, err := primitive.ObjectIDFromHex(showtimeID)
	if err != nil {
		return ErrShowtimeNotFound
	}
	showtime, err := s.showtimeRepo.FindByID(ctx, stID)
	if err != nil {
		return ErrShowtimeNotFound
	}

	var seat *models.Seat
	for i := range showtime.Seats {
		if showtime.Seats[i].ID == seatID {
			seat = &showtime.Seats[i]
			break
		}
	}
	if seat == nil {
		return fmt.Errorf("seat %s not found", seatID)
	}
	if seat.Status == models.StatusBooked {
		return ErrSeatBooked
	}

	// Atomic SET NX EX — only succeeds if key does not exist
	key := lockKey(showtimeID, seatID)
	ok, err := s.redis.SetNX(ctx, key, userID, lockTTL).Result()
	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	if !ok {
		return ErrSeatLocked
	}

	// Update seat status in MongoDB
	if err := s.showtimeRepo.UpdateSeatStatus(ctx, stID, seatID, models.StatusLocked); err != nil {
		// Rollback redis lock if mongo fails
		s.redis.Del(ctx, key)
		return err
	}

	// Broadcast to all connected clients
	s.hub.Broadcast(models.WSMessage{
		Type:       "seat_update",
		ShowtimeID: showtimeID,
		SeatID:     seatID,
		Status:     models.StatusLocked,
		UserID:     userID,
	})

	log.Printf("[LOCK] Seat %s locked by user %s for showtime %s", seatID, userID, showtimeID)
	return nil
}

// ConfirmBooking verifies lock ownership then finalises the booking atomically using a Lua script.
func (s *BookingService) ConfirmBooking(ctx context.Context, showtimeID, seatID, userID, userEmail string) (*models.Booking, error) {
	key := lockKey(showtimeID, seatID)

	// Lua script: check lock owner and delete atomically
	// This prevents a race condition where the lock expires between GET and DEL
	luaScript := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	result, err := luaScript.Run(ctx, s.redis, []string{key}, userID).Int()
	if err != nil || result == 0 {
		return nil, ErrLockExpired
	}

	stID, _ := primitive.ObjectIDFromHex(showtimeID)
	showtime, err := s.showtimeRepo.FindByID(ctx, stID)
	if err != nil {
		return nil, ErrShowtimeNotFound
	}

	var price float64
	for _, seat := range showtime.Seats {
		if seat.ID == seatID {
			price = seat.Price
			break
		}
	}

	userObjID, _ := primitive.ObjectIDFromHex(userID)
	booking := &models.Booking{
		UserID:     userObjID,
		UserEmail:  userEmail,
		ShowtimeID: stID,
		MovieTitle: showtime.MovieTitle,
		SeatID:     seatID,
		TotalPrice: price,
		Status:     "confirmed",
	}

	if err := s.bookingRepo.Create(ctx, booking); err != nil {
		return nil, err
	}

	// Update seat to BOOKED in MongoDB
	_ = s.showtimeRepo.UpdateSeatStatus(ctx, stID, seatID, models.StatusBooked)

	// Broadcast BOOKED status
	s.hub.Broadcast(models.WSMessage{
		Type:       "seat_update",
		ShowtimeID: showtimeID,
		SeatID:     seatID,
		Status:     models.StatusBooked,
	})

	// Publish to RabbitMQ (async audit + notification)
	_ = s.producer.Publish("booking.completed", map[string]interface{}{
		"booking_id": booking.ID.Hex(),
		"user_id":    userID,
		"user_email": userEmail,
		"showtime_id": showtimeID,
		"seat_id":    seatID,
		"price":      price,
	})

	return booking, nil
}

// WatchExpiredLocks subscribes to Redis keyspace notifications for expired lock keys.
// When a lock expires without payment, the seat is released automatically.
func (s *BookingService) WatchExpiredLocks(ctx context.Context) {
	pubsub := s.redis.PSubscribe(ctx, "__keyevent@0__:expired")
	defer pubsub.Close()

	log.Println("[LOCK] Watching for expired seat locks...")

	for msg := range pubsub.Channel() {
		key := msg.Payload
		parts := strings.SplitN(key, ":", 3)
		if len(parts) != 3 {
    		continue
		}
		st := parts[1]
		seat := parts[2]

		log.Printf("[LOCK] Lock expired for seat %s showtime %s — releasing", seat, st)

		stID, err := primitive.ObjectIDFromHex(st)
		if err != nil {
			continue
		}

		_ = s.showtimeRepo.UpdateSeatStatus(ctx, stID, seat, models.StatusAvailable)

		s.hub.Broadcast(models.WSMessage{
			Type:       "seat_update",
			ShowtimeID: st,
			SeatID:     seat,
			Status:     models.StatusAvailable,
		})

		_ = s.producer.Publish("booking.timeout", map[string]interface{}{
			"showtime_id": st,
			"seat_id":     seat,
		})
	}
}
