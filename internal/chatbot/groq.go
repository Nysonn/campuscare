package chatbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func CallGroq(messages []map[string]string, temperature ...float64) (string, error) {
	temp := 0.7
	if len(temperature) > 0 {
		temp = temperature[0]
	}

	body := map[string]interface{}{
		"model":       "llama-3.3-70b-versatile",
		"messages":    messages,
		"temperature": temp,
	}

	b, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST",
		"https://api.groq.com/openai/v1/chat/completions",
		bytes.NewBuffer(b),
	)

	req.Header.Set("Authorization", "Bearer "+os.Getenv("GROQ_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("groq: failed to decode response: %w", err)
	}

	// Surface API-level errors (auth failure, rate limit, etc.)
	if errObj, ok := result["error"]; ok {
		if errMap, ok := errObj.(map[string]interface{}); ok {
			return "", fmt.Errorf("groq API error: %v", errMap["message"])
		}
		return "", fmt.Errorf("groq API error: %v", errObj)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("groq: no choices in response: %v", result)
	}

	msg, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("groq: unexpected message structure")
	}

	content, ok := msg["content"].(string)
	if !ok {
		return "", fmt.Errorf("groq: content is not a string")
	}

	return content, nil
}
