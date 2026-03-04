package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string
	DatabaseURL string
	SessionTTL  int
}

func Load() *Config {
	_ = godotenv.Load()

	ttl, err := strconv.Atoi(os.Getenv("SESSION_TTL_HOURS"))
	if err != nil {
		log.Fatal("Invalid SESSION_TTL_HOURS")
	}

	return &Config{
		AppPort:     os.Getenv("APP_PORT"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		SessionTTL:  ttl,
	}
}
