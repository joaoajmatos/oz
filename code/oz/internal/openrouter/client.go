// Package openrouter provides an HTTP client for the OpenRouter AI gateway.
// OpenRouter exposes an OpenAI-compatible chat completions API that supports
// models from multiple providers via a single endpoint.
//
// Authentication requires the OPENROUTER_API_KEY environment variable.
// The default model is anthropic/claude-3.5-haiku.
package openrouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// DefaultBaseURL is the OpenRouter API base URL.
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// DefaultModel is used when no model is specified.
const DefaultModel = "anthropic/claude-3.5-haiku"

// Client calls the OpenRouter chat completions endpoint.
type Client struct {
	APIKey  string
	BaseURL string
	Model   string
	HTTP    *http.Client
}

// New returns a Client configured from the environment.
// Returns an error if OPENROUTER_API_KEY is not set.
// model may be empty to use DefaultModel.
func New(model string) (*Client, error) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is not set")
	}
	if model == "" {
		model = DefaultModel
	}
	return &Client{
		APIKey:  key,
		BaseURL: DefaultBaseURL,
		Model:   model,
		HTTP:    &http.Client{},
	}, nil
}

// Message is a single chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest is the request body sent to the completions endpoint.
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// ChatResponse is the response from the completions endpoint.
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	// Usage is populated by OpenRouter when cost information is available.
	Usage *struct {
		Cost float64 `json:"cost"`
	} `json:"usage"`
}

// Complete sends a chat completion request and returns the response.
func (c *Client) Complete(messages []Message) (*ChatResponse, error) {
	body, err := json.Marshal(chatRequest{
		Model:    c.Model,
		Messages: messages,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/joaoajmatos/oz")
	req.Header.Set("X-Title", "oz context enrich")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	return &chatResp, nil
}
