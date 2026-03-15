package middleware

import (
	"net/http"
	"strings"

	"github.com/Nysonn/campuscare/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
