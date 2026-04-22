package classifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joaoajmatos/oz/internal/openrouter"
)

// newTestLLMClassifier creates an llmClassifier pointed at a test HTTP server.
func newTestLLMClassifier(t *testing.T, srv *httptest.Server) *llmClassifier {
	t.Helper()
	return &llmClassifier{
		client: &openrouter.Client{
			APIKey:  "test-key",
			BaseURL: srv.URL,
			Model:   "test-model",
			HTTP:    &http.Client{},
		},
		model:   "test-model",
		context: minimalContext,
	}
}

func mockSrv(t *testing.T, responseJSON string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": responseJSON}},
			},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLLMClassifier_ParseValidResponse(t *testing.T) {
	payload := `{"type":"adr","confidence":"high","title":"Auth Rewrite","reason":"contains decision language"}`
	srv := mockSrv(t, payload)
	llm := newTestLLMClassifier(t, srv)

	got, err := llm.classify("notes/auth.md", []byte("We decided to rewrite auth."))
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	if got.Type != TypeADR {
		t.Errorf("type = %q, want %q", got.Type, TypeADR)
	}
	if got.Confidence != ConfidenceHigh {
		t.Errorf("confidence = %q, want %q", got.Confidence, ConfidenceHigh)
	}
	if got.Title != "Auth Rewrite" {
		t.Errorf("title = %q, want %q", got.Title, "Auth Rewrite")
	}
	if got.Source != SourceLLM {
		t.Errorf("source = %q, want %q", got.Source, SourceLLM)
	}
}

func TestLLMClassifier_StripMarkdownFences(t *testing.T) {
	// LLM wraps JSON in code fences despite instructions.
	payload := "```json\n{\"type\":\"spec\",\"confidence\":\"medium\",\"title\":\"My Spec\",\"reason\":\"has must language\"}\n```"
	srv := mockSrv(t, payload)
	llm := newTestLLMClassifier(t, srv)

	got, err := llm.classify("notes/spec.md", []byte("All agents MUST..."))
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	if got.Type != TypeSpec {
		t.Errorf("type = %q, want %q", got.Type, TypeSpec)
	}
}

func TestLLMClassifier_RetryOnBadJSON(t *testing.T) {
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		w.Header().Set("Content-Type", "application/json")
		var content string
		if call == 1 {
			content = "not json at all"
		} else {
			content = `{"type":"guide","confidence":"high","title":"Setup","reason":"has steps"}`
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": content}},
			},
		})
	}))
	defer srv.Close()
	llm := newTestLLMClassifier(t, srv)

	got, err := llm.classify("notes/setup.md", []byte("Step 1: install. Step 2: run."))
	if err != nil {
		t.Fatalf("classify after retry: %v", err)
	}
	if got.Type != TypeGuide {
		t.Errorf("type = %q, want %q", got.Type, TypeGuide)
	}
	if call != 2 {
		t.Errorf("expected 2 LLM calls (1 fail + 1 retry), got %d", call)
	}
}

func TestLLMClassifier_UnknownType(t *testing.T) {
	payload := `{"type":"unknown","confidence":"low","title":"Scratch","reason":"too vague"}`
	srv := mockSrv(t, payload)
	llm := newTestLLMClassifier(t, srv)

	got, err := llm.classify("notes/scratch.md", []byte("quick note"))
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	if got.Type != TypeUnknown {
		t.Errorf("type = %q, want %q", got.Type, TypeUnknown)
	}
}

func TestParseClassifyResponse_AllTypes(t *testing.T) {
	cases := []struct {
		raw      string
		wantType ArtifactType
	}{
		{`{"type":"adr","confidence":"high","title":"T","reason":"R"}`, TypeADR},
		{`{"type":"spec","confidence":"high","title":"T","reason":"R"}`, TypeSpec},
		{`{"type":"guide","confidence":"high","title":"T","reason":"R"}`, TypeGuide},
		{`{"type":"arch","confidence":"high","title":"T","reason":"R"}`, TypeArch},
		{`{"type":"open-item","confidence":"high","title":"T","reason":"R"}`, TypeOpenItem},
		{`{"type":"open_item","confidence":"high","title":"T","reason":"R"}`, TypeOpenItem},
		{`{"type":"unknown","confidence":"low","title":"T","reason":"R"}`, TypeUnknown},
		{`{"type":"GARBAGE","confidence":"low","title":"T","reason":"R"}`, TypeUnknown},
	}
	for _, tc := range cases {
		got, err := parseClassifyResponse(tc.raw)
		if err != nil {
			t.Errorf("parseClassifyResponse(%q): %v", tc.raw, err)
			continue
		}
		if got.Type != tc.wantType {
			t.Errorf("parseClassifyResponse(%q) type = %q, want %q", tc.raw, got.Type, tc.wantType)
		}
	}
}

func TestParseClassifyResponse_MissingTitle(t *testing.T) {
	_, err := parseClassifyResponse(`{"type":"adr","confidence":"high","title":"","reason":"R"}`)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestParseClassifyResponse_MalformedJSON(t *testing.T) {
	_, err := parseClassifyResponse(`{bad json`)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
