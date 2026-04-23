package query

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_RoundTripWrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "context"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultScoringConfig()
	cfg.ConfidenceThreshold = 0.65
	if err := WriteScoringTOML(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got := LoadConfig(dir)
	if got.ConfidenceThreshold != 0.65 {
		t.Fatalf("ConfidenceThreshold = %v, want 0.65", got.ConfidenceThreshold)
	}
	if got.K1 != cfg.K1 {
		t.Fatalf("K1 drift after round-trip")
	}
}

func TestValidateScoringFile_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "context"), 0755)
	bad := `[bm25]
k1 = 1.2
[unknown]
x = 1
`
	if err := os.WriteFile(filepath.Join(dir, "context", "scoring.toml"), []byte(bad), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateScoringFile(dir); err == nil {
		t.Fatal("expected error for unknown top-level table")
	}
}

func TestUnknownKeysError_Nested(t *testing.T) {
	bad := `[bm25]
k1 = 1.2
extra = 3
`
	if err := unknownKeysError([]byte(bad)); err == nil {
		t.Fatal("expected error for unknown key in [bm25]")
	}
}
