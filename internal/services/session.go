package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionService struct {
	DB         *pgxpool.Pool
	SessionTTL int
}

func (s *SessionService) Create(userID uuid.UUID) (uuid.UUID, error) {
	id := uuid.New()
	expiry := time.Now().Add(time.Duration(s.SessionTTL) * time.Hour)

	_, err := s.DB.Exec(context.Background(),
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1,$2,$3)`,
		id, userID, expiry,
	)

	return id, err
}

func (s *SessionService) GetUser(sessionID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := s.DB.QueryRow(context.Background(),
		`SELECT user_id FROM sessions WHERE id=$1 AND expires_at > now()`,
		sessionID,
	).Scan(&userID)

	return userID, err
}

func (s *SessionService) Delete(sessionID uuid.UUID) error {
	_, err := s.DB.Exec(context.Background(),
		`DELETE FROM sessions WHERE id=$1`, sessionID)
	return err
}
