package chatbot

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetRecentMessages(db *pgxpool.Pool, userID uuid.UUID, limit int) ([]map[string]string, error) {

	rows, err := db.Query(context.Background(),
		`SELECT role, content
		 FROM chatbot_messages
		 WHERE user_id=$1
		 ORDER BY created_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []map[string]string

	for rows.Next() {
		var role, content string
		rows.Scan(&role, &content)

		messages = append(messages, map[string]string{
			"role":    role,
			"content": content,
		})
	}

	return messages, nil
}

func StoreMessage(db *pgxpool.Pool, userID uuid.UUID, role, content string) {
	db.Exec(context.Background(),
		`INSERT INTO chatbot_messages (user_id, role, content)
		 VALUES ($1, $2, $3)`,
		userID, role, content,
	)
}
