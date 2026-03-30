package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/Nysonn/campuscare/internal/audit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignHandler struct {
	DB *pgxpool.Pool
}

type AttachmentInput struct {
	URL   string `json:"url"`
	Label string `json:"label"`
}

type CreateCampaignRequest struct {
	Title                   string           `json:"title"`
	Description             string           `json:"description"`
	Target                  int64            `json:"target_amount"`
	Category                string           `json:"category"`
	Attachments             []AttachmentInput `json:"attachments"`
	IsAnonymous             bool             `json:"is_anonymous"`
	UrgencyLevel            string           `json:"urgency_level"`
	BeneficiaryType         string           `json:"beneficiary_type"`
	BeneficiaryName         string           `json:"beneficiary_name"`
	VerificationContactName string           `json:"verification_contact_name"`
	VerificationContactInfo string           `json:"verification_contact_info"`
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
		  verification_contact_name, verification_contact_info)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id`,
		userID, req.Title, req.Description, req.Target, req.Category,
		urgency, beneficiaryType, req.BeneficiaryName,
		req.VerificationContactName, req.VerificationContactInfo,
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
		 SET title=$1,
		     description=$2,
		     target_amount=$3,
		     category=$4,
		     urgency_level=$5,
		     beneficiary_type=$6,
		     beneficiary_name=$7,
		     verification_contact_name=$8,
		     verification_contact_info=$9,
		     status='pending',
		     updated_at=now()
		 WHERE id=$10 AND student_id=$11 AND deleted_at IS NULL`,
		req.Title, req.Description, req.Target, req.Category,
		urgency, beneficiaryType, req.BeneficiaryName,
		req.VerificationContactName, req.VerificationContactInfo,
		campaignID, userID,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Update failed"})
		return
	}

	// Replace attachments: delete existing then insert new ones.
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
		`UPDATE campaigns
		 SET deleted_at=now()
		 WHERE id=$1 AND student_id=$2`,
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
		        sp.is_anonymous,
		        CASE WHEN sp.is_anonymous THEN '' ELSE sp.display_name END AS author,
		        CASE WHEN sp.is_anonymous THEN '' ELSE sp.avatar_url END AS avatar_url
		 FROM campaigns c
		 JOIN student_profiles sp ON sp.user_id = c.student_id
		 WHERE c.status='approved'
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
		var title, desc, author, avatarURL string
		var target, current int64
		var createdAt time.Time
		var isAnonymous bool

		rows.Scan(&id, &title, &desc, &target, &current, &createdAt, &isAnonymous, &author, &avatarURL)

		campaigns = append(campaigns, gin.H{
			"id":             id,
			"title":          title,
			"description":    desc,
			"target_amount":  target,
			"current_amount": current,
			"created_at":     createdAt,
			"is_anonymous":   isAnonymous,
			"author":         author,
			"avatar_url":     avatarURL,
		})
	}

	c.JSON(http.StatusOK, campaigns)
}

func (h *CampaignHandler) ListPending(c *gin.Context) {

	rows, err := h.DB.Query(context.Background(),
		`SELECT c.id, c.student_id, c.title, c.description, c.target_amount, c.category, c.created_at,
		        c.urgency_level, c.beneficiary_type,
		        COALESCE(c.beneficiary_name, '') AS beneficiary_name,
		        COALESCE(c.verification_contact_name, '') AS verification_contact_name,
		        COALESCE(c.verification_contact_info, '') AS verification_contact_info,
		        sp.is_anonymous,
		        COALESCE(sp.display_name, '') AS student_name
		 FROM campaigns c
		 JOIN student_profiles sp ON sp.user_id = c.student_id
		 WHERE c.status = 'pending'
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
		var verificationContactName, verificationContactInfo string
		var studentName string
		var target int64
		var createdAt time.Time
		var isAnonymous bool

		rows.Scan(&id, &studentID, &title, &desc, &target, &category, &createdAt,
			&urgencyLevel, &beneficiaryType, &beneficiaryName,
			&verificationContactName, &verificationContactInfo,
			&isAnonymous, &studentName)

		// Fetch attachments for this campaign.
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
			"category":                  category,
			"created_at":                createdAt,
			"urgency_level":             urgencyLevel,
			"beneficiary_type":          beneficiaryType,
			"beneficiary_name":          beneficiaryName,
			"verification_contact_name": verificationContactName,
			"verification_contact_info": verificationContactInfo,
			"is_anonymous":              isAnonymous,
			"attachments":               attachments,
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

	_, err := h.DB.Exec(context.Background(),
		`UPDATE campaigns SET status=$1 WHERE id=$2`,
		body.Status, campaignID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Approval failed"})
		return
	}

	audit.Log(h.DB, adminID, "APPROVE_CAMPAIGN", "campaign", campaignID, body)

	c.JSON(http.StatusOK, gin.H{"message": "Campaign status updated"})
}

func (h *CampaignHandler) MyCampaigns(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)

	rows, err := h.DB.Query(context.Background(),
		`SELECT id, title, description, target_amount, current_amount, category, status, created_at,
		        urgency_level, beneficiary_type,
		        COALESCE(beneficiary_name, '') AS beneficiary_name,
		        COALESCE(verification_contact_name, '') AS verification_contact_name,
		        COALESCE(verification_contact_info, '') AS verification_contact_info
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
		var target, current int64
		var createdAt time.Time

		rows.Scan(&id, &title, &desc, &target, &current, &category, &status, &createdAt,
			&urgencyLevel, &beneficiaryType, &beneficiaryName,
			&verificationContactName, &verificationContactInfo)

		// Fetch attachments for this campaign.
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
			"attachments":               attachments,
		})
	}

	if campaigns == nil {
		campaigns = []gin.H{}
	}

	c.JSON(http.StatusOK, campaigns)
}
