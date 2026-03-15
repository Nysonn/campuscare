package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/Nysonn/campuscare/internal/audit"
	calendarPkg "github.com/Nysonn/campuscare/internal/calendar"
	"github.com/Nysonn/campuscare/internal/mail"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingHandler struct {
	DB     *pgxpool.Pool
	Mailer *mail.Mailer
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

	// If accepted → create Google Calendar event + notify student
	if body.Status == "accepted" {
		go h.createCalendarEvent(bookingID)
		go h.notifyStudentBookingStatus(bookingID, "accepted")
	}

	if body.Status == "declined" {
		go h.notifyStudentBookingStatus(bookingID, "declined")
	}

	audit.Log(h.DB, counselorID, "UPDATE_BOOKING", "booking", bookingID, body)

	c.JSON(http.StatusOK, gin.H{"message": "Booking updated"})
}

func (h *BookingHandler) notifyStudentBookingStatus(bookingID uuid.UUID, status string) {
	var studentEmail, studentName, counselorName, sessionType string
	var startTime, endTime time.Time

	err := h.DB.QueryRow(context.Background(),
		`SELECT u.email, sp.display_name, cp.full_name, b.type::text, b.start_time, b.end_time
		 FROM bookings b
		 JOIN users u ON u.id = b.student_id
		 JOIN student_profiles sp ON sp.user_id = b.student_id
		 JOIN counselor_profiles cp ON cp.user_id = b.counselor_id
		 WHERE b.id = $1`,
		bookingID,
	).Scan(&studentEmail, &studentName, &counselorName, &sessionType, &startTime, &endTime)
	if err != nil {
		return
	}

	start := startTime.Format("02 Jan 2006 · 15:04")
	end := endTime.Format("15:04")

	var subject, body string
	if status == "accepted" {
		subject = "Your Counselling Session Has Been Confirmed"
		body = mail.BookingAcceptedTemplate(studentName, counselorName, sessionType, start, end)
	} else {
		subject = "Your Counselling Session Request Was Declined"
		body = mail.BookingDeclinedTemplate(studentName, counselorName, start)
	}

	h.Mailer.Send(studentEmail, subject, body)
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

func (h *BookingHandler) ListCounselors(c *gin.Context) {

	rows, err := h.DB.Query(c,
		`SELECT u.id, cp.full_name, cp.specialization, cp.bio
		 FROM users u
		 JOIN counselor_profiles cp ON cp.user_id = u.id
		 WHERE u.role = 'counselor'
		   AND u.deleted_at IS NULL
		 ORDER BY cp.full_name ASC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch counselors"})
		return
	}
	defer rows.Close()

	var list []gin.H

	for rows.Next() {
		var id uuid.UUID
		var fullName, specialization, bio string

		rows.Scan(&id, &fullName, &specialization, &bio)

		list = append(list, gin.H{
			"id":             id,
			"full_name":      fullName,
			"specialization": specialization,
			"bio":            bio,
		})
	}

	if list == nil {
		list = []gin.H{}
	}

	c.JSON(http.StatusOK, list)
}

func (h *BookingHandler) MyBookings(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)

	rows, err := h.DB.Query(c,
		`SELECT b.id, b.counselor_id, cp.full_name AS counselor_name,
		        b.type::text, b.start_time, b.end_time, b.location, b.status::text
		 FROM bookings b
		 JOIN counselor_profiles cp ON cp.user_id = b.counselor_id
		 WHERE b.student_id = $1
		   AND b.deleted_at IS NULL
		 ORDER BY b.start_time DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fetch failed"})
		return
	}
	defer rows.Close()

	var list []gin.H

	for rows.Next() {
		var id, counselorID uuid.UUID
		var counselorName, sessionType, location, status string
		var startTime, endTime time.Time

		rows.Scan(&id, &counselorID, &counselorName, &sessionType, &startTime, &endTime, &location, &status)

		list = append(list, gin.H{
			"id":             id,
			"counselor_id":   counselorID,
			"counselor_name": counselorName,
			"type":           sessionType,
			"start_time":     startTime,
			"end_time":       endTime,
			"location":       location,
			"status":         status,
		})
	}

	if list == nil {
		list = []gin.H{}
	}

	c.JSON(http.StatusOK, list)
}

func (h *BookingHandler) CounselorBookings(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)
	status := c.Query("status")

	rows, err := h.DB.Query(c,
		`SELECT b.id, b.student_id, sp.display_name AS student_name,
		        b.type::text, b.start_time, b.end_time, b.location, b.status::text
		 FROM bookings b
		 JOIN student_profiles sp ON sp.user_id = b.student_id
		 WHERE b.counselor_id = $1
		   AND ($2 = '' OR b.status::text = $2)
		   AND b.deleted_at IS NULL
		 ORDER BY b.start_time ASC`,
		userID, status,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fetch failed"})
		return
	}
	defer rows.Close()

	var list []gin.H

	for rows.Next() {
		var id, studentID uuid.UUID
		var studentName, sessionType, location, bookingStatus string
		var startTime, endTime time.Time

		rows.Scan(&id, &studentID, &studentName, &sessionType, &startTime, &endTime, &location, &bookingStatus)

		list = append(list, gin.H{
			"id":           id,
			"student_id":   studentID,
			"student_name": studentName,
			"type":         sessionType,
			"start_time":   startTime,
			"end_time":     endTime,
			"location":     location,
			"status":       bookingStatus,
		})
	}

	if list == nil {
		list = []gin.H{}
	}

	c.JSON(http.StatusOK, list)
}
