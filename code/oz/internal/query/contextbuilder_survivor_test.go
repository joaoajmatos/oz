package query

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/convention"
	"github.com/joaoajmatos/oz/internal/graph"
)

func TestBuildContextBlocks_EnsuresScopeSurvivorAfterTruncation(t *testing.T) {
	ws := t.TempDir()
	mustWrite(t, ws, "specs/top.md", "## top\napi api api api")
	mustWrite(t, ws, "code/win/impl.md", "## impl\napi")

	g := &graph.Graph{
		ContentHash: "survivor-hash",
		Nodes: []graph.Node{
			{
				ID:   "agent:winner",
				Type: graph.NodeTypeAgent,
				Name: "winner",
				Scope: []string{
					"code/win/**",
				},
			},
			{
				ID:      "spec_section:specs/top.md:top",
				Type:    graph.NodeTypeSpecSection,
				File:    "specs/top.md",
				Section: "top",
				Name:    "top",
				Tier:    convention.TierSpecs,
			},
			{
				ID:      "doc:code/win/impl.md:impl",
				Type:    graph.NodeTypeDoc,
				File:    "code/win/impl.md",
				Section: "impl",
				Name:    "impl",
				Tier:    convention.TierDocs,
			},
		},
	}

	cfg := DefaultScoringConfig()
	cfg.RetrievalMinRelevance = 0.0
	cfg.RetrievalMaxBlocks = 1

	blocks, _, _ := BuildContextBlocks(ws, g, "winner", []string{"api"}, cfg)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].File != "code/win/impl.md" {
		t.Fatalf("expected scope survivor block, got %s", blocks[0].File)
	}
}

func mustWrite(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

