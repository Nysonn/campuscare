package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TestimonialHandler struct {
	DB *pgxpool.Pool
}

type Testimonial struct {
	ID          uuid.UUID `json:"id"`
	StudentID   uuid.UUID `json:"student_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	University  string    `json:"university"`
	Content     string    `json:"content"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SubmitTestimonialRequest struct {
	Content string `json:"content"`
}

// cooldownDays is the minimum gap between testimonial submissions.
const cooldownDays = 90

// ListTestimonials — public, returns approved testimonials.
func (h *TestimonialHandler) ListTestimonials(c *gin.Context) {
	rows, err := h.DB.Query(context.Background(),
		`SELECT id, student_id, display_name, avatar_url, university, content, status, created_at, updated_at
		 FROM testimonials
		 WHERE status = 'approved'
		 ORDER BY updated_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch testimonials"})
		return
	}
	defer rows.Close()

	var testimonials []Testimonial
	for rows.Next() {
		var t Testimonial
		if err := rows.Scan(&t.ID, &t.StudentID, &t.DisplayName, &t.AvatarURL,
			&t.University, &t.Content, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		testimonials = append(testimonials, t)
	}
	if testimonials == nil {
		testimonials = []Testimonial{}
	}
	c.JSON(http.StatusOK, testimonials)
}

// MyTestimonials — student, returns all their testimonials plus cooldown info.
func (h *TestimonialHandler) MyTestimonial(c *gin.Context) {
	studentID := c.MustGet("user_id").(uuid.UUID)

	rows, err := h.DB.Query(context.Background(),
		`SELECT id, student_id, display_name, avatar_url, university, content, status, created_at, updated_at
		 FROM testimonials
		 WHERE student_id = $1
		 ORDER BY created_at DESC`, studentID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch testimonials"})
		return
	}
	defer rows.Close()

	var testimonials []Testimonial
	for rows.Next() {
		var t Testimonial
		if err := rows.Scan(&t.ID, &t.StudentID, &t.DisplayName, &t.AvatarURL,
			&t.University, &t.Content, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		testimonials = append(testimonials, t)
	}
	if testimonials == nil {
		testimonials = []Testimonial{}
	}

	canSubmit := true
	var nextAllowedAt *time.Time
	if len(testimonials) > 0 {
		next := testimonials[0].CreatedAt.Add(cooldownDays * 24 * time.Hour)
		if time.Now().Before(next) {
			canSubmit = false
			nextAllowedAt = &next
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"testimonials":    testimonials,
		"can_submit":      canSubmit,
		"next_allowed_at": nextAllowedAt,
	})
}

// SubmitTestimonial — student, inserts a new testimonial (max one per 90 days).
func (h *TestimonialHandler) SubmitTestimonial(c *gin.Context) {
	studentID := c.MustGet("user_id").(uuid.UUID)

	var req SubmitTestimonialRequest
	if err := c.ShouldBindJSON(&req); err != nil || len([]rune(req.Content)) < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content must be at least 10 characters"})
		return
	}

	// Enforce 90-day cooldown.
	var lastCreatedAt time.Time
	err := h.DB.QueryRow(context.Background(),
		`SELECT created_at FROM testimonials WHERE student_id = $1 ORDER BY created_at DESC LIMIT 1`,
		studentID,
	).Scan(&lastCreatedAt)
	if err == nil {
		nextAllowed := lastCreatedAt.Add(cooldownDays * 24 * time.Hour)
		if time.Now().Before(nextAllowed) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":           "You can submit a new testimonial on " + nextAllowed.Format("January 2, 2006"),
				"next_allowed_at": nextAllowed.Format(time.RFC3339),
			})
			return
		}
	}

	// Look up student profile for display_name, avatar_url, university.
	var displayName, avatarURL, university string
	h.DB.QueryRow(context.Background(),
		`SELECT display_name, COALESCE(avatar_url,''), COALESCE(university,'')
		 FROM student_profiles WHERE user_id = $1`, studentID,
	).Scan(&displayName, &avatarURL, &university)

	if displayName == "" {
		displayName = "Anonymous Student"
	}

	var t Testimonial
	err = h.DB.QueryRow(context.Background(),
		`INSERT INTO testimonials (student_id, display_name, avatar_url, university, content, status)
		 VALUES ($1, $2, $3, $4, $5, 'pending')
		 RETURNING id, student_id, display_name, avatar_url, university, content, status, created_at, updated_at`,
		studentID, displayName, avatarURL, university, req.Content,
	).Scan(&t.ID, &t.StudentID, &t.DisplayName, &t.AvatarURL,
		&t.University, &t.Content, &t.Status, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save testimonial"})
		return
	}

	c.JSON(http.StatusOK, t)
}

// AdminListTestimonials — admin, all testimonials with optional status filter.
func (h *TestimonialHandler) AdminListTestimonials(c *gin.Context) {
	status := c.Query("status") // 'pending', 'approved', 'rejected', or '' for all

	query := `SELECT id, student_id, display_name, avatar_url, university, content, status, created_at, updated_at
	          FROM testimonials`
	args := []any{}
	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := h.DB.Query(context.Background(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch testimonials"})
		return
	}
	defer rows.Close()

	var testimonials []Testimonial
	for rows.Next() {
		var t Testimonial
		if err := rows.Scan(&t.ID, &t.StudentID, &t.DisplayName, &t.AvatarURL,
			&t.University, &t.Content, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		testimonials = append(testimonials, t)
	}
	if testimonials == nil {
		testimonials = []Testimonial{}
	}
	c.JSON(http.StatusOK, testimonials)
}

// AdminUpdateTestimonialStatus — admin, approve or reject.
func (h *TestimonialHandler) AdminUpdateTestimonialStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid testimonial ID"})
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil ||
		(body.Status != "approved" && body.Status != "rejected") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'approved' or 'rejected'"})
		return
	}

	cmd, err := h.DB.Exec(context.Background(),
		`UPDATE testimonials SET status = $1, updated_at = NOW() WHERE id = $2`,
		body.Status, id)
	if err != nil || cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Testimonial not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}

// AdminDeleteTestimonial — admin, hard delete.
func (h *TestimonialHandler) AdminDeleteTestimonial(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid testimonial ID"})
		return
	}

	cmd, err := h.DB.Exec(context.Background(),
		`DELETE FROM testimonials WHERE id = $1`, id)
	if err != nil || cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Testimonial not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Testimonial deleted"})
}
