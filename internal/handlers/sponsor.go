package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Nysonn/campuscare/internal/mail"
	"github.com/Nysonn/campuscare/internal/stream"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SponsorHandler struct {
	DB     *pgxpool.Pool
	Mailer *mail.Mailer
	Stream *stream.Client
}

// BecomeSponsor — POST /sponsors/me
// Registers the authenticated student as a sponsor and forces their profile
// to be non-anonymous (sponsors must be visible to the community).
func (h *SponsorHandler) BecomeSponsor(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var body struct {
		WhatIOffer string `json:"what_i_offer"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if strings.TrimSpace(body.WhatIOffer) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please describe what support you can offer"})
		return
	}

	// Sponsors must be visible — clear anonymous flag.
	if _, err := h.DB.Exec(c,
		`UPDATE student_profiles SET is_anonymous = false, updated_at = now() WHERE user_id = $1`,
		userID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	if _, err := h.DB.Exec(c,
		`INSERT INTO sponsor_profiles (user_id, what_i_offer)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE
		   SET what_i_offer = EXCLUDED.what_i_offer,
		       is_active    = true,
		       updated_at   = now()`,
		userID, strings.TrimSpace(body.WhatIOffer),
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register as sponsor"})
		return
	}

	go func() {
		var sponsorEmail, sponsorName string
		if err := h.DB.QueryRow(context.Background(),
			`SELECT u.email, sp.display_name FROM users u
			 JOIN student_profiles sp ON sp.user_id = u.id WHERE u.id = $1`,
			userID,
		).Scan(&sponsorEmail, &sponsorName); err != nil {
			return
		}
		h.Mailer.SendAsync(
			sponsorEmail,
			"You're now a sponsor on CampusCare!",
			mail.NewSponsorTemplate(sponsorName),
		)
		CreateNotification(context.Background(), h.DB, userID,
			"You're a Sponsor! 🤝",
			"You are now listed as a sponsor on CampusCare. Students can now send you sponsorship requests.",
			"sponsor",
		)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "You are now listed as a sponsor"})
}

// OptOut — DELETE /sponsors/me
// Removes the sponsor listing and terminates all active sponsorships.
func (h *SponsorHandler) OptOut(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	// Collect active sponsorships to terminate.
	rows, err := h.DB.Query(c,
		`SELECT id, stream_channel_id, sponsee_id
		 FROM sponsorships
		 WHERE sponsor_id = $1 AND terminated_at IS NULL`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query sponsorships"})
		return
	}
	type activeSponsorship struct {
		id        uuid.UUID
		channelID string
		sponseeID uuid.UUID
	}
	var actives []activeSponsorship
	for rows.Next() {
		var s activeSponsorship
		rows.Scan(&s.id, &s.channelID, &s.sponseeID)
		actives = append(actives, s)
	}
	rows.Close()

	// Terminate each active sponsorship.
	for _, s := range actives {
		h.DB.Exec(c, `UPDATE sponsorships SET terminated_at = now() WHERE id = $1`, s.id)
		go h.Stream.DeleteChannel(s.channelID)
		go h.notifySponseeTerminated(s.sponseeID, userID)
	}

	// Deactivate the sponsor profile (keeps history, just hides from listing).
	if _, err := h.DB.Exec(c,
		`UPDATE sponsor_profiles SET is_active = false, updated_at = now() WHERE user_id = $1`,
		userID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate sponsor profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "You have opted out as a sponsor"})
}

// IsSponsor — GET /sponsors/me/status
// Tells the frontend whether the current user is an active sponsor.
func (h *SponsorHandler) IsSponsor(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var isActive bool
	var whatIOffer string
	err := h.DB.QueryRow(c,
		`SELECT is_active, what_i_offer FROM sponsor_profiles WHERE user_id = $1`,
		userID,
	).Scan(&isActive, &whatIOffer)
	if err != nil {
		// No row means not a sponsor.
		c.JSON(http.StatusOK, gin.H{"is_sponsor": false, "what_i_offer": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_sponsor": isActive, "what_i_offer": whatIOffer})
}

// ListSponsors — GET /sponsors
// Returns all active sponsors visible to students, including whether the
// calling student already has a pending request or active sponsorship with each.
func (h *SponsorHandler) ListSponsors(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	rows, err := h.DB.Query(c,
		`SELECT
		    u.id,
		    sp.display_name,
		    sp.bio,
		    sp.university,
		    sp.course,
		    sp.avatar_url,
		    spo.what_i_offer,
		    EXISTS(
		        SELECT 1 FROM sponsor_requests sr
		        WHERE sr.requester_id = $1
		          AND sr.sponsor_id = u.id
		          AND sr.status = 'pending'
		    ) AS has_pending_request,
		    EXISTS(
		        SELECT 1 FROM sponsorships s
		        WHERE s.sponsee_id = $1
		          AND s.sponsor_id = u.id
		          AND s.terminated_at IS NULL
		    ) AS is_my_sponsor,
		    EXISTS(
		        SELECT 1 FROM sponsorships s
		        WHERE s.sponsor_id = u.id
		          AND s.terminated_at IS NULL
		    ) AS sponsor_is_busy
		 FROM users u
		 JOIN student_profiles sp  ON sp.user_id  = u.id
		 JOIN sponsor_profiles spo ON spo.user_id = u.id
		 WHERE u.deleted_at IS NULL
		   AND spo.is_active = true
		   AND u.id != $1
		 ORDER BY spo.created_at DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sponsors"})
		return
	}
	defer rows.Close()

	var list []gin.H
	for rows.Next() {
		var id uuid.UUID
		var displayName, bio, university, course, avatarURL, whatIOffer string
		var hasPendingRequest, isMySponsors, sponsorIsBusy bool

		rows.Scan(&id, &displayName, &bio, &university, &course, &avatarURL,
			&whatIOffer, &hasPendingRequest, &isMySponsors, &sponsorIsBusy)

		list = append(list, gin.H{
			"id":                  id,
			"display_name":        displayName,
			"bio":                 bio,
			"university":          university,
			"course":              course,
			"avatar_url":          avatarURL,
			"what_i_offer":        whatIOffer,
			"has_pending_request": hasPendingRequest,
			"is_my_sponsor":       isMySponsors,
			"sponsor_is_busy":     sponsorIsBusy,
		})
	}
	if list == nil {
		list = []gin.H{}
	}
	c.JSON(http.StatusOK, list)
}

