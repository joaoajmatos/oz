package enrich_test

import (
	"strings"
	"testing"
	"time"

	"github.com/joaoajmatos/oz/internal/enrich"
	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/review"
	"github.com/joaoajmatos/oz/internal/semantic"
)

// minimalGraph builds a small graph sufficient for proposal testing.
func minimalGraph() *graph.Graph {
	return &graph.Graph{
		ContentHash: "e2e-test-hash",
		Nodes: []graph.Node{
			{ID: "agent:oz-coding", Type: graph.NodeTypeAgent, Name: "oz-coding"},
			{ID: "spec_section:specs/semantic-overlay.md:Merge contract", Type: graph.NodeTypeSpecSection},
			{ID: "code_package:github.com/joaoajmatos/oz/internal/enrich", Type: graph.NodeTypeCodePackage},
		},
	}
}

// prebaked is a valid single-concept LLM response referencing nodes in minimalGraph.
const prebaked = `{
  "concepts": [{
    "id": "concept:oz-throwaway-concept",
    "name": "oz-throwaway-concept",
    "description": "A throwaway concept for dogfood testing.",
    "source_files": ["code/oz/internal/enrich/enrich.go"],
    "tag": "EXTRACTED",
    "confidence": 0.95
  }],
  "edges": [{
    "from": "concept:oz-throwaway-concept",
    "to": "agent:oz-coding",
    "type": "agent_owns_concept",
    "tag": "EXTRACTED",
    "confidence": 0.95
  }]
}`

// TestCCA1_05_E2E_ProposeAcceptReject exercises the full propose → review cycle
// without calling OpenRouter. It simulates what a live dogfood run does:
//  1. Parse a pre-baked single-concept response.
//  2. Merge into semantic.json with reviewed: false.
//  3. Accept-all via oz context review → concept becomes reviewed: true.
//  4. Write a second unreviewed concept, then reject it → concept removed.
func TestCCA1_05_E2E_ProposeAcceptReject(t *testing.T) {
	ws := t.TempDir()
	g := minimalGraph()

	nodeIDs := map[string]struct{}{}
	for _, n := range g.Nodes {
		nodeIDs[n.ID] = struct{}{}
	}

	// --- Phase 1: propose (simulate ProposeConcept without API call) ---
	concept, edges, skipped, err := enrich.ParseSingleConcept(prebaked, nodeIDs)
	if err != nil {
		t.Fatalf("ParseSingleConcept: %v", err)
	}
	if len(skipped) > 0 {
		t.Logf("skipped: %v", skipped)
	}
	if concept.Reviewed {
		t.Fatal("new concept must have reviewed=false after parse")
	}

	incoming := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		GraphHash:     g.ContentHash,
		Model:         "test",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Concepts:      []semantic.Concept{concept},
		Edges:         edges,
	}
	merged := semantic.Merge(nil, incoming)
	if err := semantic.Write(ws, merged); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Verify unreviewed concept is present.
	loaded, err := semantic.Load(ws)
	if err != nil || loaded == nil {
		t.Fatalf("Load after propose: %v", err)
	}
	if len(loaded.Concepts) != 1 {
		t.Fatalf("expected 1 concept after propose, got %d", len(loaded.Concepts))
	}
	if loaded.Concepts[0].Reviewed {
		t.Fatal("concept should be unreviewed after propose")
	}
	if loaded.Concepts[0].ID != "concept:oz-throwaway-concept" {
		t.Errorf("unexpected concept ID: %s", loaded.Concepts[0].ID)
	}

	// --- Phase 2: accept-all via review ---
	var out strings.Builder
	sum, err := review.Run(ws, review.Options{
		AcceptAll: true,
		NoColor:   true,
		Out:       &out,
	})
	if err != nil {
		t.Fatalf("review.Run (accept-all): %v", err)
	}
	if sum.Accepted == 0 {
		t.Errorf("expected accepted > 0, got summary=%+v; output=%s", sum, out.String())
	}

	afterAccept, err := semantic.Load(ws)
	if err != nil {
		t.Fatalf("Load after accept: %v", err)
	}
	for _, c := range afterAccept.Concepts {
		if !c.Reviewed {
			t.Errorf("concept %s should be reviewed=true after accept-all", c.ID)
		}
	}

	// --- Phase 3: propose a second throwaway, then reject it ---
	const prebaked2 = `{
  "concepts": [{
    "id": "concept:oz-throwaway-reject",
    "name": "oz-throwaway-reject",
    "description": "Second throwaway to test rejection.",
    "tag": "EXTRACTED",
    "confidence": 0.8
  }],
  "edges": []
}`
	concept2, edges2, _, err := enrich.ParseSingleConcept(prebaked2, nodeIDs)
	if err != nil {
		t.Fatalf("ParseSingleConcept (second): %v", err)
	}
	incoming2 := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		GraphHash:     g.ContentHash,
		Model:         "test",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Concepts:      []semantic.Concept{concept2},
		Edges:         edges2,
	}
	// Merge with the existing (which has the accepted concept from phase 2).
	existing2, _ := semantic.Load(ws)
	merged2 := semantic.Merge(existing2, incoming2)
	if err := semantic.Write(ws, merged2); err != nil {
		t.Fatalf("Write (second propose): %v", err)
	}

	// Reject via review: send "n" for the unreviewed concept.
	var out2 strings.Builder
	sum2, err := review.Run(ws, review.Options{
		NoColor: true,
		Out:     &out2,
		In:      strings.NewReader("n\n"),
	})
	if err != nil {
		t.Fatalf("review.Run (reject): %v", err)
	}
	if sum2.Rejected == 0 {
		t.Errorf("expected at least one rejection; summary=%+v output=%s", sum2, out2.String())
	}

	afterReject, err := semantic.Load(ws)
	if err != nil {
		t.Fatalf("Load after reject: %v", err)
	}
	for _, c := range afterReject.Concepts {
		if c.ID == "concept:oz-throwaway-reject" {
			t.Errorf("rejected concept should be removed from semantic.json")
		}
	}
	// The originally accepted concept must survive.
	found := false
	for _, c := range afterReject.Concepts {
		if c.ID == "concept:oz-throwaway-concept" {
			found = true
			break
		}
	}
	if !found {
		t.Error("accepted concept should survive rejection of the second concept")
	}
}
