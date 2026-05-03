package chatbot

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AskPublic handles a single unauthenticated chatbot query from a landing-page visitor.
// No history is loaded or stored — this is a stateless, one-shot interaction.
func (h *ChatbotHandler) AskPublic(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	// Crisis pre-check — same safety net as the authenticated flow.
	if DetectCrisis(req.Message) {
		c.JSON(http.StatusOK, gin.H{
			"reply":          CrisisResponse(),
			"crisis_flagged": true,
		})
		return
	}

	responseMode := DecideResponseMode(req.Message)

	messages := []map[string]string{
		{"role": "system", "content": SystemPrompt},
		{"role": "system", "content": BuildModeInstruction(responseMode)},
		{"role": "user", "content": req.Message},
	}

	reply, err := CallGroq(messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get response"})
		return
	}

	// Post-moderation: sanitise the reply if it accidentally triggers crisis logic.
	if DetectCrisis(reply) {
		reply = SafeRewrite(reply)
	}

	c.JSON(http.StatusOK, gin.H{
		"reply":          reply,
		"crisis_flagged": false,
	})
}
