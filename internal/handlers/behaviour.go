package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BehaviourHandler struct {
	DB *pgxpool.Pool
}

// CreateGoal — POST /behaviour/goals
// Creates a new behaviour goal. Only one active goal is allowed at a time.
func (h *BehaviourHandler) CreateGoal(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		Title     string `json:"title"`
		Direction string `json:"direction"` // "build" | "quit"
		StartDate string `json:"start_date"` // YYYY-MM-DD
		EndDate   string `json:"end_date"`   // YYYY-MM-DD
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if req.Title == "" || req.Direction == "" || req.StartDate == "" || req.EndDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "All fields are required"})
		return
	}
	if req.Direction != "build" && req.Direction != "quit" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Direction must be 'build' or 'quit'"})
		return
	}

	// Only one active goal at a time
	var count int
	if err := h.DB.QueryRow(c,
		`SELECT COUNT(*) FROM behaviour_goals WHERE student_id = $1 AND status = 'active'`,
		userID,
	).Scan(&count); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "You already have an active behaviour goal. Complete it before starting a new one."})
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format, use YYYY-MM-DD"})
		return
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format, use YYYY-MM-DD"})
		return
	}
	if !endDate.After(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date must be after start date"})
		return
	}

	var goalID uuid.UUID
	if err := h.DB.QueryRow(c,
		`INSERT INTO behaviour_goals (student_id, title, direction, start_date, end_date)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		userID, req.Title, req.Direction, startDate, endDate,
	).Scan(&goalID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create goal"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Behaviour goal created", "id": goalID})
}

// GetCurrentGoal — GET /behaviour/goals/current
// Returns the student's active goal and all its daily logs.
func (h *BehaviourHandler) GetCurrentGoal(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	type Log struct {
		LogDate string `json:"log_date"`
		DidIt   bool   `json:"did_it"`
	}
	type Goal struct {
		ID        uuid.UUID `json:"id"`
		Title     string    `json:"title"`
		Direction string    `json:"direction"`
		StartDate string    `json:"start_date"`
		EndDate   string    `json:"end_date"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
		Logs      []Log     `json:"logs"`
	}

	var g Goal
	err := h.DB.QueryRow(c,
		`SELECT id, title, direction, start_date::text, end_date::text, status, created_at
		 FROM behaviour_goals
		 WHERE student_id = $1 AND status = 'active'
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID,
	).Scan(&g.ID, &g.Title, &g.Direction, &g.StartDate, &g.EndDate, &g.Status, &g.CreatedAt)
	if err != nil {
		// No active goal found — return null, not an error
		c.JSON(http.StatusOK, gin.H{"goal": nil})
		return
	}

	rows, err := h.DB.Query(c,
		`SELECT log_date::text, did_it FROM behaviour_logs WHERE goal_id = $1 ORDER BY log_date`,
		g.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load logs"})
		return
	}
	defer rows.Close()

	g.Logs = []Log{}
	for rows.Next() {
		var l Log
		if err := rows.Scan(&l.LogDate, &l.DidIt); err != nil {
			continue
		}
		g.Logs = append(g.Logs, l)
	}

	c.JSON(http.StatusOK, gin.H{"goal": g})
}

// GetAllGoals — GET /behaviour/goals
// Returns all goals (active and completed) for history view.
func (h *BehaviourHandler) GetAllGoals(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	rows, err := h.DB.Query(c,
		`SELECT id, title, direction, start_date::text, end_date::text, status, created_at
		 FROM behaviour_goals
		 WHERE student_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load goals"})
		return
	}
	defer rows.Close()

	type Goal struct {
		ID        uuid.UUID `json:"id"`
		Title     string    `json:"title"`
		Direction string    `json:"direction"`
		StartDate string    `json:"start_date"`
		EndDate   string    `json:"end_date"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
	}

	goals := []Goal{}
	for rows.Next() {
		var g Goal
		if err := rows.Scan(&g.ID, &g.Title, &g.Direction, &g.StartDate, &g.EndDate, &g.Status, &g.CreatedAt); err != nil {
			continue
		}
		goals = append(goals, g)
	}

	c.JSON(http.StatusOK, gin.H{"goals": goals})
}

// LogDay — POST /behaviour/goals/:id/logs
// Upserts a daily log entry (did_it true/false) for a given date.
func (h *BehaviourHandler) LogDay(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	goalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid goal ID"})
		return
	}

	var req struct {
		LogDate string `json:"log_date"` // YYYY-MM-DD
		DidIt   bool   `json:"did_it"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	logDate, err := time.Parse("2006-01-02", req.LogDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log_date format, use YYYY-MM-DD"})
		return
	}

	// Verify goal belongs to this student and is active
	var count int
	if err := h.DB.QueryRow(c,
		`SELECT COUNT(*) FROM behaviour_goals
		 WHERE id = $1 AND student_id = $2 AND status = 'active'`,
		goalID, userID,
	).Scan(&count); err != nil || count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Active goal not found"})
		return
	}

	if _, err := h.DB.Exec(c,
		`INSERT INTO behaviour_logs (goal_id, log_date, did_it)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (goal_id, log_date) DO UPDATE SET did_it = EXCLUDED.did_it`,
		goalID, logDate, req.DidIt,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log day"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Day logged"})
}

// CompleteGoal — POST /behaviour/goals/:id/complete
// Marks an active goal as completed.
func (h *BehaviourHandler) CompleteGoal(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	goalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid goal ID"})
		return
	}

	res, err := h.DB.Exec(c,
		`UPDATE behaviour_goals
		 SET status = 'completed', updated_at = now()
		 WHERE id = $1 AND student_id = $2 AND status = 'active'`,
		goalID, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete goal"})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Active goal not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Goal marked as completed"})
}

// GetGoalStats — GET /behaviour/goals/:id/stats
// Returns statistics (total days, logged, succeeded, success rate) for any goal.
func (h *BehaviourHandler) GetGoalStats(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	goalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid goal ID"})
		return
	}

	var title, direction, startDate, endDate, status string
	if err := h.DB.QueryRow(c,
		`SELECT title, direction, start_date::text, end_date::text, status
		 FROM behaviour_goals WHERE id = $1 AND student_id = $2`,
		goalID, userID,
	).Scan(&title, &direction, &startDate, &endDate, &status); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Goal not found"})
		return
	}

	var totalDays, daysLogged, daysSucceeded int
	if err := h.DB.QueryRow(c,
		`SELECT
		   (bg.end_date - bg.start_date + 1)::int             AS total_days,
		   COUNT(bl.id)::int                                   AS days_logged,
		   COALESCE(SUM(CASE WHEN bl.did_it THEN 1 ELSE 0 END), 0)::int AS days_succeeded
		 FROM behaviour_goals bg
		 LEFT JOIN behaviour_logs bl ON bl.goal_id = bg.id
		 WHERE bg.id = $1
		 GROUP BY bg.start_date, bg.end_date`,
		goalID,
	).Scan(&totalDays, &daysLogged, &daysSucceeded); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate stats"})
		return
	}

	successRate := 0.0
	if daysLogged > 0 {
		successRate = float64(daysSucceeded) / float64(daysLogged) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"title":          title,
		"direction":      direction,
		"start_date":     startDate,
		"end_date":       endDate,
		"status":         status,
		"total_days":     totalDays,
		"days_logged":    daysLogged,
		"days_succeeded": daysSucceeded,
		"success_rate":   successRate,
	})
}
