package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportHandler struct {
	DB *pgxpool.Pool
}

// SubmitReport handles POST /reports (public — no auth required).
func (h *ReportHandler) SubmitReport(c *gin.Context) {
	var body struct {
		ReporterName   string `json:"reporter_name"`
		SubjectName    string `json:"subject_name"`
		SubjectContact string `json:"subject_contact"`
		University     string `json:"university"`
		Description    string `json:"description"`
		Urgency        string `json:"urgency"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	body.SubjectName = strings.TrimSpace(body.SubjectName)
	body.Description = strings.TrimSpace(body.Description)

	if body.SubjectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_name is required"})
		return
	}
	if body.Description == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "description is required"})
		return
	}

	validUrgencies := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if body.Urgency == "" {
		body.Urgency = "medium"
	}
	if !validUrgencies[body.Urgency] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "urgency must be one of: low, medium, high, critical"})
		return
	}

	var id uuid.UUID
	err := h.DB.QueryRow(c,
		`INSERT INTO reports (reporter_name, subject_name, subject_contact, university, description, urgency)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		nullableString(body.ReporterName),
		body.SubjectName,
		nullableString(body.SubjectContact),
		nullableString(body.University),
		body.Description,
		body.Urgency,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit report"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Report submitted successfully. Our team will follow up.",
		"id":      id,
	})
}

// AdminListReports handles GET /admin/reports (admin only).
func (h *ReportHandler) AdminListReports(c *gin.Context) {
	statusFilter := c.Query("status")

	rows, err := h.DB.Query(c,
		`SELECT id, reporter_name, subject_name, subject_contact, university,
		        description, urgency, status, admin_notes, created_at, updated_at
		 FROM reports
		 WHERE ($1 = '' OR status = $1)
		 ORDER BY
		   CASE urgency
		     WHEN 'critical' THEN 1
		     WHEN 'high'     THEN 2
		     WHEN 'medium'   THEN 3
		     WHEN 'low'      THEN 4
		   END,
		   created_at DESC`,
		statusFilter,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Report struct {
		ID             uuid.UUID  `json:"id"`
		ReporterName   *string    `json:"reporter_name"`
		SubjectName    string     `json:"subject_name"`
		SubjectContact *string    `json:"subject_contact"`
		University     *string    `json:"university"`
		Description    string     `json:"description"`
		Urgency        string     `json:"urgency"`
		Status         string     `json:"status"`
		AdminNotes     *string    `json:"admin_notes"`
		CreatedAt      time.Time  `json:"created_at"`
		UpdatedAt      time.Time  `json:"updated_at"`
	}

	result := []Report{}
	for rows.Next() {
		var r Report
		if err := rows.Scan(
			&r.ID, &r.ReporterName, &r.SubjectName, &r.SubjectContact,
			&r.University, &r.Description, &r.Urgency, &r.Status,
			&r.AdminNotes, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			continue
		}
		result = append(result, r)
	}

	c.JSON(http.StatusOK, result)
}

// AdminUpdateReport handles PUT /admin/reports/:id (admin only).
func (h *ReportHandler) AdminUpdateReport(c *gin.Context) {
	idParam := c.Param("id")
	reportID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	var body struct {
		Status     string `json:"status"`
		AdminNotes string `json:"admin_notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	validStatuses := map[string]bool{"pending": true, "reviewed": true, "actioned": true, "closed": true}
	if body.Status != "" && !validStatuses[body.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	tag, err := h.DB.Exec(c,
		`UPDATE reports
		 SET status      = CASE WHEN $2 != '' THEN $2 ELSE status END,
		     admin_notes = CASE WHEN $3 != '' THEN $3 ELSE admin_notes END,
		     updated_at  = NOW()
		 WHERE id = $1`,
		reportID, body.Status, body.AdminNotes,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "report updated"})
}

// AdminDeleteReport handles DELETE /admin/reports/:id (admin only).
func (h *ReportHandler) AdminDeleteReport(c *gin.Context) {
	idParam := c.Param("id")
	reportID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	tag, err := h.DB.Exec(c, `DELETE FROM reports WHERE id = $1`, reportID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "report deleted"})
}

// nullableString returns nil for empty strings so they are stored as NULL.
func nullableString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
