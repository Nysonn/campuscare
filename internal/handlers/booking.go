package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/Nysonn/campuscare/internal/audit"
	calendarPkg "github.com/Nysonn/campuscare/internal/calendar"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingHandler struct {
	DB *pgxpool.Pool
}

type CreateBookingRequest struct {
	CounselorID string `json:"counselor_id"`
	Type        string `json:"type"`       // online or physical
	StartTime   string `json:"start_time"` // RFC3339
	EndTime     string `json:"end_time"`
	Location    string `json:"location"`
}

func (h *BookingHandler) Create(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)

	var req CreateBookingRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if req.Type != "online" && req.Type != "physical" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be online or physical"})
		return
	}

	counselorID, _ := uuid.Parse(req.CounselorID)
	start, _ := time.Parse(time.RFC3339, req.StartTime)
	end, _ := time.Parse(time.RFC3339, req.EndTime)

	tx, _ := h.DB.Begin(context.Background())
	defer tx.Rollback(context.Background())

	// Overlap check
	var exists bool
	err := tx.QueryRow(context.Background(),
		`SELECT EXISTS (
			SELECT 1 FROM bookings
			WHERE counselor_id=$1
			AND status IN ('pending','accepted')
			AND $2 < end_time
			AND $3 > start_time
		)`,
		counselorID, start, end,
	).Scan(&exists)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Availability check failed"})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Time slot unavailable"})
		return
	}

	var bookingID uuid.UUID
	tx.QueryRow(context.Background(),
		`INSERT INTO bookings
		 (student_id,counselor_id,type,start_time,end_time,location)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING id`,
		userID, counselorID, req.Type, start, end, req.Location,
	).Scan(&bookingID)

	tx.Commit(context.Background())

	audit.Log(h.DB, userID, "CREATE_BOOKING", "booking", bookingID, nil)

	c.JSON(http.StatusCreated, gin.H{"booking_id": bookingID})
}

func (h *BookingHandler) UpdateStatus(c *gin.Context) {

	counselorID := c.MustGet("user_id").(uuid.UUID)
	idParam := c.Param("id")
	bookingID, _ := uuid.Parse(idParam)

	var body struct {
		Status string `json:"status"`
	}
	c.BindJSON(&body)

	if body.Status != "accepted" && body.Status != "declined" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be accepted or declined"})
		return
	}

	_, err := h.DB.Exec(context.Background(),
		`UPDATE bookings
		 SET status=$1
		 WHERE id=$2 AND counselor_id=$3`,
		body.Status, bookingID, counselorID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}

	// If accepted → create Google Calendar event
	if body.Status == "accepted" {
		go h.createCalendarEvent(bookingID)
	}

	audit.Log(h.DB, counselorID, "UPDATE_BOOKING", "booking", bookingID, body)

	c.JSON(http.StatusOK, gin.H{"message": "Booking updated"})
}

func (h *BookingHandler) createCalendarEvent(bookingID uuid.UUID) {
	var start, end time.Time
	var location string

	err := h.DB.QueryRow(context.Background(),
		`SELECT start_time, end_time, location FROM bookings WHERE id=$1`,
		bookingID,
	).Scan(&start, &end, &location)
	if err != nil {
		return
	}

	srv, err := calendarPkg.NewService()
	if err != nil {
		return
	}

	calendarPkg.CreateEvent(srv, "Counseling Session: "+location, start.Format(time.RFC3339), end.Format(time.RFC3339))
}
