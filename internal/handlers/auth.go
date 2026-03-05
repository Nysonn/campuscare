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
		`INSERT INTO users (full_name,email,password_hash,role,consent_given)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		req.FullName, req.Email, hash, req.Role, req.Consent,
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
	c.BindJSON(&req)

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

	sessionID, _ := h.SessionService.Create(id)

	c.SetCookie("session_id", sessionID.String(), 3600*24, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "Logged in", "user_id": id})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	cookie, _ := c.Cookie("session_id")
	id, _ := uuid.Parse(cookie)

	h.SessionService.Delete(id)

	c.SetCookie("session_id", "", -1, "/", "", false, true)

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
