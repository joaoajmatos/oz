package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetScoringKey_rejectInvalidMaxBlocks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "context"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "context", "scoring.toml"), []byte("[bm25]\nk1 = 1.2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SetScoringKey(dir, "retrieval.max_blocks", "-1"); err == nil {
		t.Fatal("SetScoringKey: expected error for negative max_blocks")
	}
}

func TestSetScoringKey_rejectInvalidBool(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "context"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "context", "scoring.toml"), []byte("[bm25]\nk1 = 1.2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SetScoringKey(dir, "retrieval.include_notes", "maybe"); err == nil {
		t.Fatal("SetScoringKey: expected error for invalid bool")
	}
}

func TestValidateScoringFile_rejectInvalidValueInFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "context"), 0755); err != nil {
		t.Fatal(err)
	}
	bad := `
[bm25]
k1 = 1.2
[retrieval]
max_blocks = -1
`
	if err := os.WriteFile(filepath.Join(dir, "context", "scoring.toml"), []byte(bad), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateScoringFile(dir); err == nil {
		t.Fatal("ValidateScoringFile: expected error for max_blocks = -1")
	} else if !strings.Contains(err.Error(), "retrieval.max_blocks") {
		t.Fatalf("error should mention key: %v", err)
	}
}
