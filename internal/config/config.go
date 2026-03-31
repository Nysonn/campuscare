package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort         string
	DatabaseURL     string
	SessionTTL      int
	StreamAPIKey    string
	StreamAPISecret string
}

func Load() *Config {
	_ = godotenv.Load()

	ttl, err := strconv.Atoi(os.Getenv("SESSION_TTL_HOURS"))
	if err != nil {
		log.Fatal("Invalid SESSION_TTL_HOURS")
	}

	return &Config{
		AppPort:         os.Getenv("APP_PORT"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		SessionTTL:      ttl,
		StreamAPIKey:    os.Getenv("STREAM_API_KEY"),
		StreamAPISecret: os.Getenv("STREAM_API_SECRET"),
	}
}
