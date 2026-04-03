package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Nysonn/campuscare/internal/audit"
	"github.com/Nysonn/campuscare/internal/mail"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignHandler struct {
	DB     *pgxpool.Pool
	Mailer *mail.Mailer
}

type AttachmentInput struct {
	URL   string `json:"url"`
	Label string `json:"label"`
}

type CreateCampaignRequest struct {
	Title                   string            `json:"title"`
	Description             string            `json:"description"`
	Target                  int64             `json:"target_amount"`
	Category                string            `json:"category"`
	Attachments             []AttachmentInput `json:"attachments"`
	IsAnonymous             bool              `json:"is_anonymous"`
	UrgencyLevel            string            `json:"urgency_level"`
	BeneficiaryType         string            `json:"beneficiary_type"`
	BeneficiaryName         string            `json:"beneficiary_name"`
	VerificationContactName string            `json:"verification_contact_name"`
	VerificationContactInfo string            `json:"verification_contact_info"`
	// Payment destination / account details
	BeneficiaryOrgName string `json:"beneficiary_org_name"`
	BankName           string `json:"bank_name"`
	AccountNumber      string `json:"account_number"`
	AccountHolderName  string `json:"account_holder_name"`
}

func (h *CampaignHandler) Create(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)

	var req CreateCampaignRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	urgency := req.UrgencyLevel
	if urgency == "" {
		urgency = "normal"
	}
	beneficiaryType := req.BeneficiaryType
	if beneficiaryType == "" {
		beneficiaryType = "self"
	}

	var campaignID uuid.UUID
	err := h.DB.QueryRow(context.Background(),
		`INSERT INTO campaigns
		 (student_id, title, description, target_amount, category,
		  urgency_level, beneficiary_type, beneficiary_name,
		  verification_contact_name, verification_contact_info,
		  beneficiary_org_name, bank_name, account_number, account_holder_name)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 RETURNING id`,
		userID, req.Title, req.Description, req.Target, req.Category,
		urgency, beneficiaryType, req.BeneficiaryName,
		req.VerificationContactName, req.VerificationContactInfo,
		req.BeneficiaryOrgName, req.BankName, req.AccountNumber, req.AccountHolderName,
	).Scan(&campaignID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Creation failed"})
		return
	}

	for _, att := range req.Attachments {
		label := att.Label
		if label == "" {
			label = "Document"
		}
		h.DB.Exec(context.Background(),
			`INSERT INTO campaign_attachments (campaign_id, file_url, label)
			 VALUES ($1,$2,$3)`,
			campaignID, att.URL, label,
		)
	}

	h.DB.Exec(context.Background(),
		`UPDATE student_profiles SET is_anonymous=$1 WHERE user_id=$2`,
		req.IsAnonymous, userID,
	)

	audit.Log(h.DB, userID, "CREATE_CAMPAIGN", "campaign", campaignID, req)

	c.JSON(http.StatusCreated, gin.H{"message": "Campaign submitted for approval", "campaign_id": campaignID})
}

