package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportHandler struct {
	DB *pgxpool.Pool
}

type Report struct {
	ID                 uuid.UUID  `json:"id"`
	ReporterName       *string    `json:"reporter_name"`
	SubjectName        string     `json:"subject_name"`
	SubjectContact     *string    `json:"subject_contact"`
	University         *string    `json:"university"`
	Description        string     `json:"description"`
	Urgency            string     `json:"urgency"`
	Status             string     `json:"status"`
	AdminNotes         *string    `json:"admin_notes"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	WantsFollowup      bool       `json:"wants_followup"`
	FollowupEmail      *string    `json:"followup_email"`
	PoolHelperName     *string    `json:"pool_helper_name"`
	WeeklyReportsCount int        `json:"weekly_reports_count"`
}

type FollowupCase struct {
	ID             uuid.UUID  `json:"id"`
	SubjectName    string     `json:"subject_name"`
	SubjectContact *string    `json:"subject_contact"`
	University     *string    `json:"university"`
	Description    string     `json:"description"`
	Urgency        string     `json:"urgency"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	HelperType     string     `json:"helper_type"`
	WantsFollowup  bool       `json:"wants_followup"`
}

type WelfareReport struct {
	ID             uuid.UUID `json:"id"`
	ReportID       uuid.UUID `json:"report_id"`
	SubjectName    string    `json:"subject_name"`
	HelperName     string    `json:"helper_name"`
	HelperEmail    string    `json:"helper_email"`
	HelperType     string    `json:"helper_type"`
	WeekOf         string    `json:"week_of"`
	WellbeingScore int       `json:"wellbeing_score"`
	Observations   string    `json:"observations"`
	CreatedAt      time.Time `json:"created_at"`
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
		WantsFollowup  bool   `json:"wants_followup"`
		FollowupEmail  string `json:"followup_email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	body.SubjectName = strings.TrimSpace(body.SubjectName)
	body.FollowupEmail = strings.TrimSpace(body.FollowupEmail)
	body.Description = strings.TrimSpace(body.Description)

	if body.SubjectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject_name is required"})
		return
	}
	if body.Description == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "description is required"})
		return
	}
	if body.WantsFollowup && body.FollowupEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "followup_email is required when wants_followup is true"})
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
		`INSERT INTO reports (reporter_name, subject_name, subject_contact, university, description, urgency, wants_followup, followup_email)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		nullableString(body.ReporterName),
		body.SubjectName,
		nullableString(body.SubjectContact),
		nullableString(body.University),
		body.Description,
		body.Urgency,
		body.WantsFollowup,
		nullableString(body.FollowupEmail),
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit report"})
		return
	}

	message := "Report submitted successfully. Our team will review it."
	if body.WantsFollowup {
		message = "Report submitted successfully. Register or log in with the same email to submit weekly follow-up reports offline."
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        message,
		"id":             id,
		"wants_followup": body.WantsFollowup,
	})
}

