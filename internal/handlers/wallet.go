package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WalletHandler struct {
	DB *pgxpool.Pool
}

// GetPoolBalance — GET /admin/wallet/balance
// Returns total donated, total disbursed to campaigns, total withdrawn, and current balance.
func (h *WalletHandler) GetPoolBalance(c *gin.Context) {
	var totalDonated, totalDisbursed, totalWithdrawn int64

	h.DB.QueryRow(c,
		`SELECT COALESCE(SUM(amount), 0) FROM general_pool_donations WHERE status = 'success'`,
	).Scan(&totalDonated)

	h.DB.QueryRow(c,
		`SELECT COALESCE(SUM(amount), 0) FROM pool_disbursements`,
	).Scan(&totalDisbursed)

	h.DB.QueryRow(c,
		`SELECT COALESCE(SUM(amount), 0) FROM pool_withdrawals`,
	).Scan(&totalWithdrawn)

	balance := totalDonated - totalDisbursed - totalWithdrawn

	c.JSON(http.StatusOK, gin.H{
		"total_donated":    totalDonated,
		"total_disbursed":  totalDisbursed,
		"total_withdrawn":  totalWithdrawn,
		"balance":          balance,
	})
}

// DisburseToC ampaign — POST /admin/wallet/disburse
// Transfers an amount from the general pool into a specific campaign's current_amount.
func (h *WalletHandler) DisburseToCampaign(c *gin.Context) {
	var req struct {
		CampaignID string `json:"campaign_id"`
		Amount     int64  `json:"amount"`
		Note       string `json:"note"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than zero"})
		return
	}
	campaignID, err := uuid.Parse(req.CampaignID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign ID"})
		return
	}

	// Check campaign exists, is approved, and has not yet hit its funding target.
	var count int
	h.DB.QueryRow(c,
		`SELECT COUNT(*) FROM campaigns
		 WHERE id = $1 AND status::text = 'approved' AND deleted_at IS NULL
		   AND current_amount < target_amount`,
		campaignID,
	).Scan(&count)
	if count == 0 {
		// Distinguish between "not found" and "already fully funded".
		var exists int
		h.DB.QueryRow(c,
			`SELECT COUNT(*) FROM campaigns WHERE id = $1 AND status::text = 'approved' AND deleted_at IS NULL`,
			campaignID,
		).Scan(&exists)
		if exists == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Approved campaign not found"})
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "This campaign has already reached its funding target and cannot receive further disbursements"})
		}
		return
	}

	// Check sufficient balance.
	var totalDonated, totalDisbursed, totalWithdrawn int64
	h.DB.QueryRow(c, `SELECT COALESCE(SUM(amount), 0) FROM general_pool_donations WHERE status = 'success'`).Scan(&totalDonated)
	h.DB.QueryRow(c, `SELECT COALESCE(SUM(amount), 0) FROM pool_disbursements`).Scan(&totalDisbursed)
	h.DB.QueryRow(c, `SELECT COALESCE(SUM(amount), 0) FROM pool_withdrawals`).Scan(&totalWithdrawn)
	balance := totalDonated - totalDisbursed - totalWithdrawn

	if req.Amount > balance {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient pool balance"})
		return
	}

	// Record disbursement and increase campaign's current_amount.
	tx, err := h.DB.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c)

	var disbID uuid.UUID
	if err := tx.QueryRow(c,
		`INSERT INTO pool_disbursements (campaign_id, amount, note) VALUES ($1, $2, $3) RETURNING id`,
		campaignID, req.Amount, req.Note,
	).Scan(&disbID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record disbursement"})
		return
	}

	if _, err := tx.Exec(c,
		`UPDATE campaigns SET current_amount = current_amount + $1 WHERE id = $2`,
		req.Amount, campaignID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update campaign"})
		return
	}

	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Funds disbursed successfully",
		"disbursement_id": disbID,
	})
}

// WithdrawFromPool — POST /admin/wallet/withdraw
// Records a withdrawal from the general pool to a bank or mobile money account.
func (h *WalletHandler) WithdrawFromPool(c *gin.Context) {
	var req struct {
		Amount          int64  `json:"amount"`
		DestinationType string `json:"destination_type"` // "bank" | "mtn_momo" | "airtel_money"
		DestinationName string `json:"destination_name"`
		AccountNumber   string `json:"account_number"`
		Note            string `json:"note"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than zero"})
		return
	}
	if req.DestinationType == "" || req.DestinationName == "" || req.AccountNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "destination_type, destination_name, and account_number are required"})
		return
	}

	// Check sufficient balance.
	var totalDonated, totalDisbursed, totalWithdrawn int64
	h.DB.QueryRow(c, `SELECT COALESCE(SUM(amount), 0) FROM general_pool_donations WHERE status = 'success'`).Scan(&totalDonated)
	h.DB.QueryRow(c, `SELECT COALESCE(SUM(amount), 0) FROM pool_disbursements`).Scan(&totalDisbursed)
	h.DB.QueryRow(c, `SELECT COALESCE(SUM(amount), 0) FROM pool_withdrawals`).Scan(&totalWithdrawn)
	balance := totalDonated - totalDisbursed - totalWithdrawn

	if req.Amount > balance {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient pool balance"})
		return
	}

	var withdrawID uuid.UUID
	if err := h.DB.QueryRow(c,
		`INSERT INTO pool_withdrawals (amount, destination_type, destination_name, account_number, note)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		req.Amount, req.DestinationType, req.DestinationName, req.AccountNumber, req.Note,
	).Scan(&withdrawID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record withdrawal"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Withdrawal processed successfully",
		"withdrawal_id": withdrawID,
	})
}

// ListDisbursements — GET /admin/wallet/disbursements
func (h *WalletHandler) ListDisbursements(c *gin.Context) {
	rows, err := h.DB.Query(c, `
		SELECT pd.id, pd.campaign_id, c.title AS campaign_title,
		       pd.amount, pd.note, pd.created_at
		FROM pool_disbursements pd
		JOIN campaigns c ON c.id = pd.campaign_id
		ORDER BY pd.created_at DESC
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load disbursements"})
		return
	}
	defer rows.Close()

	type Row struct {
		ID             uuid.UUID `json:"id"`
		CampaignID     uuid.UUID `json:"campaign_id"`
		CampaignTitle  string    `json:"campaign_title"`
		Amount         int64     `json:"amount"`
		Note           string    `json:"note"`
		CreatedAt      time.Time `json:"created_at"`
	}

	result := []Row{}
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.CampaignID, &r.CampaignTitle, &r.Amount, &r.Note, &r.CreatedAt); err != nil {
			continue
		}
		result = append(result, r)
	}
	c.JSON(http.StatusOK, gin.H{"disbursements": result})
}

