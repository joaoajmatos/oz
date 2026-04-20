package audit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/oz-tools/oz/internal/audit"
	"github.com/oz-tools/oz/internal/audit/coverage"
	"github.com/oz-tools/oz/internal/audit/orphans"
	"github.com/oz-tools/oz/internal/audit/staleness"
	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/semantic"
	"github.com/oz-tools/oz/internal/testws"
)

// TestAuditE2E builds a real workspace, runs context.Build + Serialize to
// produce graph.json, then calls audit.RunAll with all stub checks and asserts:
//   - schema_version == "1"
//   - findings is empty (all stubs return nil)
//   - counts has all three keys, all zero
//   - JSON round-trip preserves the fields
func TestAuditE2E(t *testing.T) {
	ws := testws.New(t).
		WithAgent("test",
			testws.Role("Test agent for audit E2E"),
			testws.Scope("code/**"),
		).
		Build()

	root := ws.Path()

	// Build and serialize the context graph so checks that need it can load it.
	buildResult, err := ozcontext.Build(root)
	if err != nil {
		t.Fatalf("context.Build: %v", err)
	}
	if err := ozcontext.Serialize(root, buildResult.Graph); err != nil {
		t.Fatalf("context.Serialize: %v", err)
	}

	// All stub checks return no findings.
	stubs := []audit.Check{
		&orphansStub{},
		&coverageStub{},
		&stalenessStub{},
		&driftStub{},
	}

	r, err := audit.RunAll(root, stubs, audit.Options{})
	if err != nil {
		t.Fatalf("audit.RunAll: %v", err)
	}

	if r.SchemaVersion != "1" {
		t.Errorf("SchemaVersion = %q, want %q", r.SchemaVersion, "1")
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(r.Findings))
	}
	for _, sev := range []audit.Severity{audit.SeverityError, audit.SeverityWarn, audit.SeverityInfo} {
		if v, ok := r.Counts[sev]; !ok {
			t.Errorf("Counts missing key %q", sev)
		} else if v != 0 {
			t.Errorf("Counts[%q] = %d, want 0", sev, v)
		}
	}

	// JSON round-trip.
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var r2 audit.Report
	if err := json.Unmarshal(b, &r2); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if r2.SchemaVersion != "1" {
		t.Errorf("round-trip SchemaVersion = %q, want %q", r2.SchemaVersion, "1")
	}
	if r2.Findings == nil {
		t.Error("round-trip: Findings must not be nil")
	}
	if len(r2.Findings) != 0 {
		t.Errorf("round-trip: expected 0 findings, got %d", len(r2.Findings))
	}
}

// buildGraph is a test helper: builds and serializes the context graph.
func buildGraph(t *testing.T, root string) {
	t.Helper()
	result, err := ozcontext.Build(root)
	if err != nil {
		t.Fatalf("context.Build: %v", err)
	}
	if err := ozcontext.Serialize(root, result.Graph); err != nil {
		t.Fatalf("context.Serialize: %v", err)
	}
}

// hasCode reports whether any finding in fs has the given code.
func hasCode(fs []audit.Finding, code string) bool {
	for _, f := range fs {
		if f.Code == code {
			return true
		}
	}
	return false
}

// findingsForFile returns findings whose File field matches the given path.
func findingsForFile(fs []audit.Finding, file string) []audit.Finding {
	var out []audit.Finding
	for _, f := range fs {
		if f.File == file {
			out = append(out, f)
		}
	}
	return out
}

// TestOrphansE2E_UnreferencedSpec verifies that a spec not in any agent's
// read-chain produces an ORPH001 finding.
func TestOrphansE2E_UnreferencedSpec(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coding", testws.Role("Builds things")).
		WithSpec("specs/api.md", testws.Section("Overview", "API overview")).
		// Note: agent read-chain does NOT include specs/api.md.
		Build()

	buildGraph(t, ws.Path())

	r, err := audit.RunAll(ws.Path(), []audit.Check{&orphans.Check{}}, audit.Options{})
	if err != nil {
		t.Fatalf("audit.RunAll: %v", err)
	}

	if ff := findingsForFile(r.Findings, "specs/api.md"); len(ff) == 0 {
		t.Error("expected ORPH001 for unreferenced specs/api.md, got none")
	} else if !hasCode(ff, "ORPH001") {
		t.Errorf("expected ORPH001 for specs/api.md, got %+v", ff)
	}
}