// MyFollowups handles GET /reports/my-followups (student only).
func (h *ReportHandler) MyFollowups(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	userEmail, err := h.userEmail(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve user email"})
		return
	}

	rows, err := h.DB.Query(c,
		`SELECT r.id, r.subject_name, r.subject_contact, r.university, r.description,
		        r.urgency, r.status, r.created_at, r.updated_at, r.wants_followup,
		        CASE WHEN pa.sponsor_id = $1 THEN 'sponsor_pool' ELSE 'reporter_followup' END AS helper_type
		 FROM reports r
		 LEFT JOIN pool_assignments pa ON pa.report_id = r.id AND pa.ended_at IS NULL
		 WHERE (r.wants_followup = true AND LOWER(COALESCE(r.followup_email, '')) = LOWER($2))
		    OR pa.sponsor_id = $1
		 ORDER BY r.created_at DESC`,
		userID, userEmail,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch follow-up cases"})
		return
	}
	defer rows.Close()

	result := []FollowupCase{}
	for rows.Next() {
		var item FollowupCase
		if err := rows.Scan(
			&item.ID,
			&item.SubjectName,
			&item.SubjectContact,
			&item.University,
			&item.Description,
			&item.Urgency,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.WantsFollowup,
			&item.HelperType,
		); err != nil {
			continue
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, result)
}

// ListPoolReports handles GET /reports/pool (active sponsors only).
func (h *ReportHandler) ListPoolReports(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	if ok, err := h.isActiveSponsor(c, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify sponsor status"})
		return
	} else if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be an active sponsor to access the Help A Student pool"})
		return
	}

	rows, err := h.DB.Query(c,
		`SELECT r.id, r.subject_name, r.subject_contact, r.university, r.description,
		        r.urgency, r.status, r.created_at, r.updated_at, r.wants_followup
		 FROM reports r
		 LEFT JOIN pool_assignments pa ON pa.report_id = r.id AND pa.ended_at IS NULL
		 WHERE r.wants_followup = false
		   AND r.status != 'closed'
		   AND pa.id IS NULL
		 ORDER BY
		   CASE r.urgency
		     WHEN 'critical' THEN 1
		     WHEN 'high' THEN 2
		     WHEN 'medium' THEN 3
		     WHEN 'low' THEN 4
		   END,
		   r.created_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pool reports"})
		return
	}
	defer rows.Close()

	result := []FollowupCase{}
	for rows.Next() {
		var item FollowupCase
		item.HelperType = "sponsor_pool"
		if err := rows.Scan(
			&item.ID,
			&item.SubjectName,
			&item.SubjectContact,
			&item.University,
			&item.Description,
			&item.Urgency,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.WantsFollowup,
		); err != nil {
			continue
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, result)
}

// ClaimPoolReport handles POST /reports/:id/claim (active sponsors only).
func (h *ReportHandler) ClaimPoolReport(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	reportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	if ok, err := h.isActiveSponsor(c, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify sponsor status"})
		return
	} else if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be an active sponsor to help from the pool"})
		return
	}

	var hasActiveAssignment bool
	if err := h.DB.QueryRow(c,
		`SELECT EXISTS(SELECT 1 FROM pool_assignments WHERE sponsor_id = $1 AND ended_at IS NULL)`,
		userID,
	).Scan(&hasActiveAssignment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify pool assignment limit"})
		return
	}
	if hasActiveAssignment {
		c.JSON(http.StatusConflict, gin.H{"error": "You are already helping one student from the Help A Student pool"})
		return
	}

	var subjectName string
	if err := h.DB.QueryRow(c,
		`SELECT subject_name
		 FROM reports
		 WHERE id = $1 AND wants_followup = false AND status != 'closed'`,
		reportID,
	).Scan(&subjectName); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found or unavailable for pool support"})
		return
	}

	_, err = h.DB.Exec(c,
		`INSERT INTO pool_assignments (report_id, sponsor_id) VALUES ($1, $2)`,
		reportID, userID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "This report has already been claimed or you have already claimed another pool case"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to claim report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "You are now helping " + subjectName + " from the Help A Student pool"})
}

// ListWelfareReports handles GET /reports/:id/welfare-reports (assigned helpers only).
func (h *ReportHandler) ListWelfareReports(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	reportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	if _, err := h.helperTypeForReport(c, userID, reportID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not assigned to this report"})
		return
	}

	h.listWelfareReports(c, reportID)
}

// SubmitWelfareReport handles POST /reports/:id/welfare-reports (assigned helpers only).
func (h *ReportHandler) SubmitWelfareReport(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	reportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	if _, err := h.helperTypeForReport(c, userID, reportID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not assigned to this report"})
		return
	}

	var body struct {
		WeekOf         string `json:"week_of"`
		WellbeingScore int    `json:"wellbeing_score"`
		Observations   string `json:"observations"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	body.Observations = strings.TrimSpace(body.Observations)
	if body.Observations == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "observations are required"})
		return
	}
	if body.WellbeingScore < 1 || body.WellbeingScore > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wellbeing_score must be between 1 and 5"})
		return
	}

	weekOf := startOfWeek(time.Now().UTC())
	if strings.TrimSpace(body.WeekOf) != "" {
		parsed, err := time.Parse("2006-01-02", body.WeekOf)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "week_of must be in YYYY-MM-DD format"})
			return
		}
		weekOf = startOfWeek(parsed)
	}

	var welfareID uuid.UUID
	err = h.DB.QueryRow(c,
		`INSERT INTO welfare_reports (report_id, submitted_by, week_of, wellbeing_score, observations)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (report_id, submitted_by, week_of)
		 DO UPDATE SET wellbeing_score = EXCLUDED.wellbeing_score,
		               observations = EXCLUDED.observations,
		               created_at = NOW()
		 RETURNING id`,
		reportID, userID, weekOf.Format("2006-01-02"), body.WellbeingScore, body.Observations,
	).Scan(&welfareID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save welfare report"})
		return
	}

	_, _ = h.DB.Exec(c, `UPDATE reports SET updated_at = NOW() WHERE id = $1`, reportID)

	c.JSON(http.StatusOK, gin.H{"message": "Weekly welfare report saved", "id": welfareID})
}