// ListWithdrawals — GET /admin/wallet/withdrawals
func (h *WalletHandler) ListWithdrawals(c *gin.Context) {
	rows, err := h.DB.Query(c, `
		SELECT id, amount, destination_type, destination_name, account_number, note, created_at
		FROM pool_withdrawals
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load withdrawals"})
		return
	}
	defer rows.Close()

	type Row struct {
		ID              uuid.UUID `json:"id"`
		Amount          int64     `json:"amount"`
		DestinationType string    `json:"destination_type"`
		DestinationName string    `json:"destination_name"`
		AccountNumber   string    `json:"account_number"`
		Note            string    `json:"note"`
		CreatedAt       time.Time `json:"created_at"`
	}

	result := []Row{}
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Amount, &r.DestinationType, &r.DestinationName, &r.AccountNumber, &r.Note, &r.CreatedAt); err != nil {
			continue
		}
		result = append(result, r)
	}
	c.JSON(http.StatusOK, gin.H{"withdrawals": result})
}

// ListApprovedCampaigns — GET /admin/wallet/campaigns
// Returns approved campaigns for the disburse dropdown.
func (h *WalletHandler) ListApprovedCampaigns(c *gin.Context) {
	rows, err := h.DB.Query(c, `
		SELECT id, title, current_amount, target_amount
		FROM campaigns
		WHERE status::text = 'approved'
		  AND deleted_at IS NULL
		  AND current_amount < target_amount
		ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load campaigns"})
		return
	}
	defer rows.Close()

	type Row struct {
		ID            uuid.UUID `json:"id"`
		Title         string    `json:"title"`
		CurrentAmount int64     `json:"current_amount"`
		TargetAmount  int64     `json:"target_amount"`
	}

	result := []Row{}
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Title, &r.CurrentAmount, &r.TargetAmount); err != nil {
			continue
		}
		result = append(result, r)
	}
	c.JSON(http.StatusOK, gin.H{"campaigns": result})
}
