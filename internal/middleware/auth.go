package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Nysonn/campuscare/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func AuthRequired(sessionService *services.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var raw string

		if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			raw = strings.TrimPrefix(auth, "Bearer ")
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		sessionID, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
			c.Abort()
			return
		}

		userID, err := sessionService.GetUser(sessionID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// UpdateLastActive fires a non-blocking DB update after each authenticated request
// so that the partner-notify endpoint can determine if a user is currently online.
func UpdateLastActive(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		userID, exists := c.Get("user_id")
		if !exists {
			return
		}
		go db.Exec(context.Background(),
			`UPDATE users SET last_active_at = now() WHERE id = $1`,
			userID.(uuid.UUID),
		)
	}
}
