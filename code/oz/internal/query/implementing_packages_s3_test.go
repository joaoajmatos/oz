package query

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/semantic"
)

func TestLoadImplementingPackages_RankedAndCapped(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "context"), 0755); err != nil {
		t.Fatalf("mkdir context: %v", err)
	}
	overlay := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		Concepts: []semantic.Concept{
			{ID: "concept:drift", Name: "drift detection", Description: "find drift in code and specs"},
			{ID: "concept:index", Name: "graph index", Description: "index graph structures"},
		},
		Edges: []semantic.ConceptEdge{
			{From: "concept:drift", To: "code_package:audit/drift", Type: semantic.EdgeTypeImplements, Reviewed: true},
			{From: "concept:index", To: "code_package:graph/index", Type: semantic.EdgeTypeImplements, Reviewed: true},
		},
	}
	if err := semantic.Write(workspace, overlay); err != nil {
		t.Fatalf("write semantic overlay: %v", err)
	}
	cfg := DefaultScoringConfig()
	cfg.RetrievalConceptMinRelevance = 0.0
	cfg.RetrievalMaxImplementingPackages = 1
	if err := WriteScoringTOML(workspace, cfg); err != nil {
		t.Fatalf("write scoring.toml: %v", err)
	}

	got := loadImplementingPackages(workspace, []string{"drift"})
	if len(got) != 1 {
		t.Fatalf("expected 1 package after cap, got %d: %+v", len(got), got)
	}
	if got[0] != "audit/drift" {
		t.Fatalf("expected drift package first for drift query, got %+v", got)
	}
}

func TestLoadImplementingPackages_ConceptThresholdAndReviewedGate(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "context"), 0755); err != nil {
		t.Fatalf("mkdir context: %v", err)
	}
	overlay := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		Concepts: []semantic.Concept{
			{ID: "concept:drift", Name: "drift detection", Description: "find drift in code and specs"},
			{ID: "concept:graph", Name: "graph model", Description: "graph serialization details"},
		},
		Edges: []semantic.ConceptEdge{
			{From: "concept:drift", To: "code_package:audit/drift", Type: semantic.EdgeTypeImplements, Reviewed: false},
			{From: "concept:graph", To: "code_package:graph/core", Type: semantic.EdgeTypeImplements, Reviewed: true},
		},
	}
	if err := semantic.Write(workspace, overlay); err != nil {
		t.Fatalf("write semantic overlay: %v", err)
	}
	cfg := DefaultScoringConfig()
	cfg.RetrievalConceptMinRelevance = 1.0 // strict: only strong match survives
	cfg.RetrievalMaxImplementingPackages = 5
	if err := WriteScoringTOML(workspace, cfg); err != nil {
		t.Fatalf("write scoring.toml: %v", err)
	}
	got := loadImplementingPackages(workspace, []string{"drift"})
	if len(got) != 0 {
		t.Fatalf("expected no packages (drift edge unreviewed, graph concept below threshold), got %+v", got)
	}
}
