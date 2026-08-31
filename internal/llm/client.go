package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	token    string
	model    string
	endpoint string
}

func NewClient(token string) *Client {
	return &Client{token: token, model: "openai/gpt-oss-120b", endpoint: "https://api.groq.com/openai/v1/chat/completions"}
}

func (c *Client) Complete(system string, user string) (string, error) {
	payload := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"max_tokens":  1024,
		"temperature": 0.2,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm request failed: status %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("could not parse llm response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llm response contained no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}
