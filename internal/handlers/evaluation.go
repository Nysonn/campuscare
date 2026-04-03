package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Nysonn/campuscare/internal/chatbot"
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

// fallbackQuestions are used when Groq is unavailable.
var fallbackQuestions = []evaluationQuestion{
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

// scoreToCategory maps a total score (8–32) to a category label.
func scoreToCategory(score int) string {
	switch {
	case score <= 13:
		return "Needs Support"
	case score <= 19:
		return "Moderate Concern"
	case score <= 25:
		return "Doing Well"
	default:
		return "Thriving"
	}
}

// generateQuestionsFromGroq asks Groq to produce 8 fresh evaluation questions.
// Returns an error if Groq is unreachable or returns malformed JSON.
func generateQuestionsFromGroq() ([]evaluationQuestion, error) {
	prompt := `You are a mental health self-assessment tool for university students in Uganda. Generate exactly 8 unique mental health evaluation questions. Each question must have exactly 4 answer options ordered from worst to best:
- index 0 = score 1 (worst state)
- index 1 = score 2 (poor state)
- index 2 = score 3 (okay state)
- index 3 = score 4 (best state)

Cover these 8 topics in any order, with fresh and varied wording each time: sleep quality, overall mood, academic stress, social connection, ability to focus, physical activity, anxiety levels, general wellbeing.

Respond ONLY with valid JSON — no markdown, no code fences, no explanation. Exact structure required:
{"questions":[{"id":1,"text":"...","options":["worst","poor","okay","best"]},{"id":2,"text":"...","options":["...","...","...","..."]},...,{"id":8,"text":"...","options":["...","...","...","..."]}]}`

	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}

	raw, err := chatbot.CallGroq(messages, 1.0)
	if err != nil {
		return nil, fmt.Errorf("groq call failed: %w", err)
	}

	// Strip markdown code fences that some models include despite instructions.
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) > 2 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var result struct {
		Questions []evaluationQuestion `json:"questions"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("failed to parse Groq response as JSON: %w", err)
	}
	if len(result.Questions) != 8 {
		return nil, fmt.Errorf("expected 8 questions, got %d", len(result.Questions))
	}
	for _, q := range result.Questions {
		if len(q.Options) != 4 {
			return nil, fmt.Errorf("question %d has %d options, expected 4", q.ID, len(q.Options))
		}
	}

	return result.Questions, nil
}

// generatePersonalizedFeedback asks Groq to write a warm, personalised message
// based on the student's actual scores. Falls back to a static message on error.
func generatePersonalizedFeedback(score int, category string, answers map[string]int) string {
	// Build a per-question score summary for context.
	var sb strings.Builder
	for i := 1; i <= 8; i++ {
		key := fmt.Sprintf("%d", i)
		if s, ok := answers[key]; ok {
			sb.WriteString(fmt.Sprintf("  Question %d: %d/4\n", i, s))
		}
	}

	prompt := fmt.Sprintf(`A university student in Uganda just completed a mental health self-evaluation.

Results:
- Total score: %d / 32
- Category: %s
- Per-question scores (1=worst, 4=best):
%s
Write a warm, empathetic, personalised 2–3 sentence message addressed directly to this student. Acknowledge their current state honestly but compassionately, highlight something specific from their scores, and offer one constructive encouragement. Do not give medical advice. Output only the message — no labels, no prefixes.`,
		score, category, sb.String(),
	)

	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}

	feedback, err := chatbot.CallGroq(messages)
	if err != nil {
		// Static fallback per category.
		switch category {
		case "Needs Support":
			return "You may be going through a really difficult time right now. You are not alone — reaching out to a counselor or someone you trust is a brave and important step."
		case "Moderate Concern":
			return "You are facing some challenges and that is okay. Small acts of self-care and talking to someone can make a meaningful difference. You are doing better than you think."
		case "Doing Well":
			return "You are managing reasonably well. Keep nurturing your wellbeing through rest, connection, and balance. Do not hesitate to seek support if things get harder."
		default:
			return "You are in a great place! Your habits and mindset are working for you. Keep it up and continue showing up for yourself every day."
		}
	}

	return strings.TrimSpace(feedback)
}

// GetQuestions — GET /evaluations/questions
// Uses Groq to generate 8 fresh evaluation questions on every request.
// Falls back to the static question set if Groq is unavailable.
func (h *EvaluationHandler) GetQuestions(c *gin.Context) {
	questions, err := generateQuestionsFromGroq()
	if err != nil {
		// Graceful degradation — serve static questions.
		c.JSON(http.StatusOK, gin.H{"questions": fallbackQuestions})
		return
	}
	c.JSON(http.StatusOK, gin.H{"questions": questions})
}

// SubmitEvaluation — POST /evaluations
// Accepts answers (map of question ID → score 1–4), calculates the result,
// generates a personalised AI feedback message, and saves the record.
func (h *EvaluationHandler) SubmitEvaluation(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		Answers map[string]int `json:"answers"` // "1" -> 3, "2" -> 2, etc.
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if len(req.Answers) != 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please answer all 8 questions before submitting"})
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

	category := scoreToCategory(total)
	message := generatePersonalizedFeedback(total, category, req.Answers)

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