func (h *CampaignHandler) Update(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)
	idParam := c.Param("id")
	campaignID, _ := uuid.Parse(idParam)

	var req CreateCampaignRequest
	c.BindJSON(&req)

	urgency := req.UrgencyLevel
	if urgency == "" {
		urgency = "normal"
	}
	beneficiaryType := req.BeneficiaryType
	if beneficiaryType == "" {
		beneficiaryType = "self"
	}

	_, err := h.DB.Exec(context.Background(),
		`UPDATE campaigns
		 SET title=$1, description=$2, target_amount=$3, category=$4,
		     urgency_level=$5, beneficiary_type=$6, beneficiary_name=$7,
		     verification_contact_name=$8, verification_contact_info=$9,
		     beneficiary_org_name=$10, bank_name=$11, account_number=$12, account_holder_name=$13,
		     status='pending', account_status='unverified', updated_at=now()
		 WHERE id=$14 AND student_id=$15 AND deleted_at IS NULL`,
		req.Title, req.Description, req.Target, req.Category,
		urgency, beneficiaryType, req.BeneficiaryName,
		req.VerificationContactName, req.VerificationContactInfo,
		req.BeneficiaryOrgName, req.BankName, req.AccountNumber, req.AccountHolderName,
		campaignID, userID,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Update failed"})
		return
	}

	h.DB.Exec(context.Background(),
		`DELETE FROM campaign_attachments WHERE campaign_id=$1`, campaignID,
	)
	for _, att := range req.Attachments {
		label := att.Label
		if label == "" {
			label = "Document"
		}
		h.DB.Exec(context.Background(),
			`INSERT INTO campaign_attachments (campaign_id, file_url, label) VALUES ($1,$2,$3)`,
			campaignID, att.URL, label,
		)
	}

	audit.Log(h.DB, userID, "UPDATE_CAMPAIGN", "campaign", campaignID, req)

	h.DB.Exec(context.Background(),
		`UPDATE student_profiles SET is_anonymous=$1 WHERE user_id=$2`,
		req.IsAnonymous, userID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "Campaign updated and pending approval"})
}

func (h *CampaignHandler) Delete(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)
	idParam := c.Param("id")
	campaignID, _ := uuid.Parse(idParam)

	_, err := h.DB.Exec(context.Background(),
		`UPDATE campaigns SET deleted_at=now() WHERE id=$1 AND student_id=$2`,
		campaignID, userID,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Delete failed"})
		return
	}

	audit.Log(h.DB, userID, "DELETE_CAMPAIGN", "campaign", campaignID, nil)

	c.JSON(http.StatusOK, gin.H{"message": "Campaign deleted"})
}

func (h *CampaignHandler) PublicList(c *gin.Context) {

	rows, err := h.DB.Query(context.Background(),
		`SELECT c.id, c.title, c.description, c.target_amount, c.current_amount, c.created_at,
		        c.status::text, c.account_status,
		        sp.is_anonymous,
		        CASE WHEN sp.is_anonymous THEN '' ELSE sp.display_name END AS author,
		        CASE WHEN sp.is_anonymous THEN '' ELSE sp.avatar_url END AS avatar_url
		 FROM campaigns c
		 JOIN student_profiles sp ON sp.user_id = c.student_id
		 WHERE c.status::text IN ('approved', 'completed')
		   AND c.deleted_at IS NULL
		 ORDER BY c.created_at DESC
		 LIMIT 6`,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fetch failed"})
		return
	}
	defer rows.Close()

	var campaigns []gin.H

	for rows.Next() {
		var id uuid.UUID
		var title, desc, author, avatarURL, status, accountStatus string
		var target, current int64
		var createdAt time.Time
		var isAnonymous bool

		rows.Scan(&id, &title, &desc, &target, &current, &createdAt, &status, &accountStatus, &isAnonymous, &author, &avatarURL)

		campaigns = append(campaigns, gin.H{
			"id":             id,
			"title":          title,
			"description":    desc,
			"target_amount":  target,
			"current_amount": current,
			"created_at":     createdAt,
			"status":         status,
			"account_status": accountStatus,
			"is_anonymous":   isAnonymous,
			"author":         author,
			"avatar_url":     avatarURL,
		})
	}

	if campaigns == nil {
		campaigns = []gin.H{}
	}

	c.JSON(http.StatusOK, campaigns)
}

