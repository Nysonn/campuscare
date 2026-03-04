package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RequireRole(db *pgxpool.Pool, role string) gin.HandlerFunc {
	return func(c *gin.Context) {

		userID := c.MustGet("user_id").(uuid.UUID)

		var userRole string
		err := db.QueryRow(context.Background(),
			`SELECT role FROM users WHERE id=$1`,
			userID,
		).Scan(&userRole)

		if err != nil || userRole != role {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			c.Abort()
			return
		}

		c.Next()
	}
}