// SendRequest — POST /sponsors/:id/request
// Student sends a connection request to a sponsor.
func (h *SponsorHandler) SendRequest(c *gin.Context) {
	requesterID := c.MustGet("user_id").(uuid.UUID)
	sponsorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sponsor ID"})
		return
	}
	if requesterID == sponsorID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot request yourself as a sponsor"})
		return
	}

	// One active sponsorship per student.
	var hasActiveSponsor bool
	h.DB.QueryRow(c,
		`SELECT EXISTS(SELECT 1 FROM sponsorships WHERE sponsee_id=$1 AND terminated_at IS NULL)`,
		requesterID,
	).Scan(&hasActiveSponsor)
	if hasActiveSponsor {
		c.JSON(http.StatusConflict, gin.H{"error": "You already have an active sponsor"})
		return
	}

	// Check target sponsor exists and is active.
	var sponsorActive bool
	h.DB.QueryRow(c,
		`SELECT EXISTS(SELECT 1 FROM sponsor_profiles WHERE user_id=$1 AND is_active=true)`,
		sponsorID,
	).Scan(&sponsorActive)
	if !sponsorActive {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sponsor not found or no longer active"})
		return
	}

	// Each sponsor can only take on one sponsee at a time.
	var sponsorIsBusy bool
	h.DB.QueryRow(c,
		`SELECT EXISTS(SELECT 1 FROM sponsorships WHERE sponsor_id=$1 AND terminated_at IS NULL)`,
		sponsorID,
	).Scan(&sponsorIsBusy)
	if sponsorIsBusy {
		c.JSON(http.StatusConflict, gin.H{"error": "This sponsor is currently supporting someone else"})
		return
	}

	// Check for an existing request between this pair.
	var existingStatus string
	existingErr := h.DB.QueryRow(c,
		`SELECT status::text FROM sponsor_requests WHERE requester_id=$1 AND sponsor_id=$2`,
		requesterID, sponsorID,
	).Scan(&existingStatus)

	if existingErr == nil {
		switch existingStatus {
		case "pending":
			c.JSON(http.StatusConflict, gin.H{"error": "You already have a pending request to this sponsor"})
			return
		case "accepted":
			c.JSON(http.StatusConflict, gin.H{"error": "This sponsor already accepted your request"})
			return
		case "declined":
			// Allow re-requesting after a decline.
			h.DB.Exec(c,
				`UPDATE sponsor_requests SET status='pending', updated_at=now()
				 WHERE requester_id=$1 AND sponsor_id=$2`,
				requesterID, sponsorID,
			)
			go h.notifySponsorOfRequest(sponsorID, requesterID)
			c.JSON(http.StatusOK, gin.H{"message": "Request sent"})
			return
		}
	}

	// Fresh insert.
	var requestID uuid.UUID
	if err := h.DB.QueryRow(c,
		`INSERT INTO sponsor_requests (requester_id, sponsor_id) VALUES ($1, $2) RETURNING id`,
		requesterID, sponsorID,
	).Scan(&requestID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Could not send request. Please try again."})
		return
	}

	go h.notifySponsorOfRequest(sponsorID, requesterID)
	c.JSON(http.StatusCreated, gin.H{"message": "Request sent", "request_id": requestID})
}

