package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/Nysonn/campuscare/internal/mail"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminHandler struct {
	DB     *pgxpool.Pool
	Mailer *mail.Mailer
}

func (h *AdminHandler) Dashboard(c *gin.Context) {

	var users, campaigns, bookings int
	var totalRaised int64

	h.DB.QueryRow(c, `SELECT COUNT(*) FROM users`).Scan(&users)
	h.DB.QueryRow(c, `SELECT COUNT(*) FROM campaigns WHERE deleted_at IS NULL`).Scan(&campaigns)
	h.DB.QueryRow(c, `SELECT COUNT(*) FROM bookings`).Scan(&bookings)
	h.DB.QueryRow(c, `SELECT COALESCE(SUM(current_amount),0) FROM campaigns`).Scan(&totalRaised)

	c.JSON(http.StatusOK, gin.H{
		"users":        users,
		"campaigns":    campaigns,
		"bookings":     bookings,
		"total_raised": totalRaised,
	})
}

func (h *AdminHandler) ListUsers(c *gin.Context) {

	role := c.Query("role")
	status := c.Query("status")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit := 20
	offset := (page - 1) * limit

	rows, err := h.DB.Query(c,
		`SELECT u.id, u.email, u.role::text, u.status, u.created_at,
		        COALESCE(cp.full_name, sp.display_name, '') AS full_name
		 FROM users u
		 LEFT JOIN student_profiles sp ON sp.user_id = u.id
		 LEFT JOIN counselor_profiles cp ON cp.user_id = u.id
		 WHERE u.deleted_at IS NULL
		   AND ($1='' OR u.role::text=$1)
		   AND ($2='' OR u.status=$2)
		 ORDER BY u.created_at DESC
		 LIMIT $3 OFFSET $4`,
		role, status, limit, offset,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	result := []gin.H{}

	for rows.Next() {
		var id uuid.UUID
		var email, role, status, fullName string
		var createdAt time.Time

		rows.Scan(&id, &email, &role, &status, &createdAt, &fullName)

		result = append(result, gin.H{
			"id":         id,
			"full_name":  fullName,
			"email":      email,
			"role":       role,
			"status":     status,
			"created_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, result)
}

func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {

	idParam := c.Param("id")
	userID, _ := uuid.Parse(idParam)

	var body struct {
		Status string `json:"status"`
	}
	c.BindJSON(&body)

	if body.Status != "active" && body.Status != "suspended" {
		c.JSON(400, gin.H{"error": "Invalid status"})
		return
	}

	_, err := h.DB.Exec(c,
		`UPDATE users SET status=$1 WHERE id=$2`,
		body.Status, userID,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": "Update failed"})
		return
	}

	c.JSON(200, gin.H{"message": "User updated"})
}

func (h *AdminHandler) ListCampaigns(c *gin.Context) {

	status := c.Query("status")

	rows, err := h.DB.Query(c,
		`SELECT id,title,status,current_amount,created_at
		 FROM campaigns
		 WHERE deleted_at IS NULL
		   AND ($1='' OR status=$1)
		 ORDER BY created_at DESC`,
		status,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed"})
		return
	}
	defer rows.Close()

	var list []gin.H

	for rows.Next() {
		var id uuid.UUID
		var title, status string
		var amount int64
		var createdAt time.Time

		rows.Scan(&id, &title, &status, &amount, &createdAt)

		list = append(list, gin.H{
			"id":             id,
			"title":          title,
			"status":         status,
			"current_amount": amount,
			"created_at":     createdAt,
		})
	}

	c.JSON(200, list)
}

func (h *AdminHandler) DeleteCampaign(c *gin.Context) {

	idParam := c.Param("id")
	campaignID, _ := uuid.Parse(idParam)

	_, err := h.DB.Exec(c,
		`UPDATE campaigns SET deleted_at=now() WHERE id=$1`,
		campaignID,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed"})
		return
	}

	c.JSON(200, gin.H{"message": "Campaign removed"})
}

func (h *AdminHandler) ListBookings(c *gin.Context) {

	rows, _ := h.DB.Query(c,
		`SELECT b.id, b.student_id, b.counselor_id, b.start_time, b.end_time, b.status,
		        COALESCE(sp.display_name, '') AS student_name,
		        COALESCE(cp.full_name, '') AS counselor_name
		 FROM bookings b
		 LEFT JOIN student_profiles sp ON sp.user_id = b.student_id
		 LEFT JOIN counselor_profiles cp ON cp.user_id = b.counselor_id
		 ORDER BY b.start_time DESC`,
	)

	var list []gin.H

	for rows.Next() {
		var id, student, counselor uuid.UUID
		var status, studentName, counselorName string
		var startTime, endTime time.Time

		rows.Scan(&id, &student, &counselor, &startTime, &endTime, &status, &studentName, &counselorName)

		list = append(list, gin.H{
			"id":             id,
			"student_id":     student,
			"student_name":   studentName,
			"counselor_id":   counselor,
			"counselor_name": counselorName,
			"start_time":     startTime,
			"end_time":       endTime,
			"status":         status,
		})
	}

	c.JSON(200, list)
}

func (h *AdminHandler) ListContributions(c *gin.Context) {

	rows, _ := h.DB.Query(c,
		`SELECT id, campaign_id, donor_name, donor_email, amount, status, created_at
		 FROM contributions
		 ORDER BY created_at DESC`,
	)

	var list []gin.H

	for rows.Next() {
		var id, campaignID uuid.UUID
		var donorName, donorEmail, status string
		var amount int64
		var createdAt time.Time

		rows.Scan(&id, &campaignID, &donorName, &donorEmail, &amount, &status, &createdAt)

		list = append(list, gin.H{
			"id":          id,
			"campaign_id": campaignID,
			"donor_name":  donorName,
			"donor_email": donorEmail,
			"amount":      amount,
			"status":      status,
			"created_at":  createdAt,
		})
	}

	if list == nil {
		list = []gin.H{}
	}

	c.JSON(200, list)
}

// ListPendingCounselors — GET /admin/counselors?status=pending|approved|rejected|all
func (h *AdminHandler) ListPendingCounselors(c *gin.Context) {
	statusFilter := c.DefaultQuery("status", "pending")

	rows, err := h.DB.Query(context.Background(),
		`SELECT u.id, u.email, u.created_at,
		        cp.full_name, cp.specialization, cp.bio, cp.phone,
		        COALESCE(cp.avatar_url,''), COALESCE(cp.location,''),
		        cp.age, COALESCE(cp.years_of_experience,''),
		        COALESCE(cp.licence_url,''), cp.verification_status
		 FROM users u
		 JOIN counselor_profiles cp ON cp.user_id = u.id
		 WHERE u.role = 'counselor'
		   AND u.deleted_at IS NULL
		   AND ($1 = 'all' OR cp.verification_status = $1)
		 ORDER BY u.created_at DESC`,
		statusFilter,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch counselors"})
		return
	}
	defer rows.Close()

	var list []gin.H
	for rows.Next() {
		var id uuid.UUID
		var email, fullName, specialization, bio, phone string
		var avatarURL, location, yearsOfExperience, licenceURL, verificationStatus string
		var age *int
		var createdAt time.Time

		rows.Scan(&id, &email, &createdAt,
			&fullName, &specialization, &bio, &phone,
			&avatarURL, &location, &age, &yearsOfExperience,
			&licenceURL, &verificationStatus)

		list = append(list, gin.H{
			"id":                  id,
			"email":               email,
			"full_name":           fullName,
			"specialization":      specialization,
			"bio":                 bio,
			"phone":               phone,
			"avatar_url":          avatarURL,
			"location":            location,
			"age":                 age,
			"years_of_experience": yearsOfExperience,
			"licence_url":         licenceURL,
			"verification_status": verificationStatus,
			"created_at":          createdAt,
		})
	}

	if list == nil {
		list = []gin.H{}
	}

	c.JSON(http.StatusOK, list)
}

// ApproveCounselor — PUT /admin/counselors/:id/verify
func (h *AdminHandler) ApproveCounselor(c *gin.Context) {
	counselorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid counselor ID"})
		return
	}

	var body struct {
		Status string `json:"status"` // "approved" or "rejected"
	}
	if err := c.BindJSON(&body); err != nil || (body.Status != "approved" && body.Status != "rejected") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'approved' or 'rejected'"})
		return
	}

	_, err = h.DB.Exec(context.Background(),
		`UPDATE counselor_profiles SET verification_status=$1 WHERE user_id=$2`,
		body.Status, counselorID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}

	// Send approval email only — no email on rejection per spec.
	if body.Status == "approved" {
		go h.notifyCounselorApproved(counselorID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Counselor verification status updated"})
}

func (h *AdminHandler) notifyCounselorApproved(counselorID uuid.UUID) {
	var email, fullName string
	err := h.DB.QueryRow(context.Background(),
		`SELECT u.email, cp.full_name
		 FROM users u
		 JOIN counselor_profiles cp ON cp.user_id = u.id
		 WHERE u.id = $1`,
		counselorID,
	).Scan(&email, &fullName)
	if err != nil {
		return
	}

	h.Mailer.SendAsync(
		email,
		"Your CampusCare account has been approved!",
		mail.CounselorApprovedTemplate(fullName),
	)
}

// ListGeneralPoolDonations — GET /admin/general-pool
func (h *AdminHandler) ListGeneralPoolDonations(c *gin.Context) {
	rows, err := h.DB.Query(c,
		`SELECT id, donor_name, donor_email, amount, payment_method, is_anonymous, status, created_at
		 FROM general_pool_donations
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed"})
		return
	}
	defer rows.Close()

	type Donation struct {
		ID            uuid.UUID `json:"id"`
		DonorName     string    `json:"donor_name"`
		DonorEmail    string    `json:"donor_email"`
		Amount        int64     `json:"amount"`
		PaymentMethod string    `json:"payment_method"`
		IsAnonymous   bool      `json:"is_anonymous"`
		Status        string    `json:"status"`
		CreatedAt     time.Time `json:"created_at"`
	}

	var donations []Donation
	for rows.Next() {
		var d Donation
		rows.Scan(&d.ID, &d.DonorName, &d.DonorEmail, &d.Amount, &d.PaymentMethod, &d.IsAnonymous, &d.Status, &d.CreatedAt)
		donations = append(donations, d)
	}
	if donations == nil {
		donations = []Donation{}
	}

	c.JSON(200, donations)
}
