package review_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oz-tools/oz/internal/review"
	"github.com/oz-tools/oz/internal/semantic"
	"github.com/oz-tools/oz/internal/testws"
)

// writeSemanticJSON writes a full semantic.json to the workspace's context/ dir.
func writeSemanticJSON(t *testing.T, root string, o *semantic.Overlay) {
	t.Helper()
	if err := semantic.Write(root, o); err != nil {
		t.Fatalf("write semantic.json: %v", err)
	}
}

// overlay with two unreviewed concepts and one unreviewed edge.
func unreviewedOverlay() *semantic.Overlay {
	return &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		GraphHash:     "hash1",
		Concepts: []semantic.Concept{
			{ID: "concept:rest-api", Name: "REST API", Tag: semantic.TagExtracted, Confidence: 1.0},
			{ID: "concept:auth", Name: "Auth", Tag: semantic.TagInferred, Confidence: 0.8},
		},
		Edges: []semantic.ConceptEdge{
			{From: "concept:rest-api", To: "agent:backend", Type: semantic.EdgeTypeAgentOwnsConcept,
				Tag: semantic.TagExtracted, Confidence: 1.0},
		},
	}
}

// --- Scenario 1: --accept-all accepts every unreviewed item ------------------

func TestReview_AcceptAll(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("Builds REST endpoints")).
		Build()
	writeSemanticJSON(t, ws.Path(), unreviewedOverlay())

	var out bytes.Buffer
	summary, err := review.Run(ws.Path(), review.Options{
		AcceptAll: true,
		Out:       &out,
	})
	if err != nil {
		t.Fatalf("review.Run: %v", err)
	}
	if summary.Accepted != 3 {
		t.Errorf("accepted = %d, want 3", summary.Accepted)
	}
	if summary.Rejected != 0 {
		t.Errorf("rejected = %d, want 0", summary.Rejected)
	}

	// Reload and verify all items are now reviewed.
	reloaded, err := semantic.Load(ws.Path())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, c := range reloaded.Concepts {
		if !c.Reviewed {
			t.Errorf("concept %q should be reviewed after --accept-all", c.Name)
		}
	}
	for _, e := range reloaded.Edges {
		if !e.Reviewed {
			t.Errorf("edge %s→%s should be reviewed after --accept-all", e.From, e.To)
		}
	}
}

// --- Scenario 2: nothing to review -------------------------------------------

func TestReview_NothingToReview(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("Builds REST endpoints")).
		Build()

	all := unreviewedOverlay()
	for i := range all.Concepts {
		all.Concepts[i].Reviewed = true
	}
	for i := range all.Edges {
		all.Edges[i].Reviewed = true
	}
	writeSemanticJSON(t, ws.Path(), all)

	var out bytes.Buffer
	summary, err := review.Run(ws.Path(), review.Options{AcceptAll: true, Out: &out})
	if err != nil {
		t.Fatalf("review.Run: %v", err)
	}
	if !summary.NothingToReview {
		t.Error("expected NothingToReview = true when all items are reviewed")
	}
	if !strings.Contains(out.String(), "nothing to review") {
		t.Errorf("expected 'nothing to review' message, got: %q", out.String())
	}
}

// --- Scenario 3: missing semantic.json returns error --------------------------

func TestReview_NoSemanticJSON(t *testing.T) {
	dir := t.TempDir()
	// create a minimal workspace so findWorkspaceRoot-equivalent doesn't fail,
	// but no semantic.json.
	if err := os.MkdirAll(filepath.Join(dir, "context"), 0755); err != nil {
		t.Fatal(err)
	}
	_, err := review.Run(dir, review.Options{AcceptAll: true, Out: &bytes.Buffer{}})
	if err == nil {
		t.Error("expected error when semantic.json is absent")
	}
}

// --- Scenario 4: interactive accept -------------------------------------------

func TestReview_Interactive_AcceptAll(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("Builds REST endpoints")).
		Build()
	writeSemanticJSON(t, ws.Path(), unreviewedOverlay())

	// Provide "y\ny\ny\n" — accept all three items interactively.
	input := strings.NewReader("y\ny\ny\n")
	var out bytes.Buffer

	summary, err := review.Run(ws.Path(), review.Options{
		AcceptAll: false,
		In:        input,
		Out:       &out,
	})
	if err != nil {
		t.Fatalf("review.Run: %v", err)
	}
	if summary.Accepted != 3 {
		t.Errorf("accepted = %d, want 3", summary.Accepted)
	}
}

