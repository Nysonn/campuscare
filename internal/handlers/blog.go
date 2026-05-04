package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BlogHandler struct {
	DB *pgxpool.Pool
}

type Blog struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	ImageURL    string    `json:"image_url"`
	Author      string    `json:"author"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateBlogRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	ImageURL    string `json:"image_url"`
}

// ListBlogs — public, returns all blogs newest-first.
func (h *BlogHandler) ListBlogs(c *gin.Context) {
	rows, err := h.DB.Query(context.Background(),
		`SELECT id, title, description, content, image_url, author, created_at, updated_at
		 FROM blogs
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch blogs"})
		return
	}
	defer rows.Close()

	var blogs []Blog
	for rows.Next() {
		var b Blog
		if err := rows.Scan(&b.ID, &b.Title, &b.Description, &b.Content,
			&b.ImageURL, &b.Author, &b.CreatedAt, &b.UpdatedAt); err != nil {
			continue
		}
		blogs = append(blogs, b)
	}

	if blogs == nil {
		blogs = []Blog{}
	}

	c.JSON(http.StatusOK, blogs)
}

// GetBlog — public, returns a single blog by id.
func (h *BlogHandler) GetBlog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blog ID"})
		return
	}

	var b Blog
	err = h.DB.QueryRow(context.Background(),
		`SELECT id, title, description, content, image_url, author, created_at, updated_at
		 FROM blogs WHERE id = $1`, id,
	).Scan(&b.ID, &b.Title, &b.Description, &b.Content,
		&b.ImageURL, &b.Author, &b.CreatedAt, &b.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
		return
	}

	c.JSON(http.StatusOK, b)
}

// CreateBlog — admin only.
func (h *BlogHandler) CreateBlog(c *gin.Context) {
	var req CreateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if req.Title == "" || req.Description == "" || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title, description and content are required"})
		return
	}

	var b Blog
	err := h.DB.QueryRow(context.Background(),
		`INSERT INTO blogs (title, description, content, image_url, author)
		 VALUES ($1, $2, $3, $4, 'CampusCare Team')
		 RETURNING id, title, description, content, image_url, author, created_at, updated_at`,
		req.Title, req.Description, req.Content, req.ImageURL,
	).Scan(&b.ID, &b.Title, &b.Description, &b.Content,
		&b.ImageURL, &b.Author, &b.CreatedAt, &b.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create blog"})
		return
	}

	c.JSON(http.StatusCreated, b)
}

// DeleteBlog — admin only.
func (h *BlogHandler) DeleteBlog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blog ID"})
		return
	}

	cmd, err := h.DB.Exec(context.Background(), `DELETE FROM blogs WHERE id = $1`, id)
	if err != nil || cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Blog deleted"})
}

// UpdateBlog — admin only.
func (h *BlogHandler) UpdateBlog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blog ID"})
		return
	}

	var req CreateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if req.Title == "" || req.Description == "" || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title, description and content are required"})
		return
	}

	var b Blog
	err = h.DB.QueryRow(context.Background(),
		`UPDATE blogs
		 SET title=$1, description=$2, content=$3, image_url=$4, updated_at=now()
		 WHERE id=$5
		 RETURNING id, title, description, content, image_url, author, created_at, updated_at`,
		req.Title, req.Description, req.Content, req.ImageURL, id,
	).Scan(&b.ID, &b.Title, &b.Description, &b.Content,
		&b.ImageURL, &b.Author, &b.CreatedAt, &b.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog not found or update failed"})
		return
	}

	c.JSON(http.StatusOK, b)
}