// AdminListReports handles GET /admin/reports (admin only).
func (h *ReportHandler) AdminListReports(c *gin.Context) {
	statusFilter := c.Query("status")

	rows, err := h.DB.Query(c,
		`SELECT r.id, r.reporter_name, r.subject_name, r.subject_contact, r.university,
			r.description, r.urgency, r.status, r.admin_notes, r.created_at, r.updated_at,
			r.wants_followup, r.followup_email,
			sp.display_name AS pool_helper_name,
			COALESCE((SELECT COUNT(*) FROM welfare_reports wr WHERE wr.report_id = r.id), 0) AS weekly_reports_count
		 FROM reports r
		 LEFT JOIN pool_assignments pa ON pa.report_id = r.id AND pa.ended_at IS NULL
		 LEFT JOIN student_profiles sp ON sp.user_id = pa.sponsor_id
		 WHERE ($1 = '' OR r.status = $1)
		 ORDER BY
		   CASE r.urgency
		     WHEN 'critical' THEN 1
		     WHEN 'high'     THEN 2
		     WHEN 'medium'   THEN 3
		     WHEN 'low'      THEN 4
		   END,
		   r.created_at DESC`,
		statusFilter,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	result := []Report{}
	for rows.Next() {
		var r Report
		if err := rows.Scan(
			&r.ID, &r.ReporterName, &r.SubjectName, &r.SubjectContact,
			&r.University, &r.Description, &r.Urgency, &r.Status,
				&r.AdminNotes, &r.CreatedAt, &r.UpdatedAt,
				&r.WantsFollowup, &r.FollowupEmail, &r.PoolHelperName, &r.WeeklyReportsCount,
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

// AdminListWelfareReports handles GET /admin/reports/welfare (admin only).
func (h *ReportHandler) AdminListWelfareReports(c *gin.Context) {
	h.listWelfareReports(c, uuid.Nil)
}

func (h *ReportHandler) listWelfareReports(c *gin.Context, reportID uuid.UUID) {
	query := `SELECT wr.id, wr.report_id, r.subject_name,
		        COALESCE(sp.display_name, u.email) AS helper_name,
		        u.email,
		        CASE WHEN pa.sponsor_id = wr.submitted_by THEN 'sponsor_pool' ELSE 'reporter_followup' END AS helper_type,
		        wr.week_of, wr.wellbeing_score, wr.observations, wr.created_at
		 FROM welfare_reports wr
		 JOIN reports r ON r.id = wr.report_id
		 JOIN users u ON u.id = wr.submitted_by
		 LEFT JOIN student_profiles sp ON sp.user_id = wr.submitted_by
		 LEFT JOIN pool_assignments pa ON pa.report_id = wr.report_id AND pa.ended_at IS NULL
		 WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR wr.report_id = $1)
		 ORDER BY wr.week_of DESC, wr.created_at DESC`

	rows, err := h.DB.Query(c, query, reportID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch welfare reports"})
		return
	}
	defer rows.Close()

	result := []WelfareReport{}
	for rows.Next() {
		var item WelfareReport
		var weekOf time.Time
		if err := rows.Scan(
			&item.ID,
			&item.ReportID,
			&item.SubjectName,
			&item.HelperName,
			&item.HelperEmail,
			&item.HelperType,
			&weekOf,
			&item.WellbeingScore,
			&item.Observations,
			&item.CreatedAt,
		); err != nil {
			continue
		}
		item.WeekOf = weekOf.Format("2006-01-02")
		result = append(result, item)
	}

	c.JSON(http.StatusOK, result)
}

func (h *ReportHandler) helperTypeForReport(c *gin.Context, userID, reportID uuid.UUID) (string, error) {
	userEmail, err := h.userEmail(c, userID)
	if err != nil {
		return "", err
	}

	var helperType string
	err = h.DB.QueryRow(c,
		`SELECT CASE
		    WHEN pa.sponsor_id = $1 THEN 'sponsor_pool'
		    WHEN r.wants_followup = true AND LOWER(COALESCE(r.followup_email, '')) = LOWER($2) THEN 'reporter_followup'
		    ELSE ''
		  END
		 FROM reports r
		 LEFT JOIN pool_assignments pa ON pa.report_id = r.id AND pa.ended_at IS NULL
		 WHERE r.id = $3`,
		userID, userEmail, reportID,
	).Scan(&helperType)
	if err != nil || helperType == "" {
		return "", errors.New("not assigned")
	}

	return helperType, nil
}

func (h *ReportHandler) isActiveSponsor(c *gin.Context, userID uuid.UUID) (bool, error) {
	var isSponsor bool
	err := h.DB.QueryRow(c,
		`SELECT EXISTS(SELECT 1 FROM sponsor_profiles WHERE user_id = $1 AND is_active = true)`,
		userID,
	).Scan(&isSponsor)
	return isSponsor, err
}

func (h *ReportHandler) userEmail(c *gin.Context, userID uuid.UUID) (string, error) {
	var email string
	err := h.DB.QueryRow(c, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	return email, err
}

func startOfWeek(t time.Time) time.Time {
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -(weekday - 1))
}

// nullableString returns nil for empty strings so they are stored as NULL.
func nullableString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
