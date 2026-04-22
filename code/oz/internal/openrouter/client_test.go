package openrouter_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/openrouter"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClient_Complete(t *testing.T) {
	wantContent := "extracted concepts JSON here"

	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth: %s", r.Header.Get("Authorization"))
		}

		// Decode the request to verify model is forwarded.
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("model = %q, want %q", req.Model, "test-model")
		}

		respBody, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": wantContent}},
			},
			"usage": map[string]any{"cost": 0.0012},
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(respBody))),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})

	c := &openrouter.Client{
		APIKey:  "test-key",
		BaseURL: "http://example.test",
		Model:   "test-model",
		HTTP:    &http.Client{Transport: rt},
	}

	resp, err := c.Complete([]openrouter.Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Choices[0].Message.Content != wantContent {
		t.Errorf("content = %q, want %q", resp.Choices[0].Message.Content, wantContent)
	}
	if resp.Usage == nil || resp.Usage.Cost != 0.0012 {
		t.Errorf("usage cost = %v, want 0.0012", resp.Usage)
	}
}

func TestClient_APIError(t *testing.T) {
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{"error":"invalid api key"}`)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})

	c := &openrouter.Client{
		APIKey:  "bad-key",
		BaseURL: "http://example.test",
		Model:   "test-model",
		HTTP:    &http.Client{Transport: rt},
	}

	_, err := c.Complete([]openrouter.Message{{Role: "user", Content: "hello"}})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestNew_MissingAPIKey(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	_, err := openrouter.New("")
	if err == nil {
		t.Fatal("expected error when OPENROUTER_API_KEY is not set")
	}
}

func TestNew_DefaultModel(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test")
	c, err := openrouter.New("")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.Model != openrouter.DefaultModel {
		t.Errorf("model = %q, want %q", c.Model, openrouter.DefaultModel)
	}
}
