package repository

import (
	"context"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/cinema-booking/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository struct {
	col *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{col: db.Collection("users")}
}

func (r *UserRepository) UpsertByFirebaseUID(ctx context.Context, token *auth.Token) (*models.User, error) {
	email, _ := token.Claims["email"].(string)
	name, _ := token.Claims["name"].(string)

	filter := bson.M{"firebase_uid": token.UID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"_id":        primitive.NewObjectID(),
			"role":       models.RoleUser,
			"created_at": time.Now(),
		},
		"$set": bson.M{
			"firebase_uid": token.UID,
			"email":        email,
			"name":         name,
		},
	}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var user models.User
	err := r.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&user)
	return &user, err
}

func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	return &user, err
}

// ---

type ShowtimeRepository struct {
	col *mongo.Collection
}

func NewShowtimeRepository(db *mongo.Database) *ShowtimeRepository {
	return &ShowtimeRepository{col: db.Collection("showtimes")}
}

func (r *ShowtimeRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Showtime, error) {
	var st models.Showtime
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&st)
	return &st, err
}

func (r *ShowtimeRepository) FindAll(ctx context.Context) ([]models.Showtime, error) {
	cursor, err := r.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var results []models.Showtime
	return results, cursor.All(ctx, &results)
}

func (r *ShowtimeRepository) UpdateSeatStatus(ctx context.Context, showtimeID primitive.ObjectID, seatID string, status models.SeatStatus) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"_id": showtimeID, "seats.id": seatID},
		bson.M{"$set": bson.M{"seats.$.status": status}},
	)
	return err
}

func (r *ShowtimeRepository) Seed(ctx context.Context) error {
	count, _ := r.col.CountDocuments(ctx, bson.M{})
	if count > 0 {
		return nil
	}

	rows := []string{"A", "B", "C", "D", "E"}
	var seats []models.Seat
	for _, row := range rows {
		for i := 1; i <= 8; i++ {
			seats = append(seats, models.Seat{
				ID:     row + string(rune('0'+i)),
				Row:    row,
				Number: i,
				Status: models.StatusAvailable,
				Price:  200.0,
			})
		}
	}

	now := time.Now()
	showtimes := []interface{}{
		models.Showtime{
			ID:         primitive.NewObjectID(),
			MovieTitle: "Interstellar",
			StartsAt:   now.Add(2 * time.Hour),
			Hall:       "Hall 1",
			Seats:      seats,
		},
		models.Showtime{
			ID:         primitive.NewObjectID(),
			MovieTitle: "Dune: Part Two",
			StartsAt:   now.Add(5 * time.Hour),
			Hall:       "Hall 2",
			Seats:      seats,
		},
	}
	_, err := r.col.InsertMany(ctx, showtimes)
	return err
}

// ---

type BookingRepository struct {
	col *mongo.Collection
}

func NewBookingRepository(db *mongo.Database) *BookingRepository {
	return &BookingRepository{col: db.Collection("bookings")}
}

func (r *BookingRepository) Create(ctx context.Context, b *models.Booking) error {
	b.ID = primitive.NewObjectID()
	b.BookedAt = time.Now()
	_, err := r.col.InsertOne(ctx, b)
	return err
}

func (r *BookingRepository) FindAll(ctx context.Context, filter bson.M) ([]models.Booking, error) {
	cursor, err := r.col.Find(ctx, filter, options.Find().SetSort(bson.M{"booked_at": -1}))
	if err != nil {
		return nil, err
	}
	var results []models.Booking
	return results, cursor.All(ctx, &results)
}

// ---

type AuditLogRepository struct {
	col *mongo.Collection
}

func NewAuditLogRepository(db *mongo.Database) *AuditLogRepository {
	return &AuditLogRepository{col: db.Collection("audit_logs")}
}

func (r *AuditLogRepository) Create(ctx context.Context, event string, payload map[string]interface{}) error {
	log := models.AuditLog{
		ID:        primitive.NewObjectID(),
		Event:     event,
		Payload:   payload,
		CreatedAt: time.Now(),
	}
	_, err := r.col.InsertOne(ctx, log)
	return err
}

func (r *AuditLogRepository) FindAll(ctx context.Context) ([]models.AuditLog, error) {
	cursor, err := r.col.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(200))
	if err != nil {
		return nil, err
	}
	var results []models.AuditLog
	return results, cursor.All(ctx, &results)
}