// --- Scenario 5: interactive reject -------------------------------------------

func TestReview_Interactive_RejectAll(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("Builds REST endpoints")).
		Build()
	writeSemanticJSON(t, ws.Path(), unreviewedOverlay())

	// Provide "n\nn\nn\n" — reject all three items.
	input := strings.NewReader("n\nn\nn\n")
	var out bytes.Buffer

	summary, err := review.Run(ws.Path(), review.Options{
		AcceptAll: false,
		In:        input,
		Out:       &out,
	})
	if err != nil {
		t.Fatalf("review.Run: %v", err)
	}
	if summary.Rejected != 3 {
		t.Errorf("rejected = %d, want 3", summary.Rejected)
	}
	// File should now have no concepts or edges.
	reloaded, err := semantic.Load(ws.Path())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Concepts) != 0 {
		t.Errorf("rejected concepts should be removed, got %d", len(reloaded.Concepts))
	}
	if len(reloaded.Edges) != 0 {
		t.Errorf("rejected edges should be removed, got %d", len(reloaded.Edges))
	}
}

// --- Scenario 6: quit mid-review ---------------------------------------------

func TestReview_Interactive_Quit(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("Builds REST endpoints")).
		Build()
	writeSemanticJSON(t, ws.Path(), unreviewedOverlay())

	// Accept first, then quit — only first item should be accepted.
	input := strings.NewReader("y\nq\n")
	var out bytes.Buffer

	summary, err := review.Run(ws.Path(), review.Options{
		AcceptAll: false,
		In:        input,
		Out:       &out,
	})
	if err != nil {
		t.Fatalf("review.Run: %v", err)
	}
	if summary.Accepted != 1 {
		t.Errorf("accepted = %d, want 1 (accepted first concept then quit)", summary.Accepted)
	}
}

// --- Scenario 7: review preserves graph_hash and metadata --------------------

func TestReview_AcceptAll_PreservesMetadata(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("Builds REST endpoints")).
		Build()
	o := unreviewedOverlay()
	o.Model = "anthropic/claude-haiku-4"
	writeSemanticJSON(t, ws.Path(), o)

	var out bytes.Buffer
	if _, err := review.Run(ws.Path(), review.Options{AcceptAll: true, Out: &out}); err != nil {
		t.Fatalf("review.Run: %v", err)
	}

	reloaded, err := semantic.Load(ws.Path())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.GraphHash != "hash1" {
		t.Errorf("graph_hash changed after review, got %q", reloaded.GraphHash)
	}
	if reloaded.Model != "anthropic/claude-haiku-4" {
		t.Errorf("model changed after review, got %q", reloaded.Model)
	}
}

// --- Scenario 8: diff view table output --------------------------------------

func TestReview_DiffView_TableOutput(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("Builds REST endpoints")).
		Build()
	writeSemanticJSON(t, ws.Path(), unreviewedOverlay())

	var out bytes.Buffer
	review.Run(ws.Path(), review.Options{AcceptAll: true, Out: &out}) //nolint
	output := out.String()

	if !strings.Contains(output, "Unreviewed concepts") {
		t.Errorf("output should contain 'Unreviewed concepts', got: %q", output)
	}
	if !strings.Contains(output, "REST API") {
		t.Errorf("output should list concept name 'REST API', got: %q", output)
	}
	if !strings.Contains(output, "Unreviewed edges") {
		t.Errorf("output should contain 'Unreviewed edges', got: %q", output)
	}
}

// --- Scenario 9: accept-all output is valid JSON when reloaded ---------------

func TestReview_AcceptAll_WritesValidJSON(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("Builds REST endpoints")).
		Build()
	writeSemanticJSON(t, ws.Path(), unreviewedOverlay())

	review.Run(ws.Path(), review.Options{AcceptAll: true, Out: &bytes.Buffer{}}) //nolint

	data, err := os.ReadFile(filepath.Join(ws.Path(), "context", "semantic.json"))
	if err != nil {
		t.Fatalf("read semantic.json: %v", err)
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("semantic.json is not valid JSON after review: %v", err)
	}
}
