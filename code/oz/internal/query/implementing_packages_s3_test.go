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

// TestLoadImplementingPackages_SuppressesWeakImplementsEdgeMatch guards the live
// workspace failure mode: query stems like "implement" match
// concept:implements-edge in addition to the drift concept. At the default
// concept_min_relevance, the weak spurious match must fall out so
// internal/graph is not included via the implements-edge chain.
func TestLoadImplementingPackages_SuppressesWeakImplementsEdgeMatch(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "context"), 0755); err != nil {
		t.Fatalf("mkdir context: %v", err)
	}
	overlay := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		Concepts: []semantic.Concept{
			{
				ID:          "concept:spec-code-drift-detection",
				Name:        "Spec-Code Drift Detection",
				Description: "The audit mechanism that detects when code symbols or packages implement concepts not referenced in specs or when specs reference non-existent code.",
			},
			{
				ID:          "concept:implements-edge",
				Name:        "Implements Edge",
				Description: "The semantic relationship linking a concept to the code package that realizes it, or a spec section to the code that implements its rules.",
			},
		},
		Edges: []semantic.ConceptEdge{
			{From: "concept:spec-code-drift-detection", To: "code_package:github.com/joaoajmatos/oz/internal/audit/drift", Type: semantic.EdgeTypeImplements, Reviewed: true},
			{From: "concept:implements-edge", To: "code_package:github.com/joaoajmatos/oz/internal/graph", Type: semantic.EdgeTypeImplements, Reviewed: true},
		},
	}
	if err := semantic.Write(workspace, overlay); err != nil {
		t.Fatalf("write semantic overlay: %v", err)
	}
	cfg := DefaultScoringConfig()
	if err := WriteScoringTOML(workspace, cfg); err != nil {
		t.Fatalf("write scoring.toml: %v", err)
	}
	terms := TokenizeQuery("how is drift detection implemented", false)
	got := loadImplementingPackages(workspace, terms)
	for _, p := range got {
		if p == "github.com/joaoajmatos/oz/internal/graph" {
			t.Fatalf("expected internal/graph not from implements-edge at default threshold, got %+v", got)
		}
	}
	var hasDrift bool
	for _, p := range got {
		if p == "github.com/joaoajmatos/oz/internal/audit/drift" {
			hasDrift = true
			break
		}
	}
	if !hasDrift {
		t.Fatalf("expected audit/drift, got %+v", got)
	}
}
