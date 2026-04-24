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
	cfg.RetrievalMinRelevance = 0.08
	cfg.RetrievalWeightBody = 1.4
	cfg.RetrievalMaxCodeEntryPoints = 7
	cfg.RetrievalWeightKind = 0.9
	cfg.RetrievalMaxImplementingPackages = 6
	cfg.RetrievalConceptMinRelevance = 0.07
	cfg.RetrievalConceptWeightName = 2.2
	cfg.RetrievalConceptWeightDescription = 1.3
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
	if got.RetrievalMinRelevance != 0.08 {
		t.Fatalf("RetrievalMinRelevance = %v, want 0.08", got.RetrievalMinRelevance)
	}
	if got.RetrievalWeightBody != 1.4 {
		t.Fatalf("RetrievalWeightBody = %v, want 1.4", got.RetrievalWeightBody)
	}
	if got.RetrievalMaxCodeEntryPoints != 7 {
		t.Fatalf("RetrievalMaxCodeEntryPoints = %v, want 7", got.RetrievalMaxCodeEntryPoints)
	}
	if got.RetrievalWeightKind != 0.9 {
		t.Fatalf("RetrievalWeightKind = %v, want 0.9", got.RetrievalWeightKind)
	}
	if got.RetrievalMaxImplementingPackages != 6 {
		t.Fatalf("RetrievalMaxImplementingPackages = %v, want 6", got.RetrievalMaxImplementingPackages)
	}
	if got.RetrievalConceptMinRelevance != 0.07 {
		t.Fatalf("RetrievalConceptMinRelevance = %v, want 0.07", got.RetrievalConceptMinRelevance)
	}
	if got.RetrievalConceptWeightName != 2.2 {
		t.Fatalf("RetrievalConceptWeightName = %v, want 2.2", got.RetrievalConceptWeightName)
	}
	if got.RetrievalConceptWeightDescription != 1.3 {
		t.Fatalf("RetrievalConceptWeightDescription = %v, want 1.3", got.RetrievalConceptWeightDescription)
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
