package contextretrieval

import (
	"testing"

	"github.com/joaoajmatos/oz/internal/query/bm25"
)

func TestScore_DeterministicTieBreak(t *testing.T) {
	cfg := RetrievalConfig{
		K1: 1.2,
		Fields: []bm25.BM25Field{
			{Name: "title", Weight: 1.0, B: 0.75},
		},
		TrustBoost: map[string]float64{
			"high":   1.3,
			"medium": 1.0,
			"low":    0.6,
		},
		AgentAffinityBoost: 1.2,
	}

	blocks := []Block{
		{
			File:    "b.md",
			Section: "B",
			Trust:   "high",
			TokenFields: map[string][]string{
				"title": {"drift"},
			},
			ConnectedAgents: map[string]bool{"oz-coding": false},
		},
		{
			File:    "a.md",
			Section: "A",
			Trust:   "high",
			TokenFields: map[string][]string{
				"title": {"drift"},
			},
			ConnectedAgents: map[string]bool{"oz-coding": false},
		},
		{
			File:    "c.md",
			Section: "C",
			Trust:   "medium",
			TokenFields: map[string][]string{
				"title": {"drift"},
			},
			ConnectedAgents: map[string]bool{"oz-coding": false},
		},
	}

	got := Score([]string{"drift"}, blocks, cfg, "oz-coding")
	if len(got) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(got))
	}

	if got[0].Block.File != "a.md" || got[1].Block.File != "b.md" {
		t.Fatalf("expected high-trust ties sorted by file asc, got [%s, %s]",
			got[0].Block.File, got[1].Block.File)
	}
	if got[2].Block.File != "c.md" {
		t.Fatalf("expected medium-trust block last, got %s", got[2].Block.File)
	}
}

func TestScore_StableAcrossRepeatedRuns(t *testing.T) {
	cfg := RetrievalConfig{
		K1: 1.2,
		Fields: []bm25.BM25Field{
			{Name: "title", Weight: 2.0, B: 0.75},
			{Name: "body", Weight: 1.0, B: 0.75},
		},
		TrustBoost: map[string]float64{
			"high": 1.3,
		},
		AgentAffinityBoost: 1.2,
	}

	blocks := []Block{
		{
			File:    "specs/a.md",
			Section: "A",
			Trust:   "high",
			TokenFields: map[string][]string{
				"title": {"drift", "detect"},
				"body":  {"implementation", "details"},
			},
			ConnectedAgents: map[string]bool{"oz-coding": true},
		},
		{
			File:    "specs/b.md",
			Section: "B",
			Trust:   "high",
			TokenFields: map[string][]string{
				"title": {"drift"},
				"body":  {"overview"},
			},
			ConnectedAgents: map[string]bool{"oz-coding": false},
		},
	}

	first := Score([]string{"drift", "implementation"}, blocks, cfg, "oz-coding")
	second := Score([]string{"drift", "implementation"}, blocks, cfg, "oz-coding")
	if len(first) != len(second) {
		t.Fatalf("inconsistent result lengths: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Block.File != second[i].Block.File ||
			first[i].Block.Section != second[i].Block.Section ||
			first[i].Relevance != second[i].Relevance {
			t.Fatalf("unstable ordering/scoring at idx %d: first=%+v second=%+v", i, first[i], second[i])
		}
	}
}

