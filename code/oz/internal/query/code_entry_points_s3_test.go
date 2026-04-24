package query

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/semantic"
)

func TestLoadCodeEntryPoints_EligibleByScopeOrConceptChain(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "context"), 0755); err != nil {
		t.Fatalf("mkdir context: %v", err)
	}
	g := &graph.Graph{
		Nodes: []graph.Node{
			{
				ID:    "agent:oz-coding",
				Type:  graph.NodeTypeAgent,
				Name:  "oz-coding",
				Scope: []string{"code/owned/**"},
			},
			{
				ID:         "code_symbol:code/owned/run.go:drift.Run",
				Type:       graph.NodeTypeCodeSymbol,
				File:       "code/owned/run.go",
				Name:       "drift.Run",
				SymbolKind: "func",
				Line:       10,
				Package:    "audit/drift",
			},
			{
				ID:         "code_symbol:code/other/load.go:drift.LoadSymbols",
				Type:       graph.NodeTypeCodeSymbol,
				File:       "code/other/load.go",
				Name:       "drift.LoadSymbols",
				SymbolKind: "func",
				Line:       20,
				Package:    "audit/drift",
			},
			{
				ID:         "code_symbol:code/other/skip.go:graph.Build",
				Type:       graph.NodeTypeCodeSymbol,
				File:       "code/other/skip.go",
				Name:       "graph.Build",
				SymbolKind: "func",
				Line:       30,
				Package:    "graph",
			},
		},
	}
	overlay := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		Concepts: []semantic.Concept{
			{
				ID:          "concept:drift",
				Name:        "drift detection",
				Description: "detect drift in code and spec",
			},
		},
		Edges: []semantic.ConceptEdge{
			{
				From:     "concept:drift",
				To:       "code_package:audit/drift",
				Type:     semantic.EdgeTypeImplements,
				Reviewed: true,
			},
		},
	}
	if err := semantic.Write(workspace, overlay); err != nil {
		t.Fatalf("write semantic overlay: %v", err)
	}

	cfg := DefaultScoringConfig()
	cfg.RetrievalMinRelevance = 0.05

	got := loadCodeEntryPoints(workspace, g, "oz-coding", []string{"drift", "implement"}, cfg)
	if len(got) != 2 {
		t.Fatalf("expected 2 eligible entry points, got %d: %+v", len(got), got)
	}
	if !hasSymbol(got, "drift.Run") {
		t.Fatalf("expected scope-based symbol drift.Run in entry points: %+v", got)
	}
	if !hasSymbol(got, "drift.LoadSymbols") {
		t.Fatalf("expected concept-chain symbol drift.LoadSymbols in entry points: %+v", got)
	}
}

func TestLoadCodeEntryPoints_ReviewedImplementsRequired(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "context"), 0755); err != nil {
		t.Fatalf("mkdir context: %v", err)
	}
	g := &graph.Graph{
		Nodes: []graph.Node{
			{
				ID:    "agent:oz-coding",
				Type:  graph.NodeTypeAgent,
				Name:  "oz-coding",
				Scope: []string{"code/owned/**"},
			},
			{
				ID:         "code_symbol:code/other/load.go:drift.LoadSymbols",
				Type:       graph.NodeTypeCodeSymbol,
				File:       "code/other/load.go",
				Name:       "drift.LoadSymbols",
				SymbolKind: "func",
				Line:       20,
				Package:    "audit/drift",
			},
		},
	}
	overlay := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		Concepts: []semantic.Concept{
			{
				ID:          "concept:drift",
				Name:        "drift detection",
				Description: "detect drift in code and spec",
			},
		},
		Edges: []semantic.ConceptEdge{
			{
				From:     "concept:drift",
				To:       "code_package:audit/drift",
				Type:     semantic.EdgeTypeImplements,
				Reviewed: false,
			},
		},
	}
	if err := semantic.Write(workspace, overlay); err != nil {
		t.Fatalf("write semantic overlay: %v", err)
	}

	cfg := DefaultScoringConfig()
	cfg.RetrievalMinRelevance = 0.05

	got := loadCodeEntryPoints(workspace, g, "oz-coding", []string{"drift", "implement"}, cfg)
	if len(got) != 0 {
		t.Fatalf("expected no entry points when implements edge is unreviewed, got %+v", got)
	}
}

func hasSymbol(entries []CodeEntryPoint, symbol string) bool {
	for _, e := range entries {
		if e.Symbol == symbol {
			return true
		}
	}
	return false
}
