package openrouter_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oz-tools/oz/internal/openrouter"
)

func TestClient_Complete(t *testing.T) {
	wantContent := "extracted concepts JSON here"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": wantContent}},
			},
			"usage": map[string]any{"cost": 0.0012},
		})
	}))
	defer srv.Close()

	c := &openrouter.Client{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Model:   "test-model",
		HTTP:    &http.Client{},
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer srv.Close()

	c := &openrouter.Client{
		APIKey:  "bad-key",
		BaseURL: srv.URL,
		Model:   "test-model",
		HTTP:    &http.Client{},
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