func (h *CampaignHandler) ListPending(c *gin.Context) {

	statusFilter := c.DefaultQuery("status", "pending")

	rows, err := h.DB.Query(context.Background(),
		`SELECT c.id, c.student_id, c.title, c.description, c.target_amount, c.current_amount, c.category, c.created_at,
		        c.status,
		        c.urgency_level, c.beneficiary_type,
		        COALESCE(c.beneficiary_name, '') AS beneficiary_name,
		        COALESCE(c.verification_contact_name, '') AS verification_contact_name,
		        COALESCE(c.verification_contact_info, '') AS verification_contact_info,
		        COALESCE(c.beneficiary_org_name, '') AS beneficiary_org_name,
		        COALESCE(c.bank_name, '') AS bank_name,
		        COALESCE(c.account_number, '') AS account_number,
		        COALESCE(c.account_holder_name, '') AS account_holder_name,
		        c.account_status,
		        sp.is_anonymous,
		        COALESCE(sp.display_name, '') AS student_name
		 FROM campaigns c
		 JOIN student_profiles sp ON sp.user_id = c.student_id
		 WHERE ($1 = 'all' OR c.status::text = $1)
		   AND c.deleted_at IS NULL
		 ORDER BY c.created_at DESC`,
		statusFilter,
	)

	if err != nil {
		log.Printf("ListPending query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fetch failed"})
		return
	}
	defer rows.Close()

	var campaigns []gin.H

	for rows.Next() {
		var id, studentID uuid.UUID
		var title, desc, category, status string
		var urgencyLevel, beneficiaryType, beneficiaryName string
		var verificationContactName, verificationContactInfo string
		var beneficiaryOrgName, bankName, accountNumber, accountHolderName string
		var accountStatus, studentName string
		var target, currentAmount int64
		var createdAt time.Time
		var isAnonymous bool

		rows.Scan(&id, &studentID, &title, &desc, &target, &currentAmount, &category, &createdAt,
			&status,
			&urgencyLevel, &beneficiaryType, &beneficiaryName,
			&verificationContactName, &verificationContactInfo,
			&beneficiaryOrgName, &bankName, &accountNumber, &accountHolderName,
			&accountStatus, &isAnonymous, &studentName)

		attRows, attErr := h.DB.Query(context.Background(),
			`SELECT file_url, label FROM campaign_attachments WHERE campaign_id = $1`, id,
		)
		var attachments []gin.H
		if attErr == nil {
			for attRows.Next() {
				var url, label string
				attRows.Scan(&url, &label)
				attachments = append(attachments, gin.H{"url": url, "label": label})
			}
			attRows.Close()
		}
		if attachments == nil {
			attachments = []gin.H{}
		}

		campaigns = append(campaigns, gin.H{
			"id":                        id,
			"student_id":                studentID,
			"student_name":              studentName,
			"title":                     title,
			"description":               desc,
			"target_amount":             target,
			"current_amount":            currentAmount,
			"category":                  category,
			"created_at":                createdAt,
			"status":                    status,
			"urgency_level":             urgencyLevel,
			"beneficiary_type":          beneficiaryType,
			"beneficiary_name":          beneficiaryName,
			"verification_contact_name": verificationContactName,
			"verification_contact_info": verificationContactInfo,
			"beneficiary_org_name":      beneficiaryOrgName,
			"bank_name":                 bankName,
			"account_number":            accountNumber,
			"account_holder_name":       accountHolderName,
			"account_status":            accountStatus,
			"is_anonymous":              isAnonymous,
			"attachments":               attachments,
		})
	}

	if campaigns == nil {
		campaigns = []gin.H{}
	}

	c.JSON(http.StatusOK, campaigns)
}