// TestOrphansE2E_ReferencedSpec verifies that a spec in an agent's read-chain
// does NOT produce an ORPH001 finding.
func TestOrphansE2E_ReferencedSpec(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coding",
			testws.Role("Builds things"),
			testws.ReadChain("specs/api.md"),
		).
		WithSpec("specs/api.md", testws.Section("Overview", "API overview")).
		Build()

	buildGraph(t, ws.Path())

	r, err := audit.RunAll(ws.Path(), []audit.Check{&orphans.Check{}}, audit.Options{})
	if err != nil {
		t.Fatalf("audit.RunAll: %v", err)
	}

	for _, f := range r.Findings {
		if f.Code == "ORPH001" && f.File == "specs/api.md" {
			t.Errorf("unexpected ORPH001 for referenced specs/api.md: %+v", f)
		}
	}
}

// TestCoverageE2E_DanglingPath verifies that an agent scope path that does not
// exist on disk produces a COV001 finding.
func TestCoverageE2E_DanglingPath(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coding",
			testws.Role("Builds things"),
			// Scope path that will never exist on disk.
			testws.Scope("code/nonexistent-dir"),
		).
		Build()

	buildGraph(t, ws.Path())

	r, err := audit.RunAll(ws.Path(), []audit.Check{&coverage.Check{}}, audit.Options{})
	if err != nil {
		t.Fatalf("audit.RunAll: %v", err)
	}

	if !hasCode(r.Findings, "COV001") {
		t.Error("expected COV001 for dangling scope path, got none")
	}
}

// TestCoverageE2E_UnownedCodeDir verifies that a top-level code/ directory
// with no owning agent scope produces a COV002 finding.
func TestCoverageE2E_UnownedCodeDir(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coding",
			testws.Role("Builds things"),
			// Agent owns a different path, not the code/ directory we create below.
			testws.Scope("code/other/**"),
		).
		Build()

	// Create a code/ directory that no agent owns.
	if err := os.MkdirAll(filepath.Join(ws.Path(), "code", "unowned"), 0755); err != nil {
		t.Fatal(err)
	}

	buildGraph(t, ws.Path())

	r, err := audit.RunAll(ws.Path(), []audit.Check{&coverage.Check{}}, audit.Options{})
	if err != nil {
		t.Fatalf("audit.RunAll: %v", err)
	}

	found := false
	for _, f := range r.Findings {
		if f.Code == "COV002" && f.File == "code/unowned" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected COV002 for code/unowned/, findings: %+v", r.Findings)
	}
}

// TestCoverageE2E_OwnedCodeDir verifies that a code/ directory owned via a
// wildcard scope does NOT produce a COV002 finding.
func TestCoverageE2E_OwnedCodeDir(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coding",
			testws.Role("Builds things"),
			testws.Scope("code/**"),
		).
		Build()

	if err := os.MkdirAll(filepath.Join(ws.Path(), "code", "oz"), 0755); err != nil {
		t.Fatal(err)
	}

	buildGraph(t, ws.Path())

	r, err := audit.RunAll(ws.Path(), []audit.Check{&coverage.Check{}}, audit.Options{})
	if err != nil {
		t.Fatalf("audit.RunAll: %v", err)
	}

	for _, f := range r.Findings {
		if f.Code == "COV002" && f.File == "code/oz" {
			t.Errorf("unexpected COV002 for code/oz (owned via wildcard): %+v", f)
		}
	}
}

// --- A3-06: Staleness E2E cases ---

