package audit_test

import (
	"encoding/json"
	"testing"

	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/audit"
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