// ListPendingAccounts returns approved campaigns whose bank account has not yet been verified.
func (h *CampaignHandler) ListPendingAccounts(c *gin.Context) {

	rows, err := h.DB.Query(context.Background(),
		`SELECT c.id, c.student_id, c.title, c.description, c.target_amount, c.category, c.created_at,
		        c.urgency_level, c.beneficiary_type,
		        COALESCE(c.beneficiary_name, '') AS beneficiary_name,
		        COALESCE(c.beneficiary_org_name, '') AS beneficiary_org_name,
		        COALESCE(c.bank_name, '') AS bank_name,
		        COALESCE(c.account_number, '') AS account_number,
		        COALESCE(c.account_holder_name, '') AS account_holder_name,
		        c.account_status,
		        COALESCE(sp.display_name, '') AS student_name
		 FROM campaigns c
		 JOIN student_profiles sp ON sp.user_id = c.student_id
		 WHERE c.status = 'approved'
		   AND c.account_status = 'unverified'
		   AND c.deleted_at IS NULL
		 ORDER BY c.created_at ASC`,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fetch failed"})
		return
	}
	defer rows.Close()

	var campaigns []gin.H

	for rows.Next() {
		var id, studentID uuid.UUID
		var title, desc, category string
		var urgencyLevel, beneficiaryType, beneficiaryName string
		var beneficiaryOrgName, bankName, accountNumber, accountHolderName string
		var accountStatus, studentName string
		var target int64
		var createdAt time.Time

		rows.Scan(&id, &studentID, &title, &desc, &target, &category, &createdAt,
			&urgencyLevel, &beneficiaryType, &beneficiaryName,
			&beneficiaryOrgName, &bankName, &accountNumber, &accountHolderName,
			&accountStatus, &studentName)

		campaigns = append(campaigns, gin.H{
			"id":                   id,
			"student_id":           studentID,
			"student_name":         studentName,
			"title":                title,
			"description":          desc,
			"target_amount":        target,
			"category":             category,
			"created_at":           createdAt,
			"urgency_level":        urgencyLevel,
			"beneficiary_type":     beneficiaryType,
			"beneficiary_name":     beneficiaryName,
			"beneficiary_org_name": beneficiaryOrgName,
			"bank_name":            bankName,
			"account_number":       accountNumber,
			"account_holder_name":  accountHolderName,
			"account_status":       accountStatus,
			"attachments":          []gin.H{},
		})
	}

	if campaigns == nil {
		campaigns = []gin.H{}
	}

	c.JSON(http.StatusOK, campaigns)
}

func (h *CampaignHandler) Approve(c *gin.Context) {

	adminID := c.MustGet("user_id").(uuid.UUID)
	idParam := c.Param("id")
	campaignID, _ := uuid.Parse(idParam)

	var body struct {
		Status string `json:"status"`
	}
	c.BindJSON(&body)

	if body.Status != "approved" && body.Status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	// Require verified payment account before approving the campaign.
	if body.Status == "approved" {
		var accountStatus string
		err := h.DB.QueryRow(context.Background(),
			`SELECT account_status FROM campaigns WHERE id=$1 AND deleted_at IS NULL`,
			campaignID,
		).Scan(&accountStatus)
		if err != nil || accountStatus != "verified" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payment account must be verified before approving this campaign"})
			return
		}
	}

	_, err := h.DB.Exec(context.Background(),
		`UPDATE campaigns SET status=$1 WHERE id=$2`,
		body.Status, campaignID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Approval failed"})
		return
	}

	audit.Log(h.DB, adminID, "APPROVE_CAMPAIGN", "campaign", campaignID, body)

	if body.Status == "approved" {
		go h.notifyStudentCampaignApproved(campaignID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Campaign status updated"})
}

// VerifyAccount lets the admin mark a campaign's payment account as verified or rejected.
func (h *CampaignHandler) VerifyAccount(c *gin.Context) {

	adminID := c.MustGet("user_id").(uuid.UUID)
	idParam := c.Param("id")
	campaignID, _ := uuid.Parse(idParam)

	var body struct {
		AccountStatus string `json:"account_status"`
	}
	c.BindJSON(&body)

	if body.AccountStatus != "verified" && body.AccountStatus != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_status must be 'verified' or 'rejected'"})
		return
	}

	_, err := h.DB.Exec(context.Background(),
		`UPDATE campaigns SET account_status=$1 WHERE id=$2`,
		body.AccountStatus, campaignID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}

	audit.Log(h.DB, adminID, "VERIFY_ACCOUNT", "campaign", campaignID, body)

	c.JSON(http.StatusOK, gin.H{"message": "Account status updated"})
}

