package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SeatStatus string

const (
	StatusAvailable SeatStatus = "AVAILABLE"
	StatusLocked    SeatStatus = "LOCKED"
	StatusBooked    SeatStatus = "BOOKED"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// User represents an authenticated user
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FirebaseUID string           `bson:"firebase_uid" json:"firebase_uid"`
	Email     string             `bson:"email" json:"email"`
	Name      string             `bson:"name" json:"name"`
	Role      Role               `bson:"role" json:"role"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// Movie represents a movie
type Movie struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	PosterURL   string             `bson:"poster_url" json:"poster_url"`
	DurationMin int                `bson:"duration_min" json:"duration_min"`
}

// Showtime represents a screening session
type Showtime struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MovieID   primitive.ObjectID `bson:"movie_id" json:"movie_id"`
	MovieTitle string            `bson:"movie_title" json:"movie_title"`
	StartsAt  time.Time          `bson:"starts_at" json:"starts_at"`
	Hall      string             `bson:"hall" json:"hall"`
	Seats     []Seat             `bson:"seats" json:"seats"`
}

// Seat represents a single seat in a showtime
type Seat struct {
	ID     string     `bson:"id" json:"id"`         // e.g. "A1", "B3"
	Row    string     `bson:"row" json:"row"`
	Number int        `bson:"number" json:"number"`
	Status SeatStatus `bson:"status" json:"status"`
	Price  float64    `bson:"price" json:"price"`
}

// Booking represents a completed booking
type Booking struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
	UserEmail   string             `bson:"user_email" json:"user_email"`
	ShowtimeID  primitive.ObjectID `bson:"showtime_id" json:"showtime_id"`
	MovieTitle  string             `bson:"movie_title" json:"movie_title"`
	SeatID      string             `bson:"seat_id" json:"seat_id"`
	TotalPrice  float64            `bson:"total_price" json:"total_price"`
	Status      string             `bson:"status" json:"status"` // confirmed, timeout
	BookedAt    time.Time          `bson:"booked_at" json:"booked_at"`
}

// AuditLog represents an audit event
type AuditLog struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Event     string                 `bson:"event" json:"event"`
	Payload   map[string]interface{} `bson:"payload" json:"payload"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
}

// WSMessage is the WebSocket broadcast message
type WSMessage struct {
	Type       string     `json:"type"`        // seat_update, booking_confirmed, seat_released
	ShowtimeID string     `json:"showtime_id"`
	SeatID     string     `json:"seat_id"`
	Status     SeatStatus `json:"status"`
	UserID     string     `json:"user_id,omitempty"`
}

// MQEvent is published to RabbitMQ
type MQEvent struct {
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}
