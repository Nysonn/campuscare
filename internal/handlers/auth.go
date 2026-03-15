package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/Nysonn/campuscare/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

type AuthHandler struct {
	DB             *pgxpool.Pool
	SessionService *services.SessionService
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Consent  bool   `json:"consent"`
	FullName string `json:"full_name"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if req.Role == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin accounts cannot be self-registered"})
		return
	}

	hash, _ := services.HashPassword(req.Password)

	tx, err := h.DB.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(context.Background())

	var userID uuid.UUID
	err = tx.QueryRow(context.Background(),
		`INSERT INTO users (email,password_hash,role,consent_given)
		 VALUES ($1,$2,$3,$4) RETURNING id`,
		req.Email, hash, req.Role, req.Consent,
	).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		}
		return
	}

	switch req.Role {
	case "student":
		parts := splitName(req.FullName)
		_, err = tx.Exec(context.Background(),
			`INSERT INTO student_profiles
			 (user_id,first_name,last_name,display_name,bio,university,course,year,location,avatar_url)
			 VALUES ($1,$2,$3,$4,'','','','','','')`,
			userID, parts[0], parts[1], req.FullName,
		)
	case "counselor":
		_, err = tx.Exec(context.Background(),
			`INSERT INTO counselor_profiles
			 (user_id,full_name,specialization,bio,phone)
			 VALUES ($1,$2,'','','')`,
			userID, req.FullName,
		)
	default:
		// No profile needed for other roles
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create profile"})
		return
	}

	tx.Commit(context.Background())

	c.JSON(http.StatusCreated, gin.H{"message": "Registered", "user_id": userID})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var id uuid.UUID
	var hash string

	err := h.DB.QueryRow(context.Background(),
		`SELECT id,password_hash FROM users WHERE email=$1`,
		req.Email,
	).Scan(&id, &hash)

	if err != nil || services.CheckPassword(hash, req.Password) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	sessionID, err := h.SessionService.Create(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session, please try again"})
		return
	}

	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("session_id", sessionID.String(), 3600*24, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{"message": "Logged in", "user_id": id})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	cookie, _ := c.Cookie("session_id")
	id, _ := uuid.Parse(cookie)

	h.SessionService.Delete(id)

	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("session_id", "", -1, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// splitName splits a full name into [firstName, lastName].
// If only one word is provided, lastName is an empty string.
func splitName(full string) [2]string {
	for i, ch := range full {
		if ch == ' ' {
			return [2]string{full[:i], full[i+1:]}
		}
	}
	return [2]string{full, ""}
}

func (h *AuthHandler) Profile(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)

	var role, email string
	err := h.DB.QueryRow(context.Background(),
		`SELECT role::text, email FROM users WHERE id=$1`,
		userID,
	).Scan(&role, &email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load profile"})
		return
	}

	switch role {
	case "student":
		var firstName, lastName, displayName, bio, university, course, year, location, avatarURL string
		var isAnonymous bool
		h.DB.QueryRow(context.Background(),
			`SELECT first_name, last_name, display_name, bio, university, course, year, location, avatar_url, is_anonymous
			 FROM student_profiles WHERE user_id=$1`,
			userID,
		).Scan(&firstName, &lastName, &displayName, &bio, &university, &course, &year, &location, &avatarURL, &isAnonymous)

		c.JSON(http.StatusOK, gin.H{
			"id":           userID,
			"email":        email,
			"role":         role,
			"first_name":   firstName,
			"last_name":    lastName,
			"display_name": displayName,
			"bio":          bio,
			"university":   university,
			"course":       course,
			"year":         year,
			"location":     location,
			"avatar_url":   avatarURL,
			"is_anonymous": isAnonymous,
		})

	case "counselor":
		var fullName, specialization, bio, phone string
		h.DB.QueryRow(context.Background(),
			`SELECT full_name, specialization, bio, phone FROM counselor_profiles WHERE user_id=$1`,
			userID,
		).Scan(&fullName, &specialization, &bio, &phone)

		c.JSON(http.StatusOK, gin.H{
			"id":             userID,
			"email":          email,
			"role":           role,
			"full_name":      fullName,
			"specialization": specialization,
			"bio":            bio,
			"phone":          phone,
		})

	default:
		c.JSON(http.StatusOK, gin.H{
			"id":    userID,
			"email": email,
			"role":  role,
		})
	}
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)

	var role string
	err := h.DB.QueryRow(context.Background(),
		`SELECT role::text FROM users WHERE id=$1`, userID,
	).Scan(&role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user"})
		return
	}

	switch role {
	case "student":
		var body struct {
			DisplayName *string `json:"display_name"`
			Bio         *string `json:"bio"`
			University  *string `json:"university"`
			Course      *string `json:"course"`
			Year        *string `json:"year"`
			Location    *string `json:"location"`
			AvatarURL   *string `json:"avatar_url"`
			IsAnonymous *bool   `json:"is_anonymous"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		_, err = h.DB.Exec(context.Background(),
			`UPDATE student_profiles
			 SET display_name  = COALESCE($1, display_name),
			     bio           = COALESCE($2, bio),
			     university    = COALESCE($3, university),
			     course        = COALESCE($4, course),
			     year          = COALESCE($5, year),
			     location      = COALESCE($6, location),
			     avatar_url    = COALESCE($7, avatar_url),
			     is_anonymous  = COALESCE($8, is_anonymous),
			     updated_at    = now()
			 WHERE user_id = $9`,
			body.DisplayName, body.Bio, body.University, body.Course,
			body.Year, body.Location, body.AvatarURL, body.IsAnonymous, userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}

	case "counselor":
		var body struct {
			FullName       *string `json:"full_name"`
			Specialization *string `json:"specialization"`
			Bio            *string `json:"bio"`
			Phone          *string `json:"phone"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		_, err = h.DB.Exec(context.Background(),
			`UPDATE counselor_profiles
			 SET full_name       = COALESCE($1, full_name),
			     specialization  = COALESCE($2, specialization),
			     bio             = COALESCE($3, bio),
			     phone           = COALESCE($4, phone),
			     updated_at      = now()
			 WHERE user_id = $5`,
			body.FullName, body.Specialization, body.Bio, body.Phone, userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}

	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "Profile updates not supported for this role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
}
