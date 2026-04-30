package services

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/cinema-booking/backend/models"
	"github.com/redis/go-redis/v9"
)

func setupRedis(t *testing.T) *redis.Client {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestLockSeat_Success(t *testing.T) {
	rdb := setupRedis(t)
	key := "seat_lock:showtime1:A1"

	ok, err := rdb.SetNX(context.Background(), key, "user1", lockTTL).Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected lock to succeed but it failed")
	}
}

func TestLockSeat_AlreadyLocked(t *testing.T) {
	rdb := setupRedis(t)
	key := "seat_lock:showtime1:A1"
	ctx := context.Background()

	rdb.SetNX(ctx, key, "user1", lockTTL)

	ok, err := rdb.SetNX(ctx, key, "user2", lockTTL).Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected lock to fail but it succeeded")
	}
}

func TestUnlockSeat_WrongOwner(t *testing.T) {
	rdb := setupRedis(t)
	key := "seat_lock:showtime1:A1"
	ctx := context.Background()

	rdb.SetNX(ctx, key, "user1", lockTTL)

	luaScript := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	result, _ := luaScript.Run(ctx, rdb, []string{key}, "user2").Int()
	if result != 0 {
		t.Error("expected unlock to fail for wrong owner but it succeeded")
	}
}

func TestUnlockSeat_CorrectOwner(t *testing.T) {
	rdb := setupRedis(t)
	key := "seat_lock:showtime1:A1"
	ctx := context.Background()

	rdb.SetNX(ctx, key, "user1", lockTTL)

	luaScript := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	result, _ := luaScript.Run(ctx, rdb, []string{key}, "user1").Int()
	if result != 1 {
		t.Error("expected unlock to succeed for correct owner but it failed")
	}
}

func TestSeatStatus(t *testing.T) {
	seat := models.Seat{
		ID:     "A1",
		Status: models.StatusAvailable,
	}
	if seat.Status != models.StatusAvailable {
		t.Errorf("expected AVAILABLE but got %s", seat.Status)
	}
}