// IncomingRequests — GET /sponsors/incoming-requests
// Lists all pending requests received by the calling user (as a sponsor).
func (h *SponsorHandler) IncomingRequests(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var isSponsor bool
	h.DB.QueryRow(c,
		`SELECT EXISTS(SELECT 1 FROM sponsor_profiles WHERE user_id=$1 AND is_active=true)`,
		userID,
	).Scan(&isSponsor)
	if !isSponsor {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not an active sponsor"})
		return
	}

	rows, err := h.DB.Query(c,
		`SELECT sr.id, sr.requester_id,
		        sp.display_name, sp.bio, sp.university, sp.course, sp.avatar_url,
		        sr.status::text, sr.created_at
		 FROM sponsor_requests sr
		 JOIN student_profiles sp ON sp.user_id = sr.requester_id
		 WHERE sr.sponsor_id = $1
		   AND sr.status = 'pending'
		 ORDER BY sr.created_at DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch requests"})
		return
	}
	defer rows.Close()

	var list []gin.H
	for rows.Next() {
		var id, requesterID uuid.UUID
		var displayName, bio, university, course, avatarURL, status string
		var createdAt time.Time
		rows.Scan(&id, &requesterID, &displayName, &bio, &university, &course, &avatarURL, &status, &createdAt)
		list = append(list, gin.H{
			"id":           id,
			"requester_id": requesterID,
			"display_name": displayName,
			"bio":          bio,
			"university":   university,
			"course":       course,
			"avatar_url":   avatarURL,
			"status":       status,
			"created_at":   createdAt,
		})
	}
	if list == nil {
		list = []gin.H{}
	}
	c.JSON(http.StatusOK, list)
}

// OutgoingRequests — GET /sponsors/my-requests
// Lists all requests the calling student has sent to sponsors.
func (h *SponsorHandler) OutgoingRequests(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	rows, err := h.DB.Query(c,
		`SELECT sr.id, sr.sponsor_id,
		        sp.display_name, sp.avatar_url, sp.university,
		        spo.what_i_offer, sr.status::text, sr.created_at
		 FROM sponsor_requests sr
		 JOIN student_profiles sp  ON sp.user_id  = sr.sponsor_id
		 JOIN sponsor_profiles spo ON spo.user_id = sr.sponsor_id
		 WHERE sr.requester_id = $1
		 ORDER BY sr.created_at DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch requests"})
		return
	}
	defer rows.Close()

	var list []gin.H
	for rows.Next() {
		var id, sponsorID uuid.UUID
		var displayName, avatarURL, university, whatIOffer, status string
		var createdAt time.Time
		rows.Scan(&id, &sponsorID, &displayName, &avatarURL, &university, &whatIOffer, &status, &createdAt)
		list = append(list, gin.H{
			"id":           id,
			"sponsor_id":   sponsorID,
			"display_name": displayName,
			"avatar_url":   avatarURL,
			"university":   university,
			"what_i_offer": whatIOffer,
			"status":       status,
			"created_at":   createdAt,
		})
	}
	if list == nil {
		list = []gin.H{}
	}
	c.JSON(http.StatusOK, list)
}

