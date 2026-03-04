package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"

	"github.com/Nysonn/campuscare/internal/audit"
	"github.com/Nysonn/campuscare/internal/mail"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ContributionHandler struct {
	DB     *pgxpool.Pool
	Mailer *mail.Mailer
}

type CreateContributionRequest struct {
	CampaignID    string `json:"campaign_id"`
	DonorName     string `json:"donor_name"`
	DonorEmail    string `json:"donor_email"`
	DonorPhone    string `json:"donor_phone"`
	Message       string `json:"message"`
	IsAnonymous   bool   `json:"is_anonymous"`
	PaymentMethod string `json:"payment_method"`
	Amount        int64  `json:"amount"`
}

func (h *ContributionHandler) Create(c *gin.Context) {

	var req CreateContributionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	campaignID, _ := uuid.Parse(req.CampaignID)

	validPaymentMethods := map[string]bool{"mtn_momo": true, "airtel_money": true, "visa": true}
	if !validPaymentMethods[req.PaymentMethod] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_method must be one of: mtn_momo, airtel_money, visa"})
		return
	}

	var contributionID uuid.UUID
	err := h.DB.QueryRow(context.Background(),
		`INSERT INTO contributions
		 (campaign_id, donor_name, donor_email, donor_phone,
		  message, is_anonymous, payment_method, amount)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id`,
		campaignID,
		req.DonorName,
		req.DonorEmail,
		req.DonorPhone,
		req.Message,
		req.IsAnonymous,
		req.PaymentMethod,
		req.Amount,
	).Scan(&contributionID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Contribution failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"contribution_id": contributionID,
		"message":         "Pending payment simulation",
	})
}

func (h *ContributionHandler) Simulate(c *gin.Context) {

	idParam := c.Param("id")
	contributionID, _ := uuid.Parse(idParam)

	var body struct {
		Success bool `json:"success"`
	}
	c.BindJSON(&body)

	status := "failed"
	if body.Success {
		status = "success"
	}

	_, err := h.DB.Exec(context.Background(),
		`UPDATE contributions
		 SET status=$1, updated_at=now()
		 WHERE id=$2`,
		status, contributionID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Simulation failed"})
		return
	}

	if body.Success {

		var amount int64
		var campaignID uuid.UUID
		var donorEmail string
		var donorName string

		h.DB.QueryRow(context.Background(),
			`SELECT amount,campaign_id,donor_email,donor_name
			 FROM contributions WHERE id=$1`,
			contributionID,
		).Scan(&amount, &campaignID, &donorEmail, &donorName)

		// Increment campaign amount
		h.DB.Exec(context.Background(),
			`UPDATE campaigns
			 SET current_amount = current_amount + $1
			 WHERE id=$2`,
			amount, campaignID,
		)

		// Send Emails
		h.Mailer.Send(
			donorEmail,
			"CampusCare Donation Receipt",
			mail.DonationReceiptTemplate(donorName, amount),
		)

		audit.Log(h.DB, uuid.Nil, "DONATION_SUCCESS", "contribution", contributionID, nil)
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

func (h *AdminHandler) ExportContributions(c *gin.Context) {

	rows, _ := h.DB.Query(context.Background(),
		`SELECT donor_name, donor_email, amount, status
		 FROM contributions`)

	c.Header("Content-Disposition", "attachment; filename=contributions.csv")
	c.Header("Content-Type", "text/csv")

	writer := csv.NewWriter(c.Writer)
	writer.Write([]string{"Name", "Email", "Amount", "Status"})

	for rows.Next() {
		var n, e, s string
		var a int64
		rows.Scan(&n, &e, &a, &s)
		writer.Write([]string{n, e, fmt.Sprint(a), s})
	}

	writer.Flush()
}

func (h *AdminHandler) AnonymizeUser(c *gin.Context) {

	idParam := c.Param("id")
	userID, _ := uuid.Parse(idParam)

	_, err := h.DB.Exec(context.Background(),
		`UPDATE users
		 SET name='ANONYMIZED',
		     email=concat('anon_',id,'@deleted.local')
		 WHERE id=$1`,
		userID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User anonymized"})
}
