package query

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/graph"
)

func TestLoadRetrievalBodyTokens_UsesSectionBodyAndCache(t *testing.T) {
	ws := t.TempDir()
	file := "specs/sample.md"
	abs := filepath.Join(ws, file)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "# Sample\n\n## Intro\nalpha beta\n\n## Other\ngamma\n"
	if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	node := graph.Node{
		ID:      "spec_section:specs/sample.md:Intro",
		Type:    graph.NodeTypeSpecSection,
		File:    file,
		Section: "Intro",
	}
	tokensFirst := loadRetrievalBodyTokens(ws, "hash-a", node, false)
	if len(tokensFirst) == 0 {
		t.Fatal("expected tokens from section body")
	}
	if containsToken(tokensFirst, "gamma") {
		t.Fatalf("expected section-only tokens, got %v", tokensFirst)
	}

	// Change file content; same hash should still return cached tokens.
	updated := "# Sample\n\n## Intro\nchanged\n\n## Other\ngamma\n"
	if err := os.WriteFile(abs, []byte(updated), 0644); err != nil {
		t.Fatalf("rewrite file: %v", err)
	}
	tokensCached := loadRetrievalBodyTokens(ws, "hash-a", node, false)
	if !equalTokens(tokensFirst, tokensCached) {
		t.Fatalf("expected cache hit for same hash, first=%v cached=%v", tokensFirst, tokensCached)
	}
}

func TestLoadRetrievalBodyTokens_InvalidatesOnHashChange(t *testing.T) {
	ws := t.TempDir()
	file := "notes/plan.md"
	abs := filepath.Join(ws, file)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte("first body"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	node := graph.Node{
		ID:   "note:notes/plan.md",
		Type: graph.NodeTypeNote,
		File: file,
	}
	first := loadRetrievalBodyTokens(ws, "hash-old", node, false)
	if len(first) == 0 {
		t.Fatal("expected initial tokens")
	}

	if err := os.WriteFile(abs, []byte("second body"), 0644); err != nil {
		t.Fatalf("rewrite file: %v", err)
	}
	second := loadRetrievalBodyTokens(ws, "hash-new", node, false)
	if equalTokens(first, second) {
		t.Fatalf("expected cache invalidation on hash change, first=%v second=%v", first, second)
	}
}

func equalTokens(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsToken(tokens []string, want string) bool {
	for _, tok := range tokens {
		if tok == want {
			return true
		}
	}
	return false
}