// RespondToRequest — PUT /sponsor-requests/:id
// Sponsor accepts or declines a pending request.
// On accept: creates a Stream Chat channel and a sponsorship record,
// then auto-declines all other pending requests to/from both parties.
func (h *SponsorHandler) RespondToRequest(c *gin.Context) {
	sponsorID := c.MustGet("user_id").(uuid.UUID)
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var body struct {
		Action string `json:"action"` // "accepted" or "declined"
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if body.Action != "accepted" && body.Action != "declined" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be 'accepted' or 'declined'"})
		return
	}

	// Load the request — make sure it belongs to this sponsor.
	var requesterID uuid.UUID
	var currentStatus string
	if err := h.DB.QueryRow(c,
		`SELECT requester_id, status::text FROM sponsor_requests WHERE id=$1 AND sponsor_id=$2`,
		requestID, sponsorID,
	).Scan(&requesterID, &currentStatus); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}
	if currentStatus != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "This request has already been responded to"})
		return
	}

	// Update the request status.
	if _, err := h.DB.Exec(c,
		`UPDATE sponsor_requests SET status=$1, updated_at=now() WHERE id=$2`,
		body.Action, requestID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update request"})
		return
	}

	if body.Action == "accepted" {
		// Verify neither party already has an active sponsorship (race-condition guard).
		var conflict bool
		h.DB.QueryRow(c,
			`SELECT EXISTS(
			    SELECT 1 FROM sponsorships
			    WHERE (sponsor_id=$1 OR sponsee_id=$2)
			      AND terminated_at IS NULL
			)`, sponsorID, requesterID,
		).Scan(&conflict)
		if conflict {
			// Roll back the status update.
			h.DB.Exec(c, `UPDATE sponsor_requests SET status='pending', updated_at=now() WHERE id=$1`, requestID)
			c.JSON(http.StatusConflict, gin.H{"error": "One of the parties already has an active sponsorship"})
			return
		}

		// Build deterministic channel ID from sorted UUIDs.
		a, b := sponsorID.String(), requesterID.String()
		if a > b {
			a, b = b, a
		}
		channelID := fmt.Sprintf("spo-%s-%s", a[:8], b[:8])

		// Resolve display names for Stream user upsert.
		names := make(map[string]string)
		var sponsorName, sponseeName string
		h.DB.QueryRow(c, `SELECT display_name FROM student_profiles WHERE user_id=$1`, sponsorID).Scan(&sponsorName)
		h.DB.QueryRow(c, `SELECT display_name FROM student_profiles WHERE user_id=$1`, requesterID).Scan(&sponseeName)
		names[sponsorID.String()] = sponsorName
		names[requesterID.String()] = sponseeName

		// Set up Stream Chat (upsert users → create channel).
		if err := h.Stream.UpsertUsers([]string{sponsorID.String(), requesterID.String()}, names); err != nil {
			h.DB.Exec(c, `UPDATE sponsor_requests SET status='pending', updated_at=now() WHERE id=$1`, requestID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set up messaging"})
			return
		}
		if err := h.Stream.CreateChannel(channelID, sponsorID.String(), []string{sponsorID.String(), requesterID.String()}); err != nil {
			h.DB.Exec(c, `UPDATE sponsor_requests SET status='pending', updated_at=now() WHERE id=$1`, requestID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat channel"})
			return
		}

		// Persist the sponsorship.
		if _, err := h.DB.Exec(c,
			`INSERT INTO sponsorships (sponsor_id, sponsee_id, stream_channel_id) VALUES ($1, $2, $3)`,
			sponsorID, requesterID, channelID,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sponsorship record"})
			return
		}

		// Auto-decline all other pending requests involving either party.
		h.DB.Exec(c,
			`UPDATE sponsor_requests SET status='declined', updated_at=now()
			 WHERE id != $1
			   AND status = 'pending'
			   AND (sponsor_id=$2 OR requester_id=$3)`,
			requestID, sponsorID, requesterID,
		)

		go h.notifySponseeAccepted(requesterID, sponsorID)
	} else {
		go h.notifySponseeDeclined(requesterID, sponsorID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request " + body.Action})
}

// CancelRequest — DELETE /sponsor-requests/:id
// Lets the original requester withdraw a pending request.
func (h *SponsorHandler) CancelRequest(c *gin.Context) {
	requesterID := c.MustGet("user_id").(uuid.UUID)
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	res, err := h.DB.Exec(c,
		`DELETE FROM sponsor_requests WHERE id=$1 AND requester_id=$2 AND status='pending'`,
		requestID, requesterID,
	)
	if err != nil || res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pending request not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Request cancelled"})
}

// MySponsorship — GET /sponsorships/mine
// Returns the calling student's active sponsorship (as sponsor or sponsee).
func (h *SponsorHandler) MySponsorship(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var sponsorID, sponseeID uuid.UUID
	var channelID string
	var createdAt time.Time

	// Try as sponsee first.
	err := h.DB.QueryRow(c,
		`SELECT sponsor_id, sponsee_id, stream_channel_id, created_at
		 FROM sponsorships WHERE sponsee_id=$1 AND terminated_at IS NULL`,
		userID,
	).Scan(&sponsorID, &sponseeID, &channelID, &createdAt)

	if err != nil {
		// Try as sponsor.
		err = h.DB.QueryRow(c,
			`SELECT sponsor_id, sponsee_id, stream_channel_id, created_at
			 FROM sponsorships WHERE sponsor_id=$1 AND terminated_at IS NULL`,
			userID,
		).Scan(&sponsorID, &sponseeID, &channelID, &createdAt)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"sponsorship": nil})
			return
		}
	}

	// Identify the partner.
	partnerID := sponsorID
	partnerRole := "sponsor"
	if userID == sponsorID {
		partnerID = sponseeID
		partnerRole = "sponsee"
	}

	var partnerName, partnerAvatar string
	h.DB.QueryRow(c,
		`SELECT display_name, avatar_url FROM student_profiles WHERE user_id=$1`,
		partnerID,
	).Scan(&partnerName, &partnerAvatar)

	c.JSON(http.StatusOK, gin.H{
		"sponsorship": gin.H{
			"channel_id":     channelID,
			"partner_id":     partnerID,
			"partner_name":   partnerName,
			"partner_avatar": partnerAvatar,
			"partner_role":   partnerRole,
			"created_at":     createdAt,
		},
	})
}

