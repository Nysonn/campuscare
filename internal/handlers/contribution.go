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
		  message, is_anonymous, payment_method, amount, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'success')
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

	// Increment campaign amount
	h.DB.Exec(context.Background(),
		`UPDATE campaigns
		 SET current_amount = current_amount + $1
		 WHERE id=$2`,
		req.Amount, campaignID,
	)

	// Send receipt email
	h.Mailer.Send(
		req.DonorEmail,
		"CampusCare Donation Receipt",
		mail.DonationReceiptTemplate(req.DonorName, req.Amount),
	)

	audit.Log(h.DB, uuid.Nil, "DONATION_SUCCESS", "contribution", contributionID, nil)

	c.JSON(http.StatusCreated, gin.H{
		"contribution_id": contributionID,
		"status":          "success",
	})
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
