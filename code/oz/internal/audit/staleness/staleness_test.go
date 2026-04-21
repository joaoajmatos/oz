package staleness

import (
	"encoding/json"
	"testing"

	"github.com/joaoajmatos/oz/internal/audit"
	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/semantic"
	"github.com/joaoajmatos/oz/internal/testws"
)

// hasCode reports whether any finding in fs has the given code.
func hasCode(fs []audit.Finding, code string) bool {
	for _, f := range fs {
		if f.Code == code {
			return true
		}
	}
	return false
}

// makeGraphs returns a pair of identical graph.Graph values (simulating a
// fresh build whose hash matches the on-disk graph).
func makeGraphs(hash string) (ondisk, fresh *graph.Graph) {
	ondisk = &graph.Graph{ContentHash: hash}
	fresh = &graph.Graph{ContentHash: hash}
	return
}

// --- STALE001 ---

func TestSTALE001_HashMismatch(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")
	fresh.ContentHash = "xyz" // workspace changed

	fs := runCheck(ondisk, fresh, nil)

	if !hasCode(fs, "STALE001") {
		t.Error("expected STALE001 when content hash differs")
	}
}

func TestSTALE001_HashMatch_NoFinding(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")

	fs := runCheck(ondisk, fresh, nil)

	if hasCode(fs, "STALE001") {
		t.Error("unexpected STALE001 when hashes match")
	}
}

// --- STALE004 ---

func TestSTALE004_NoOverlay(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")

	fs := runCheck(ondisk, fresh, nil)

	if !hasCode(fs, "STALE004") {
		t.Error("expected STALE004 when overlay is absent")
	}
}

func TestSTALE004_Severity_IsInfo(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")

	fs := runCheck(ondisk, fresh, nil)

	for _, f := range fs {
		if f.Code == "STALE004" && f.Severity != audit.SeverityInfo {
			t.Errorf("STALE004 severity = %q, want info", f.Severity)
		}
	}
}

func TestSTALE004_NotEmitted_WhenOverlayPresent(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")
	overlay := &semantic.Overlay{GraphHash: "abc"}

	fs := runCheck(ondisk, fresh, overlay)

	if hasCode(fs, "STALE004") {
		t.Error("unexpected STALE004 when overlay is present")
	}
}

// --- STALE002 ---

func TestSTALE002_StaleOverlay(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")
	overlay := &semantic.Overlay{GraphHash: "old-hash"}

	fs := runCheck(ondisk, fresh, overlay)

	if !hasCode(fs, "STALE002") {
		t.Error("expected STALE002 when overlay graph_hash mismatches current graph")
	}
}

func TestSTALE002_FreshOverlay_NoFinding(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")
	overlay := &semantic.Overlay{GraphHash: "abc"}

	fs := runCheck(ondisk, fresh, overlay)

	if hasCode(fs, "STALE002") {
		t.Error("unexpected STALE002 when overlay is up to date")
	}
}

// --- STALE003 ---

func TestSTALE003_UnreviewedConcepts(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")
	overlay := &semantic.Overlay{
		GraphHash: "abc",
		Concepts: []semantic.Concept{
			{ID: "concept:foo", Tag: "EXTRACTED", Reviewed: false},
		},
	}

	fs := runCheck(ondisk, fresh, overlay)

	if !hasCode(fs, "STALE003") {
		t.Error("expected STALE003 for unreviewed concept")
	}
}

func TestSTALE003_UnreviewedEdges(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")
	overlay := &semantic.Overlay{
		GraphHash: "abc",
		Edges: []semantic.ConceptEdge{
			{From: "concept:foo", To: "agent:bar", Type: "agent_owns_concept", Tag: "EXTRACTED", Reviewed: false},
		},
	}

	fs := runCheck(ondisk, fresh, overlay)

	if !hasCode(fs, "STALE003") {
		t.Error("expected STALE003 for unreviewed edge")
	}
}

func TestSTALE003_AllReviewed_NoFinding(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")
	overlay := &semantic.Overlay{
		GraphHash: "abc",
		Concepts: []semantic.Concept{
			{ID: "concept:foo", Tag: "EXTRACTED", Reviewed: true},
		},
		Edges: []semantic.ConceptEdge{
			{From: "concept:foo", To: "agent:bar", Type: "agent_owns_concept", Tag: "EXTRACTED", Reviewed: true},
		},
	}

	fs := runCheck(ondisk, fresh, overlay)

	if hasCode(fs, "STALE003") {
		t.Error("unexpected STALE003 when all items are reviewed")
	}
}

func TestSTALE003_Count_InMessage(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")
	overlay := &semantic.Overlay{
		GraphHash: "abc",
		Concepts: []semantic.Concept{
			{ID: "concept:a", Tag: "EXTRACTED", Reviewed: false},
			{ID: "concept:b", Tag: "EXTRACTED", Reviewed: false},
		},
	}

	fs := runCheck(ondisk, fresh, overlay)

	for _, f := range fs {
		if f.Code == "STALE003" {
			// Message must contain the count.
			if f.Message == "" {
				t.Error("STALE003 message is empty")
			}
			return
		}
	}
	t.Error("STALE003 not found")
}

// --- Clean: no findings when graph is fresh and overlay is up to date ---

func TestRunCheck_Clean(t *testing.T) {
	ondisk, fresh := makeGraphs("abc")
	overlay := &semantic.Overlay{
		GraphHash: "abc",
		Concepts:  []semantic.Concept{{ID: "concept:foo", Tag: "EXTRACTED", Reviewed: true}},
	}

	fs := runCheck(ondisk, fresh, overlay)

	if len(fs) != 0 {
		t.Errorf("expected no findings, got %+v", fs)
	}
}

// --- A3-05: determinism test (resolves AT-02) ---

// TestDeterminism_A3_05 builds a workspace, serializes graph.json, then runs
// the staleness check 100 times asserting byte-identical findings every run.
// STALE001 must never fire on a fresh build.
func TestDeterminism_A3_05(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coding", testws.Role("Builds things")).
		Build()
	root := ws.Path()

	result, err := ozcontext.Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := ozcontext.Serialize(root, result.Graph); err != nil {
		t.Fatalf("Serialize: %v", err)
	}

	c := &Check{}

	var ref []byte
	for i := 0; i < 100; i++ {
		findings, err := c.Run(root, audit.Options{})
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}

		if hasCode(findings, "STALE001") {
			t.Fatalf("run %d: STALE001 fired on fresh build — determinism bug", i)
		}

		b, err := json.Marshal(findings)
		if err != nil {
			t.Fatalf("marshal run %d: %v", i, err)
		}
		if i == 0 {
			ref = b
			continue
		}
		if string(b) != string(ref) {
			t.Fatalf("run %d: findings not byte-identical\ngot:  %s\nwant: %s", i, b, ref)
		}
	}
}