func (h *CampaignHandler) MyCampaigns(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)

	rows, err := h.DB.Query(context.Background(),
		`SELECT id, title, description, target_amount, current_amount, category, status::text, created_at,
		        urgency_level, beneficiary_type,
		        COALESCE(beneficiary_name, '') AS beneficiary_name,
		        COALESCE(verification_contact_name, '') AS verification_contact_name,
		        COALESCE(verification_contact_info, '') AS verification_contact_info,
		        COALESCE(beneficiary_org_name, '') AS beneficiary_org_name,
		        COALESCE(bank_name, '') AS bank_name,
		        COALESCE(account_number, '') AS account_number,
		        COALESCE(account_holder_name, '') AS account_holder_name,
		        account_status
		 FROM campaigns
		 WHERE student_id = $1
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fetch failed"})
		return
	}
	defer rows.Close()

	var campaigns []gin.H

	for rows.Next() {
		var id uuid.UUID
		var title, desc, category, status string
		var urgencyLevel, beneficiaryType, beneficiaryName string
		var verificationContactName, verificationContactInfo string
		var beneficiaryOrgName, bankName, accountNumber, accountHolderName string
		var accountStatus string
		var target, current int64
		var createdAt time.Time

		rows.Scan(&id, &title, &desc, &target, &current, &category, &status, &createdAt,
			&urgencyLevel, &beneficiaryType, &beneficiaryName,
			&verificationContactName, &verificationContactInfo,
			&beneficiaryOrgName, &bankName, &accountNumber, &accountHolderName,
			&accountStatus)

		attRows, attErr := h.DB.Query(context.Background(),
			`SELECT file_url, label FROM campaign_attachments WHERE campaign_id = $1`, id,
		)
		var attachments []gin.H
		if attErr == nil {
			for attRows.Next() {
				var url, label string
				attRows.Scan(&url, &label)
				attachments = append(attachments, gin.H{"url": url, "label": label})
			}
			attRows.Close()
		}
		if attachments == nil {
			attachments = []gin.H{}
		}

		campaigns = append(campaigns, gin.H{
			"id":                        id,
			"title":                     title,
			"description":               desc,
			"target_amount":             target,
			"current_amount":            current,
			"category":                  category,
			"status":                    status,
			"created_at":                createdAt,
			"urgency_level":             urgencyLevel,
			"beneficiary_type":          beneficiaryType,
			"beneficiary_name":          beneficiaryName,
			"verification_contact_name": verificationContactName,
			"verification_contact_info": verificationContactInfo,
			"beneficiary_org_name":      beneficiaryOrgName,
			"bank_name":                 bankName,
			"account_number":            accountNumber,
			"account_holder_name":       accountHolderName,
			"account_status":            accountStatus,
			"attachments":               attachments,
		})
	}

	if campaigns == nil {
		campaigns = []gin.H{}
	}

	c.JSON(http.StatusOK, campaigns)
}

func (h *CampaignHandler) notifyStudentCampaignApproved(campaignID uuid.UUID) {
	var studentEmail, studentName, campaignTitle string
	if err := h.DB.QueryRow(context.Background(),
		`SELECT u.email, sp.display_name, c.title
		 FROM campaigns c
		 JOIN users u ON u.id = c.student_id
		 JOIN student_profiles sp ON sp.user_id = c.student_id
		 WHERE c.id = $1`,
		campaignID,
	).Scan(&studentEmail, &studentName, &campaignTitle); err != nil {
		return
	}

	h.Mailer.SendAsync(
		studentEmail,
		"Your CampusCare campaign has been approved!",
		mail.CampaignApprovedTemplate(studentName, campaignTitle),
	)
}
