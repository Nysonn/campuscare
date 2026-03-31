package stream

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://chat.stream-io-api.com"

// Client wraps the Stream Chat REST API with JWT authentication.
type Client struct {
	APIKey    string
	APISecret string
	http      *http.Client
}

func NewClient(apiKey, apiSecret string) *Client {
	return &Client{
		APIKey:    apiKey,
		APISecret: apiSecret,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

// GenerateUserToken creates a Stream Chat JWT for a given user ID.
// The frontend passes this token to the Stream Chat JS SDK to authenticate.
func (c *Client) GenerateUserToken(userID string) (string, error) {
	header := b64url([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(map[string]any{"user_id": userID})
	if err != nil {
		return "", err
	}
	msg := header + "." + b64url(payload)
	mac := hmac.New(sha256.New, []byte(c.APISecret))
	mac.Write([]byte(msg))
	return msg + "." + b64url(mac.Sum(nil)), nil
}

// serverToken returns a short-lived server-side JWT for REST API calls.
func (c *Client) serverToken() string {
	header := b64url([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{
		"server": true,
		"iat":    time.Now().Unix(),
	})
	msg := header + "." + b64url(payload)
	mac := hmac.New(sha256.New, []byte(c.APISecret))
	mac.Write([]byte(msg))
	return msg + "." + b64url(mac.Sum(nil))
}

// UpsertUsers ensures Stream Chat has user records for the given IDs before
// channel creation. names maps user_id → display name.
func (c *Client) UpsertUsers(userIDs []string, names map[string]string) error {
	usersMap := make(map[string]any, len(userIDs))
	for _, id := range userIDs {
		entry := map[string]any{"id": id}
		if name, ok := names[id]; ok && name != "" {
			entry["name"] = name
		}
		usersMap[id] = entry
	}

	body, _ := json.Marshal(map[string]any{"users": usersMap})
	return c.do("POST", "/users", body)
}

// CreateChannel creates a private messaging channel between two users.
func (c *Client) CreateChannel(channelID, creatorID string, memberIDs []string) error {
	body, _ := json.Marshal(map[string]any{
		"created_by_id": creatorID,
		"data": map[string]any{
			"members": memberIDs,
		},
	})
	return c.do("POST", fmt.Sprintf("/channels/messaging/%s", channelID), body)
}

// DeleteChannel soft-deletes a Stream Chat channel when a sponsorship ends.
func (c *Client) DeleteChannel(channelID string) error {
	return c.do("DELETE", fmt.Sprintf("/channels/messaging/%s", channelID), nil)
}

// do executes an authenticated request against the Stream Chat REST API.
func (c *Client) do(method, path string, body []byte) error {
	url := fmt.Sprintf("%s%s?api_key=%s", baseURL, path, c.APIKey)
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.serverToken())
	req.Header.Set("Stream-Auth-Type", "jwt")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("stream API %s %s → %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// b64url returns standard base64url encoding without padding.
func b64url(data []byte) string {
	s := base64.StdEncoding.EncodeToString(data)
	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "/", "_")
	return strings.TrimRight(s, "=")
}
