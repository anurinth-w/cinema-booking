package config

import (
	"log"
	"os"
)

type Config struct {
	Port              string
	MongoURI          string
	RedisAddr         string
	RabbitMQURL       string
	FirebaseProjectID string
	FirebaseCredJSON  string
	GinMode           string
}

func Load() *Config {
	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		MongoURI:          mustEnv("MONGO_URI"),
		RedisAddr:         getEnv("REDIS_ADDR", "redis:6379"),
		RabbitMQURL:       mustEnv("RABBITMQ_URL"),
		FirebaseProjectID: mustEnv("FIREBASE_PROJECT_ID"),
		FirebaseCredJSON:  mustEnv("FIREBASE_CREDENTIALS_JSON"),
		GinMode:           getEnv("GIN_MODE", "debug"),
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
