package db

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(databaseURL string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatal("Unable to connect to DB:", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("DB ping failed:", err)
	}

	return pool
}
