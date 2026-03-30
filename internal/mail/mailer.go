package mail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Mailer struct {
	APIKey string
	From   string
	client *http.Client
}

func NewMailer() *Mailer {
	from := strings.TrimSpace(os.Getenv("RESEND_FROM"))
	if from == "" {
		from = "CampusCare <onboarding@resend.dev>"
	}

	return &Mailer{
		APIKey: strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
		From:   from,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (m *Mailer) Send(to, subject, body string) error {
	if m == nil {
		return fmt.Errorf("mailer is nil")
	}
	if m.APIKey == "" {
		return fmt.Errorf("resend api key is not configured")
	}
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient email is required")
	}

	payload, err := json.Marshal(resendPayload{
		From:    m.From,
		To:      []string{to},
		Subject: subject,
		HTML:    body,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("resend: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return nil
}
