package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EvaluationHandler struct {
	DB *pgxpool.Pool
}

type evaluationQuestion struct {
	ID      int      `json:"id"`
	Text    string   `json:"text"`
	Options []string `json:"options"` // index 0 = score 1 (worst), index 3 = score 4 (best)
}

var evaluationQuestions = []evaluationQuestion{
	{
		ID:   1,
		Text: "How would you rate your sleep quality over the past week?",
		Options: []string{
			"Very poor — I barely slept or woke up constantly",
			"Poor — I often had trouble sleeping",
			"Fair — My sleep was okay most nights",
			"Good — I slept well most nights",
		},
	},
	{
		ID:   2,
		Text: "How would you describe your overall mood this week?",
		Options: []string{
			"Very low — I felt sad or empty most of the time",
			"Low — I struggled to feel positive",
			"Moderate — My mood had ups and downs",
			"Good — I felt generally positive and upbeat",
		},
	},
	{
		ID:   3,
		Text: "How stressed have you been with your academic workload?",
		Options: []string{
			"Overwhelmed — I feel I cannot cope",
			"Very stressed — It is affecting my daily life",
			"Somewhat stressed — I am managing but it is tough",
			"Managing well — I feel in control of my studies",
		},
	},
	{
		ID:   4,
		Text: "How connected do you feel to friends and family?",
		Options: []string{
			"Very isolated — I feel completely alone",
			"Somewhat isolated — I rarely connect with others",
			"Somewhat connected — I have some social interaction",
			"Well connected — I feel supported by people around me",
		},
	},
	{
		ID:   5,
		Text: "How well have you been able to focus on tasks this week?",
		Options: []string{
			"Cannot focus — My mind is constantly distracted",
			"Struggle to focus — I get very little done",
			"Moderate focus — I can focus with some effort",
			"Good focus — I concentrate and stay on task well",
		},
	},
	{
		ID:   6,
		Text: "How physically active have you been this week?",
		Options: []string{
			"Not active at all — I have been mostly sedentary",
			"Slightly active — I moved a little",
			"Moderately active — I had some exercise or walks",
			"Very active — I exercised regularly",
		},
	},
	{
		ID:   7,
		Text: "How often have you felt anxious or worried this week?",
		Options: []string{
			"Almost always — I feel anxious most of the time",
			"Often — Anxiety is frequently on my mind",
			"Sometimes — I get anxious but it passes",
			"Rarely — I feel mostly calm and at ease",
		},
	},
	{
		ID:   8,
		Text: "Overall, how would you rate your sense of wellbeing right now?",
		Options: []string{
			"Very poor — I am really struggling",
			"Poor — Things do not feel right",
			"Fair — I am getting by day to day",
			"Good — I feel well overall",
		},
	},
}

// scoreToCategory maps a total score (8–32) to a category label and message.
func scoreToCategory(score int) (string, string) {
	switch {
	case score <= 13:
		return "Needs Support",
			"You may be going through a really difficult time right now. You are not alone — reaching out to a counselor or someone you trust is a brave and important step."
	case score <= 19:
		return "Moderate Concern",
			"You are facing some challenges and that is okay. Small acts of self-care and talking to someone can make a meaningful difference. You are doing better than you think."
	case score <= 25:
		return "Doing Well",
			"You are managing reasonably well. Keep nurturing your wellbeing through rest, connection, and balance. Do not hesitate to seek support if things get harder."
	default:
		return "Thriving",
			"You are in a great place! Your habits and mindset are working for you. Keep it up and continue showing up for yourself every day."
	}
}

// GetQuestions — GET /evaluations/questions
// Returns the list of evaluation questions (static, same for every request).
func (h *EvaluationHandler) GetQuestions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"questions": evaluationQuestions})
}

// SubmitEvaluation — POST /evaluations
// Accepts answers (map of question ID → score 1–4), calculates result, and saves it.
func (h *EvaluationHandler) SubmitEvaluation(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		Answers map[string]int `json:"answers"` // "1" -> 3, "2" -> 2, etc.
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if len(req.Answers) != len(evaluationQuestions) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please answer all questions before submitting"})
		return
	}

	total := 0
	for _, v := range req.Answers {
		if v < 1 || v > 4 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Each answer must be between 1 and 4"})
			return
		}
		total += v
	}

	category, message := scoreToCategory(total)

	var evalID uuid.UUID
	if err := h.DB.QueryRow(c,
		`INSERT INTO self_evaluations (student_id, score, category, answers)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		userID, total, category, req.Answers,
	).Scan(&evalID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save evaluation"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       evalID,
		"score":    total,
		"category": category,
		"message":  message,
	})
}

// GetHistory — GET /evaluations/history
// Returns the last 20 evaluations for the authenticated student.
func (h *EvaluationHandler) GetHistory(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	rows, err := h.DB.Query(c,
		`SELECT id, score, category, taken_at
		 FROM self_evaluations
		 WHERE student_id = $1
		 ORDER BY taken_at DESC
		 LIMIT 20`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load history"})
		return
	}
	defer rows.Close()

	type EvalSummary struct {
		ID       uuid.UUID `json:"id"`
		Score    int       `json:"score"`
		Category string    `json:"category"`
		TakenAt  time.Time `json:"taken_at"`
	}

	history := []EvalSummary{}
	for rows.Next() {
		var e EvalSummary
		if err := rows.Scan(&e.ID, &e.Score, &e.Category, &e.TakenAt); err != nil {
			continue
		}
		history = append(history, e)
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}
