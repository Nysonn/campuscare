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

type CreateCampaignRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Target      int64    `json:"target_amount"`
	Category    string   `json:"category"`
	Attachments []string `json:"attachments"`
}

func (h *CampaignHandler) Create(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)

	var req CreateCampaignRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var campaignID uuid.UUID
	err := h.DB.QueryRow(context.Background(),
		`INSERT INTO campaigns
		 (student_id,title,description,target_amount,category)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id`,
		userID, req.Title, req.Description, req.Target, req.Category,
	).Scan(&campaignID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Creation failed"})
		return
	}

	for _, file := range req.Attachments {
		h.DB.Exec(context.Background(),
			`INSERT INTO campaign_attachments (campaign_id,file_url)
			 VALUES ($1,$2)`,
			campaignID, file,
		)
	}

	audit.Log(h.DB, userID, "CREATE_CAMPAIGN", "campaign", campaignID, req)

	c.JSON(http.StatusCreated, gin.H{"message": "Campaign submitted for approval", "campaign_id": campaignID})
}

func (h *CampaignHandler) Update(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)
	idParam := c.Param("id")
	campaignID, _ := uuid.Parse(idParam)

	var req CreateCampaignRequest
	c.BindJSON(&req)

	_, err := h.DB.Exec(context.Background(),
		`UPDATE campaigns
		 SET title=$1,
		     description=$2,
		     target_amount=$3,
		     category=$4,
		     status='pending',
		     updated_at=now()
		 WHERE id=$5 AND student_id=$6 AND deleted_at IS NULL`,
		req.Title, req.Description, req.Target, req.Category, campaignID, userID,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Update failed"})
		return
	}

	audit.Log(h.DB, userID, "UPDATE_CAMPAIGN", "campaign", campaignID, req)

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
		`SELECT id,title,description,target_amount,current_amount,created_at
		 FROM campaigns
		 WHERE status='approved'
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
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
		var title, desc string
		var target, current int64
		var createdAt time.Time

		rows.Scan(&id, &title, &desc, &target, &current, &createdAt)

		campaigns = append(campaigns, gin.H{
			"id":             id,
			"title":          title,
			"description":    desc,
			"target_amount":  target,
			"current_amount": current,
			"created_at":     createdAt,
		})
	}

	c.JSON(http.StatusOK, campaigns)
}

func (h *CampaignHandler) ListPending(c *gin.Context) {

	rows, err := h.DB.Query(context.Background(),
		`SELECT c.id, c.student_id, c.title, c.description, c.target_amount, c.category, c.created_at
		 FROM campaigns c
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
		var target int64
		var createdAt time.Time

		rows.Scan(&id, &studentID, &title, &desc, &target, &category, &createdAt)

		campaigns = append(campaigns, gin.H{
			"id":            id,
			"student_id":    studentID,
			"title":         title,
			"description":   desc,
			"target_amount": target,
			"category":      category,
			"created_at":    createdAt,
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
