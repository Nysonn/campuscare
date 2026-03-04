package chatbot

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	DB *pgxpool.Pool
}

type ChatbotHandler struct {
	Service *Service
}

type ChatRequest struct {
	Message string `json:"message"`
}

func CrisisResponse() string {
	return `
I'm really sorry you're feeling this way.
You’re not alone, and help is available.

Please consider contacting:
- Campus counselor immediately
- Local emergency services
- A trusted friend or family member

If you are in immediate danger, call emergency services right now.

Would you like help booking a session with a counselor?
`
}

func (s *Service) Ask(userID uuid.UUID, message string) (string, bool, error) {

	// 1️⃣ Crisis pre-check
	if DetectCrisis(message) {
		return CrisisResponse(), true, nil
	}

	// 2️⃣ Load history
	history, err := GetRecentMessages(s.DB, userID, 10)
	if err != nil {
		return "", false, err
	}

	// 3️⃣ Build messages
	messages := []map[string]string{
		{"role": "system", "content": SystemPrompt},
	}

	messages = append(messages, history...)
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": message,
	})

	// 4️⃣ Call Groq
	reply, err := CallGroq(messages)
	if err != nil {
		return "", false, err
	}

	// 5️⃣ Post-check moderation
	if DetectCrisis(reply) {
		reply = SafeRewrite(reply)
	}

	// 6️⃣ Store conversation
	StoreMessage(s.DB, userID, "user", message)
	StoreMessage(s.DB, userID, "assistant", reply)

	return reply, false, nil
}

func (h *ChatbotHandler) Ask(c *gin.Context) {

	userID := c.MustGet("user_id").(uuid.UUID)

	var req ChatRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid"})
		return
	}

	reply, crisis, err := h.Service.Ask(userID, req.Message)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"reply":          reply,
		"crisis_flagged": crisis,
	})
}