// TerminateSponsorship — DELETE /sponsorships/mine
// Either party (sponsor or sponsee) can end the active sponsorship at any time.
// Marks it as terminated, deletes the Stream channel, and notifies the other party.
func (h *SponsorHandler) TerminateSponsorship(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var sponsorshipID uuid.UUID
	var channelID string
	var sponsorID, sponseeID uuid.UUID

	err := h.DB.QueryRow(c,
		`SELECT id, stream_channel_id, sponsor_id, sponsee_id
		 FROM sponsorships
		 WHERE (sponsor_id = $1 OR sponsee_id = $1) AND terminated_at IS NULL`,
		userID,
	).Scan(&sponsorshipID, &channelID, &sponsorID, &sponseeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active sponsorship found"})
		return
	}

	if _, err := h.DB.Exec(c,
		`UPDATE sponsorships SET terminated_at = now() WHERE id = $1`,
		sponsorshipID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end sponsorship"})
		return
	}

	go h.Stream.DeleteChannel(channelID)

	// Notify the other party depending on who triggered the termination.
	if userID == sponsorID {
		// Sponsor ended it — notify the sponsee using the existing template.
		go h.notifySponseeTerminated(sponseeID, sponsorID)
	} else {
		// Sponsee ended it — notify the sponsor with a dedicated message.
		go h.notifySponsorTerminated(sponsorID, sponseeID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sponsorship ended"})
}

// GetStreamToken — GET /stream/token
// Returns a short-lived Stream Chat JWT for the authenticated user.
func (h *SponsorHandler) GetStreamToken(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	token, err := h.Stream.GenerateUserToken(userID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"user_id": userID.String(),
		"api_key": h.Stream.APIKey,
	})
}

// AdminListSponsors — GET /admin/sponsors
// Admin view of all sponsors and their current sponsee.
func (h *SponsorHandler) AdminListSponsors(c *gin.Context) {
	rows, err := h.DB.Query(c,
		`SELECT u.id, sp.display_name, sp.university, sp.avatar_url,
		        spo.what_i_offer, spo.is_active, spo.created_at
		 FROM users u
		 JOIN student_profiles sp  ON sp.user_id  = u.id
		 JOIN sponsor_profiles spo ON spo.user_id = u.id
		 ORDER BY spo.created_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sponsors"})
		return
	}
	defer rows.Close()

	var list []gin.H
	for rows.Next() {
		var id uuid.UUID
		var displayName, university, avatarURL, whatIOffer string
		var isActive bool
		var createdAt time.Time
		rows.Scan(&id, &displayName, &university, &avatarURL, &whatIOffer, &isActive, &createdAt)

		// Load current sponsee if any.
		var sponseeInfo gin.H
		var sponseeID uuid.UUID
		var sponseeName, sponseeAvatar string
		var since time.Time
		sponseeErr := h.DB.QueryRow(c,
			`SELECT s.sponsee_id, sp2.display_name, sp2.avatar_url, s.created_at
			 FROM sponsorships s
			 JOIN student_profiles sp2 ON sp2.user_id = s.sponsee_id
			 WHERE s.sponsor_id = $1 AND s.terminated_at IS NULL
			 LIMIT 1`,
			id,
		).Scan(&sponseeID, &sponseeName, &sponseeAvatar, &since)
		if sponseeErr == nil {
			sponseeInfo = gin.H{
				"id":           sponseeID,
				"display_name": sponseeName,
				"avatar_url":   sponseeAvatar,
				"since":        since,
			}
		}

		list = append(list, gin.H{
			"id":           id,
			"display_name": displayName,
			"university":   university,
			"avatar_url":   avatarURL,
			"what_i_offer": whatIOffer,
			"is_active":    isActive,
			"created_at":   createdAt,
			"sponsee":      sponseeInfo,
		})
	}
	if list == nil {
		list = []gin.H{}
	}
	c.JSON(http.StatusOK, list)
}

// ── Email notification helpers ────────────────────────────────────────────────

func (h *SponsorHandler) notifySponsorOfRequest(sponsorID, requesterID uuid.UUID) {
	var sponsorEmail, sponsorName, requesterName string
	if err := h.DB.QueryRow(context.Background(),
		`SELECT u.email, sp.display_name FROM users u
		 JOIN student_profiles sp ON sp.user_id = u.id WHERE u.id=$1`,
		sponsorID,
	).Scan(&sponsorEmail, &sponsorName); err != nil {
		return
	}
	h.DB.QueryRow(context.Background(),
		`SELECT display_name FROM student_profiles WHERE user_id=$1`, requesterID,
	).Scan(&requesterName)

	h.Mailer.SendAsync(
		sponsorEmail,
		"Someone wants you as their sponsor on CampusCare",
		mail.SponsorRequestReceivedTemplate(sponsorName, requesterName),
	)
	CreateNotification(context.Background(), h.DB, sponsorID,
		"New Sponsor Request",
		requesterName+" has requested you as their sponsor on CampusCare.",
		"sponsor",
	)
}

func (h *SponsorHandler) notifySponseeAccepted(sponseeID, sponsorID uuid.UUID) {
	var sponseeEmail, sponseeName, sponsorName string
	if err := h.DB.QueryRow(context.Background(),
		`SELECT u.email, sp.display_name FROM users u
		 JOIN student_profiles sp ON sp.user_id = u.id WHERE u.id=$1`,
		sponseeID,
	).Scan(&sponseeEmail, &sponseeName); err != nil {
		return
	}
	h.DB.QueryRow(context.Background(),
		`SELECT display_name FROM student_profiles WHERE user_id=$1`, sponsorID,
	).Scan(&sponsorName)

	h.Mailer.SendAsync(
		sponseeEmail,
		"Your sponsor request has been accepted!",
		mail.SponsorRequestAcceptedTemplate(sponseeName, sponsorName),
	)
	CreateNotification(context.Background(), h.DB, sponseeID,
		"Sponsor Request Accepted",
		"Great news! "+sponsorName+" has accepted your sponsor request.",
		"sponsor",
	)
}

func (h *SponsorHandler) notifySponseeDeclined(sponseeID, sponsorID uuid.UUID) {
	var sponseeEmail, sponseeName, sponsorName string
	if err := h.DB.QueryRow(context.Background(),
		`SELECT u.email, sp.display_name FROM users u
		 JOIN student_profiles sp ON sp.user_id = u.id WHERE u.id=$1`,
		sponseeID,
	).Scan(&sponseeEmail, &sponseeName); err != nil {
		return
	}
	h.DB.QueryRow(context.Background(),
		`SELECT display_name FROM student_profiles WHERE user_id=$1`, sponsorID,
	).Scan(&sponsorName)

	h.Mailer.SendAsync(
		sponseeEmail,
		"Update on your sponsor request",
		mail.SponsorRequestDeclinedTemplate(sponseeName, sponsorName),
	)
	CreateNotification(context.Background(), h.DB, sponseeID,
		"Sponsor Request Declined",
		"Your sponsor request to "+sponsorName+" was not accepted at this time.",
		"sponsor",
	)
}

func (h *SponsorHandler) notifySponsorTerminated(sponsorID, sponseeID uuid.UUID) {
	var sponsorEmail, sponsorName, sponseeName string
	if err := h.DB.QueryRow(context.Background(),
		`SELECT u.email, sp.display_name FROM users u
		 JOIN student_profiles sp ON sp.user_id = u.id WHERE u.id=$1`,
		sponsorID,
	).Scan(&sponsorEmail, &sponsorName); err != nil {
		return
	}
	h.DB.QueryRow(context.Background(),
		`SELECT display_name FROM student_profiles WHERE user_id=$1`, sponseeID,
	).Scan(&sponseeName)

	h.Mailer.SendAsync(
		sponsorEmail,
		"A sponsorship has ended on CampusCare",
		mail.SponsorTerminatedBySponsoreeTemplate(sponsorName, sponseeName),
	)
	CreateNotification(context.Background(), h.DB, sponsorID,
		"Sponsorship Ended",
		"Your sponsorship with "+sponseeName+" has been terminated.",
		"sponsor",
	)
}

// NotifyPartnerMessage — POST /sponsorships/notify-message
// Called by the frontend after the user sends a chat message.
// Sends an email to the partner only if:
//   - they have not been active in the last 10 minutes, AND
//   - no notification email has been sent for this sponsorship in the last hour.
func (h *SponsorHandler) NotifyPartnerMessage(c *gin.Context) {
	senderID := c.MustGet("user_id").(uuid.UUID)

	// Resolve active sponsorship and partner details.
	var sponsorID, sponseeID uuid.UUID
	var channelID string
	var lastNotified *time.Time

	err := h.DB.QueryRow(c,
		`SELECT sponsor_id, sponsee_id, stream_channel_id, last_message_notified_at
		 FROM sponsorships
		 WHERE (sponsor_id = $1 OR sponsee_id = $1) AND terminated_at IS NULL`,
		senderID,
	).Scan(&sponsorID, &sponseeID, &channelID, &lastNotified)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"sent": false, "reason": "no active sponsorship"})
		return
	}

	// Throttle: skip if an email was sent in the last hour.
	if lastNotified != nil && time.Since(*lastNotified) < time.Hour {
		c.JSON(http.StatusOK, gin.H{"sent": false, "reason": "throttled"})
		return
	}

	// Determine who is the partner.
	partnerID := sponsorID
	senderRole := "sponsor"
	if senderID == sponsorID {
		partnerID = sponseeID
		senderRole = "sponsee"
	}

	// Check partner's last_active_at — skip email if active within last 10 minutes.
	var partnerLastActive *time.Time
	h.DB.QueryRow(c,
		`SELECT last_active_at FROM users WHERE id = $1`, partnerID,
	).Scan(&partnerLastActive)
	if partnerLastActive != nil && time.Since(*partnerLastActive) < 10*time.Minute {
		c.JSON(http.StatusOK, gin.H{"sent": false, "reason": "partner is online"})
		return
	}

	// Fetch names and partner email.
	var senderName, partnerName, partnerEmail string
	h.DB.QueryRow(c,
		`SELECT display_name FROM student_profiles WHERE user_id = $1`, senderID,
	).Scan(&senderName)
	if err := h.DB.QueryRow(c,
		`SELECT u.email, sp.display_name FROM users u
		 JOIN student_profiles sp ON sp.user_id = u.id WHERE u.id = $1`,
		partnerID,
	).Scan(&partnerEmail, &partnerName); err != nil {
		c.JSON(http.StatusOK, gin.H{"sent": false, "reason": "partner not found"})
		return
	}

	// Update throttle timestamp.
	h.DB.Exec(c,
		`UPDATE sponsorships SET last_message_notified_at = now()
		 WHERE (sponsor_id = $1 OR sponsee_id = $1) AND terminated_at IS NULL`,
		senderID,
	)

	go h.Mailer.SendAsync(
		partnerEmail,
		senderName+" sent you a message on CampusCare",
		mail.SponsorChatNotificationTemplate(partnerName, senderName, senderRole),
	)
	CreateNotification(c, h.DB, partnerID,
		"New Message from "+senderName,
		senderName+" has sent you a message. Open CampusCare to read and reply.",
		"general",
	)

	c.JSON(http.StatusOK, gin.H{"sent": true})
}

func (h *SponsorHandler) notifySponseeTerminated(sponseeID, sponsorID uuid.UUID) {
	var sponseeEmail, sponseeName, sponsorName string
	if err := h.DB.QueryRow(context.Background(),
		`SELECT u.email, sp.display_name FROM users u
		 JOIN student_profiles sp ON sp.user_id = u.id WHERE u.id=$1`,
		sponseeID,
	).Scan(&sponseeEmail, &sponseeName); err != nil {
		return
	}
	h.DB.QueryRow(context.Background(),
		`SELECT display_name FROM student_profiles WHERE user_id=$1`, sponsorID,
	).Scan(&sponsorName)

	h.Mailer.SendAsync(
		sponseeEmail,
		"Your sponsorship has ended",
		mail.SponsorshipTerminatedTemplate(sponseeName, sponsorName),
	)
	CreateNotification(context.Background(), h.DB, sponseeID,
		"Sponsorship Ended",
		"Your sponsorship with "+sponsorName+" has ended.",
		"sponsor",
	)
}