// TestStalenessE2E_Clean: build + audit → no STALE001.
func TestStalenessE2E_Clean(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coding", testws.Role("Builds things")).
		Build()
	root := ws.Path()

	buildGraph(t, root)

	r, err := audit.RunAll(root, []audit.Check{&staleness.Check{}}, audit.Options{})
	if err != nil {
		t.Fatalf("audit.RunAll: %v", err)
	}

	if hasCode(r.Findings, "STALE001") {
		t.Error("unexpected STALE001 after fresh build")
	}
}

// TestStalenessE2E_StaleGraph: build + add a new note file + audit → STALE001.
func TestStalenessE2E_StaleGraph(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coding", testws.Role("Builds things")).
		Build()
	root := ws.Path()

	buildGraph(t, root)

	// Add a new note file after the build; this creates a new graph node,
	// so the freshly-computed ContentHash will differ from graph.json.
	notesDir := filepath.Join(root, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		t.Fatalf("mkdir notes: %v", err)
	}
	notePath := filepath.Join(notesDir, "new-note.md")
	if err := os.WriteFile(notePath, []byte("## New note\n\nAdded after build.\n"), 0644); err != nil {
		t.Fatalf("write note: %v", err)
	}

	r, err := audit.RunAll(root, []audit.Check{&staleness.Check{}}, audit.Options{})
	if err != nil {
		t.Fatalf("audit.RunAll: %v", err)
	}

	if !hasCode(r.Findings, "STALE001") {
		t.Error("expected STALE001 after adding a file without rebuilding")
	}
}

// TestStalenessE2E_StaleOverlay: build + write semantic.json with wrong graph_hash → STALE002.
func TestStalenessE2E_StaleOverlay(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coding", testws.Role("Builds things")).
		Build()
	root := ws.Path()

	buildGraph(t, root)

	// Write a semantic.json with a wrong graph_hash.
	overlay := &semantic.Overlay{
		SchemaVersion: "1",
		GraphHash:     "wrong-hash-does-not-match",
		Concepts: []semantic.Concept{
			{ID: "concept:foo", Name: "foo", Tag: "EXTRACTED", Confidence: 1.0, Reviewed: true},
		},
	}
	if err := semantic.Write(root, overlay); err != nil {
		t.Fatalf("semantic.Write: %v", err)
	}

	r, err := audit.RunAll(root, []audit.Check{&staleness.Check{}}, audit.Options{})
	if err != nil {
		t.Fatalf("audit.RunAll: %v", err)
	}

	if !hasCode(r.Findings, "STALE002") {
		t.Error("expected STALE002 when semantic.json has wrong graph_hash")
	}
	if hasCode(r.Findings, "STALE001") {
		t.Error("unexpected STALE001 — graph.json should be fresh")
	}
}

// Stub check implementations for the E2E test.

type orphansStub struct{}

func (c *orphansStub) Name() string { return "orphans" }
func (c *orphansStub) Codes() []string {
	return []string{"ORPH001", "ORPH002", "ORPH003"}
}
func (c *orphansStub) Run(_ string, _ audit.Options) ([]audit.Finding, error) { return nil, nil }

type coverageStub struct{}

func (c *coverageStub) Name() string { return "coverage" }
func (c *coverageStub) Codes() []string {
	return []string{"COV001", "COV002", "COV003"}
}
func (c *coverageStub) Run(_ string, _ audit.Options) ([]audit.Finding, error) { return nil, nil }

type stalenessStub struct{}

func (c *stalenessStub) Name() string { return "staleness" }
func (c *stalenessStub) Codes() []string {
	return []string{"STALE001", "STALE002", "STALE003"}
}
func (c *stalenessStub) Run(_ string, _ audit.Options) ([]audit.Finding, error) { return nil, nil }

type driftStub struct{}

func (c *driftStub) Name() string { return "drift" }
func (c *driftStub) Codes() []string {
	return []string{"DRIFT001", "DRIFT002", "DRIFT003"}
}
func (c *driftStub) Run(_ string, _ audit.Options) ([]audit.Finding, error) { return nil, nil }
