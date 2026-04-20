package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Nysonn/campuscare/internal/mail"
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
	Mailer         *mail.Mailer
	FrontendURL    string
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Consent  bool   `json:"consent"`
	FullName string `json:"full_name"`
	// Counsellor-only fields
	Location           string `json:"location"`
	Age                *int   `json:"age"`
	YearsOfExperience  string `json:"years_of_experience"`
	LicenceURL         string `json:"licence_url"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	req.Email = normalizeEmail(req.Email)
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	req.FullName = strings.TrimSpace(req.FullName)

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
		age := req.Age // may be nil
		_, err = tx.Exec(context.Background(),
			`INSERT INTO counselor_profiles
			 (user_id,full_name,specialization,bio,phone,location,age,years_of_experience,licence_url,verification_status)
			 VALUES ($1,$2,'','','',$3,$4,$5,$6,'pending')`,
			userID, req.FullName,
			req.Location, age, req.YearsOfExperience, req.LicenceURL,
		)
	default:
		// No profile needed for other roles
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create profile"})
		return
	}

	tx.Commit(context.Background())

	h.Mailer.SendAsync(
		req.Email,
		"Welcome to CampusCare!",
		mail.WelcomeTemplate(req.FullName, req.Role),
	)

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

	req.Email = normalizeEmail(req.Email)

	var id uuid.UUID
	var hash string
	var role string

	err := h.DB.QueryRow(context.Background(),
		`SELECT id,password_hash,role::text FROM users WHERE lower(email)=lower($1)`,
		req.Email,
	).Scan(&id, &hash, &role)

	if err != nil || services.CheckPassword(hash, req.Password) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Block counsellors who have not yet been approved by an admin.
	if role == "counselor" {
		var verificationStatus string
		h.DB.QueryRow(context.Background(),
			`SELECT verification_status FROM counselor_profiles WHERE user_id=$1`,
			id,
		).Scan(&verificationStatus)

		if verificationStatus != "approved" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Your account is pending admin verification. You will receive an email once approved.",
			})
			return
		}
	}

	sessionID, err := h.SessionService.Create(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session, please try again"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged in", "user_id": id, "token": sessionID.String()})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	raw := strings.TrimPrefix(authHeader, "Bearer ")
	if id, err := uuid.Parse(raw); err == nil {
		h.SessionService.Delete(id)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Email) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}
	req.Email = normalizeEmail(req.Email)

	// Always respond 200 so we don't leak whether the email exists.
	c.JSON(http.StatusOK, gin.H{"message": "If that email is registered you will receive a reset link shortly"})

	go func() {
		var userID uuid.UUID
		var name string
		err := h.DB.QueryRow(context.Background(),
			`SELECT u.id, COALESCE(sp.display_name, cp.full_name, u.email)
			 FROM users u
			 LEFT JOIN student_profiles  sp ON sp.user_id = u.id
			 LEFT JOIN counselor_profiles cp ON cp.user_id = u.id
			 WHERE u.email = $1 AND u.deleted_at IS NULL`,
			req.Email,
		).Scan(&userID, &name)
		if err != nil {
			return // user not found — silently drop
		}

		// Generate a 32-byte cryptographically random token.
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			log.Printf("[forgot-password] token generation failed: %v", err)
			return
		}
		token := hex.EncodeToString(raw)

		_, err = h.DB.Exec(context.Background(),
			`INSERT INTO password_reset_tokens (user_id, token, expires_at)
			 VALUES ($1, $2, $3)`,
			userID, token, time.Now().Add(time.Hour),
		)
		if err != nil {
			log.Printf("[forgot-password] DB insert failed: %v", err)
			return
		}

		resetLink := h.FrontendURL + "/reset-password?token=" + token
		h.Mailer.SendAsync(req.Email, "Reset your CampusCare password",
			mail.PasswordResetTemplate(name, resetLink))
	}()
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if strings.TrimSpace(req.Token) == "" || len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token and a password of at least 8 characters are required"})
		return
	}

	var tokenID uuid.UUID
	var userID uuid.UUID
	var expiresAt time.Time
	var usedAt *time.Time

	err := h.DB.QueryRow(context.Background(),
		`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens WHERE token = $1`,
		req.Token,
	).Scan(&tokenID, &userID, &expiresAt, &usedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset link"})
		return
	}
	if usedAt != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This reset link has already been used"})
		return
	}
	if time.Now().After(expiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This reset link has expired. Please request a new one"})
		return
	}

	hash, err := services.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	tx, err := h.DB.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}
	defer tx.Rollback(context.Background())

	if _, err = tx.Exec(context.Background(),
		`UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`,
		hash, userID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	if _, err = tx.Exec(context.Background(),
		`UPDATE password_reset_tokens SET used_at = now() WHERE id = $1`, tokenID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to invalidate token"})
		return
	}

	// Invalidate all existing sessions so the user must log in with the new password.
	tx.Exec(context.Background(),
		`DELETE FROM sessions WHERE user_id = $1`, userID,
	)

	tx.Commit(context.Background())
	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

// splitName splits a full name into [firstName, lastName].
// If only one word is provided, lastName is an empty string.
func splitName(full string) [2]string {
	full = strings.TrimSpace(full)
	for i, ch := range full {
		if ch == ' ' {
			return [2]string{full[:i], full[i+1:]}
		}
	}
	return [2]string{full, ""}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
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

		var isSponsor bool
		h.DB.QueryRow(context.Background(),
			`SELECT COALESCE(is_active, false) FROM sponsor_profiles WHERE user_id=$1`,
			userID,
		).Scan(&isSponsor)

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
			"is_sponsor":   isSponsor,
		})

	case "counselor":
		var fullName, specialization, bio, phone, avatarURL string
		var location, yearsOfExperience, licenceURL, verificationStatus string
		var age *int
		h.DB.QueryRow(context.Background(),
			`SELECT full_name, specialization, bio, phone, avatar_url,
			        COALESCE(location,''), age,
			        COALESCE(years_of_experience,''), COALESCE(licence_url,''),
			        COALESCE(verification_status,'pending')
			 FROM counselor_profiles WHERE user_id=$1`,
			userID,
		).Scan(&fullName, &specialization, &bio, &phone, &avatarURL,
			&location, &age, &yearsOfExperience, &licenceURL, &verificationStatus)

		c.JSON(http.StatusOK, gin.H{
			"id":                  userID,
			"email":               email,
			"role":                role,
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

		// Sponsors cannot go anonymous — their visibility is required for the
		// sponsorship feature to work correctly.
		if body.IsAnonymous != nil && *body.IsAnonymous {
			var isSponsor bool
			h.DB.QueryRow(context.Background(),
				`SELECT EXISTS(SELECT 1 FROM sponsor_profiles WHERE user_id=$1 AND is_active=true)`,
				userID,
			).Scan(&isSponsor)
			if isSponsor {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Sponsors cannot use anonymous mode. Please opt out of being a sponsor first."})
				return
			}
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
			FullName          *string `json:"full_name"`
			Specialization    *string `json:"specialization"`
			Bio               *string `json:"bio"`
			Phone             *string `json:"phone"`
			AvatarURL         *string `json:"avatar_url"`
			Location          *string `json:"location"`
			YearsOfExperience *string `json:"years_of_experience"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		_, err = h.DB.Exec(context.Background(),
			`UPDATE counselor_profiles
			 SET full_name           = COALESCE($1, full_name),
			     specialization      = COALESCE($2, specialization),
			     bio                 = COALESCE($3, bio),
			     phone               = COALESCE($4, phone),
			     avatar_url          = COALESCE($5, avatar_url),
			     location            = COALESCE($6, location),
			     years_of_experience = COALESCE($7, years_of_experience),
			     updated_at          = now()
			 WHERE user_id = $8`,
			body.FullName, body.Specialization, body.Bio, body.Phone, body.AvatarURL,
			body.Location, body.YearsOfExperience, userID,
		)
		if err != nil {
			log.Printf("[UpdateProfile] counselor DB error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}

	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "Profile updates not supported for this role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
}
