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

func TestLoadCodeEntryPoints_CappedByMaxCodeEntryPoints(t *testing.T) {
	workspace := t.TempDir()
	_ = os.MkdirAll(filepath.Join(workspace, "context"), 0755)

	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:w", Type: graph.NodeTypeAgent, Name: "w", Scope: []string{"code/**"}},
			{ID: "s1", Type: graph.NodeTypeCodeSymbol, File: "code/a.go", Name: "p.A", SymbolKind: "func", Line: 1, Package: "p"},
			{ID: "s2", Type: graph.NodeTypeCodeSymbol, File: "code/b.go", Name: "p.B", SymbolKind: "func", Line: 1, Package: "p"},
			{ID: "s3", Type: graph.NodeTypeCodeSymbol, File: "code/c.go", Name: "p.C", SymbolKind: "func", Line: 1, Package: "p"},
			{ID: "s4", Type: graph.NodeTypeCodeSymbol, File: "code/d.go", Name: "p.D", SymbolKind: "func", Line: 1, Package: "p"},
		},
	}
	cfg := DefaultScoringConfig()
	cfg.RetrievalMinRelevance = 0
	cfg.RetrievalMaxCodeEntryPoints = 2
	got := loadCodeEntryPoints(workspace, g, "w", []string{"p"}, cfg)
	if len(got) != 2 {
		t.Fatalf("expected 2 after cap, got %d: %+v", len(got), got)
	}
}

func TestLoadCodeEntryPoints_RankedByRelevanceDesc(t *testing.T) {
	workspace := t.TempDir()
	_ = os.MkdirAll(filepath.Join(workspace, "context"), 0755)
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:w", Type: graph.NodeTypeAgent, Name: "w", Scope: []string{"code/**"}},
			{ID: "s1", Type: graph.NodeTypeCodeSymbol, File: "code/z.go", Name: "other.Thing", SymbolKind: "func", Line: 1, Package: "other"},
			{ID: "s2", Type: graph.NodeTypeCodeSymbol, File: "code/y.go", Name: "drift.Run", SymbolKind: "func", Line: 1, Package: "drift"},
		},
	}
	cfg := DefaultScoringConfig()
	cfg.RetrievalMinRelevance = 0
	cfg.RetrievalMaxCodeEntryPoints = 5
	got := loadCodeEntryPoints(workspace, g, "w", []string{"drift"}, cfg)
	if len(got) < 2 {
		t.Fatalf("expected 2 entry points, got %d: %+v", len(got), got)
	}
	if got[0].Symbol != "drift.Run" {
		t.Fatalf("expected best match drift.Run first, got order: %#v", got)
	}
	if got[0].Relevance < got[1].Relevance-1e-9 {
		t.Fatalf("expected descending relevance, got %v then %v", got[0].Relevance, got[1].Relevance)
	}
}

// When BM25 relevance ties (e.g. path weight 0, same symbol name), a more
// specific import path (subpackage) may rank before the parent so scan.go is
// not always cut by lexicographic file order under a low max_code_entry_points cap.
func TestLoadCodeEntryPoints_TieBreakDeeperPackageFirst(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "context"), 0755); err != nil {
		t.Fatalf("mkdir context: %v", err)
	}
	overlay := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		Concepts: []semantic.Concept{
			{
				ID: "concept:drift", Name: "drift", Description: "drift in audit",
			},
		},
		Edges: []semantic.ConceptEdge{
			{From: "concept:drift", To: "code_package:example.com/parent/drift", Type: semantic.EdgeTypeImplements, Reviewed: true},
			{From: "concept:drift", To: "code_package:example.com/parent/drift/specscan", Type: semantic.EdgeTypeImplements, Reviewed: true},
		},
	}
	if err := semantic.Write(workspace, overlay); err != nil {
		t.Fatalf("write semantic: %v", err)
	}

	// All symbols share the same name/kind; path field disabled so BM25 matches tie.
	// parent has three files (sorted before specscan/ only by file path in old order).
	// one child — must land first with deeper-package tie-break.
	pDrift := "example.com/parent/drift"
	pSpec := "example.com/parent/drift/specscan"
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "a", Type: graph.NodeTypeAgent, Name: "a", Scope: []string{"code/**"}},
			{ID: "1", Type: graph.NodeTypeCodeSymbol, File: "code/aa.go", Name: "F", SymbolKind: "func", Line: 1, Package: pDrift},
			{ID: "2", Type: graph.NodeTypeCodeSymbol, File: "code/bb.go", Name: "F", SymbolKind: "func", Line: 1, Package: pDrift},
			{ID: "3", Type: graph.NodeTypeCodeSymbol, File: "code/cc.go", Name: "F", SymbolKind: "func", Line: 1, Package: pDrift},
			{ID: "4", Type: graph.NodeTypeCodeSymbol, File: "code/dd/scan.go", Name: "F", SymbolKind: "func", Line: 1, Package: pSpec},
		},
	}

	cfg := DefaultScoringConfig()
	cfg.UseBigrams = false
	cfg.RetrievalMinRelevance = 0
	cfg.RetrievalMaxCodeEntryPoints = 1
	cfg.RetrievalWeightPath = 0
	// Drift in query so the overlay concept is relevant; path weight 0 so symbol
	// rows tie (title "F" only).
	got := loadCodeEntryPoints(workspace, g, "a", []string{"drift"}, cfg)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %+v", len(got), got)
	}
	if got[0].Package != pSpec {
		t.Fatalf("expected subpackage first on tie, got %q (want %q): %#v", got[0].Package, pSpec, got[0])
	}
}
