package classifier_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/crystallize/classifier"
)

func writeNote(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}
	return path
}

// TestClassify_FrontmatterOverride checks that a crystallize: tag skips both
// LLM and heuristic classifiers.
func TestClassify_FrontmatterOverride(t *testing.T) {
	dir := t.TempDir()
	path := writeNote(t, dir, "decision.md", "---\ncrystallize: adr\ncrystallize-title: Auth Rewrite\n---\n\nsome content")

	c := classifier.New(classifier.Options{
		WorkspaceRoot: dir,
		NoEnrich:      true, // LLM unavailable
	})
	got, err := c.Classify(path)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if got.Type != classifier.TypeADR {
		t.Errorf("type = %q, want %q", got.Type, classifier.TypeADR)
	}
	if got.Source != classifier.SourceFrontmatter {
		t.Errorf("source = %q, want %q", got.Source, classifier.SourceFrontmatter)
	}
	if got.Title != "Auth Rewrite" {
		t.Errorf("title = %q, want %q", got.Title, "Auth Rewrite")
	}
}

// TestClassify_LLMResult checks that the LLM classifier path is used when a
// valid OpenRouter response is returned, and the result is cached.
func TestClassify_LLMResult(t *testing.T) {
	t.Skip("LLM path tested in llm_test.go with a stubbed RoundTripper (no sockets)")
}

// TestClassify_HeuristicFallback checks that when no API key is present the
// heuristic fallback produces a reasonable result.
func TestClassify_HeuristicFallback(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")

	dir := t.TempDir()
	path := writeNote(t, dir, "decision.md",
		"We decided to use BM25F. We rejected TF-IDF. ## Alternatives considered\nTF-IDF was rejected.")

	c := classifier.New(classifier.Options{WorkspaceRoot: dir})
	got, err := c.Classify(path)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if got.Type != classifier.TypeADR {
		t.Errorf("type = %q, want %q", got.Type, classifier.TypeADR)
	}
	if got.Source != classifier.SourceHeuristic {
		t.Errorf("source = %q, want %q", got.Source, classifier.SourceHeuristic)
	}
}

// TestClassify_IsAutoAcceptable checks the confidence gating logic for
// --accept-all.
func TestClassify_IsAutoAcceptable(t *testing.T) {
	cases := []struct {
		cl   classifier.Classification
		want bool
	}{
		{classifier.Classification{Type: classifier.TypeADR, Confidence: classifier.ConfidenceHigh}, true},
		{classifier.Classification{Type: classifier.TypeSpec, Confidence: classifier.ConfidenceMedium}, false},
		{classifier.Classification{Type: classifier.TypeUnknown, Confidence: classifier.ConfidenceHigh}, false},
		{classifier.Classification{Type: classifier.TypeGuide, Confidence: classifier.ConfidenceLow}, false},
	}
	for _, tc := range cases {
		got := tc.cl.IsAutoAcceptable()
		if got != tc.want {
			t.Errorf("IsAutoAcceptable(%+v) = %v, want %v", tc.cl, got, tc.want)
		}
	}
}

// TestClassify_FrontmatterUnknownType checks that an invalid crystallize: tag
// falls through to the heuristic rather than returning an error.
func TestClassify_FrontmatterUnknownType(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")

	dir := t.TempDir()
	path := writeNote(t, dir, "note.md",
		"---\ncrystallize: bogus\n---\n\nWe decided to use X. We rejected Y. Alternatives considered.")

	c := classifier.New(classifier.Options{WorkspaceRoot: dir})
	got, err := c.Classify(path)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	// Should fall through to heuristic and classify as adr.
	if got.Source != classifier.SourceHeuristic {
		t.Errorf("source = %q, want %q (bogus tag should fall through)", got.Source, classifier.SourceHeuristic)
	}
}
