package classifier

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

// newTestLLMClassifier creates an llmClassifier with a stubbed HTTP transport.
func newTestLLMClassifier(t *testing.T, responses ...string) (*llmClassifier, *int) {
	t.Helper()
	call := 0
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if call >= len(responses) {
			t.Fatalf("unexpected extra request (call %d)", call+1)
		}
		payload := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": responses[call]}},
			},
		}
		call++
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal stub payload: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(b))),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})
	return &llmClassifier{
		client: &openrouter.Client{
			APIKey:  "test-key",
			BaseURL: "http://example.test",
			Model:   "test-model",
			HTTP:    &http.Client{Transport: rt},
		},
		model:   "test-model",
		context: minimalContext,
	}, &call
}

func TestLLMClassifier_ParseValidResponse(t *testing.T) {
	payload := `{"type":"adr","confidence":"high","title":"Auth Rewrite","reason":"contains decision language"}`
	llm, _ := newTestLLMClassifier(t, payload)

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
	llm, _ := newTestLLMClassifier(t, payload)

	got, err := llm.classify("notes/spec.md", []byte("All agents MUST..."))
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	if got.Type != TypeSpec {
		t.Errorf("type = %q, want %q", got.Type, TypeSpec)
	}
}

func TestLLMClassifier_RetryOnBadJSON(t *testing.T) {
	llm, calls := newTestLLMClassifier(t,
		"not json at all",
		`{"type":"guide","confidence":"high","title":"Setup","reason":"has steps"}`,
	)

	got, err := llm.classify("notes/setup.md", []byte("Step 1: install. Step 2: run."))
	if err != nil {
		t.Fatalf("classify after retry: %v", err)
	}
	if got.Type != TypeGuide {
		t.Errorf("type = %q, want %q", got.Type, TypeGuide)
	}
	if *calls != 2 {
		t.Errorf("expected 2 LLM calls (1 fail + 1 retry), got %d", *calls)
	}
}

func TestLLMClassifier_UnknownType(t *testing.T) {
	payload := `{"type":"unknown","confidence":"low","title":"Scratch","reason":"too vague"}`
	llm, _ := newTestLLMClassifier(t, payload)

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
