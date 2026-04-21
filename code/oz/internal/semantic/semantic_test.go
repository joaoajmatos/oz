package semantic_test

import (
	"testing"

	"github.com/joaoajmatos/oz/internal/semantic"
)

// --- IsStale -----------------------------------------------------------------

func TestIsStale_MatchingHash(t *testing.T) {
	o := &semantic.Overlay{GraphHash: "abc123"}
	if semantic.IsStale(o, "abc123") {
		t.Error("IsStale should return false when hashes match")
	}
}

func TestIsStale_DifferentHash(t *testing.T) {
	o := &semantic.Overlay{GraphHash: "abc123"}
	if !semantic.IsStale(o, "def456") {
		t.Error("IsStale should return true when hashes differ")
	}
}

func TestIsStale_NilOverlay(t *testing.T) {
	if semantic.IsStale(nil, "abc123") {
		t.Error("IsStale should return false for nil overlay")
	}
}

func TestIsStale_EmptyOverlayHash(t *testing.T) {
	// Overlay present but generated before graph_hash was introduced.
	o := &semantic.Overlay{GraphHash: ""}
	if semantic.IsStale(o, "abc123") {
		t.Error("IsStale should return false when overlay has no recorded hash")
	}
}

// --- Load (no-file case) -----------------------------------------------------

func TestLoad_MissingFile_ReturnsNilNil(t *testing.T) {
	dir := t.TempDir() // no semantic.json inside
	o, err := semantic.Load(dir)
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if o != nil {
		t.Error("Load should return nil overlay when file is absent")
	}
}
