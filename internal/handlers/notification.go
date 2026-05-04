package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationHandler handles in-app notification endpoints.
type NotificationHandler struct {
	DB *pgxpool.Pool
}

type Notification struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateNotification inserts a notification record for a user.
// This is a helper used by other handlers (booking, sponsor, etc.) — not an HTTP handler.
func CreateNotification(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID, title, message, notifType string) {
	_, _ = db.Exec(ctx,
		`INSERT INTO notifications (user_id, title, message, type) VALUES ($1, $2, $3, $4)`,
		userID, title, message, notifType,
	)
}

// ListNotifications handles GET /notifications — returns all notifications for the authenticated user.
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	raw, _ := c.Get("user_id")
	userID, ok := raw.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	rows, err := h.DB.Query(c,
		`SELECT id, title, message, type, is_read, created_at
		 FROM notifications
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT 100`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch notifications"})
		return
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Title, &n.Message, &n.Type, &n.IsRead, &n.CreatedAt); err != nil {
			continue
		}
		notifications = append(notifications, n)
	}
	if notifications == nil {
		notifications = []Notification{}
	}

	c.JSON(http.StatusOK, notifications)
}

// MarkRead handles PATCH /notifications/:id/read — marks one notification as read.
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	raw, _ := c.Get("user_id")
	userID, ok := raw.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	notifID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}

	_, err = h.DB.Exec(c,
		`UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`,
		notifID, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "marked as read"})
}

// MarkAllRead handles PATCH /notifications/read-all — marks all notifications for the user as read.
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	raw, _ := c.Get("user_id")
	userID, ok := raw.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, err = h.DB.Exec(c,
		`UPDATE notifications SET is_read = true WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark all as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all notifications marked as read"})
}

// UnreadCount handles GET /notifications/unread-count — returns count of unread notifications.
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	raw, _ := c.Get("user_id")
	userID, ok := raw.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var count int
	err = h.DB.QueryRow(c,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`,
		userID,
	).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}
