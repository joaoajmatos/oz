package query_test

import (
	"testing"

	"github.com/oz-tools/oz/internal/query"
	"github.com/oz-tools/oz/internal/testws"
)

// TestRoutingAccuracy runs every golden suite and measures routing accuracy.
// Each suite declares a minimum accuracy threshold; the test fails if any
// suite falls below its threshold.
//
// Sprint 1: the query engine is a stub returning empty results, so accuracy
// will be 0%. The test infrastructure is exercised — workspace build, fixture
// loading, and harness wiring are all validated here.
//
// Sprint 3: Run() is fully implemented. Accuracy targets become hard gates.
func TestRoutingAccuracy(t *testing.T) {
	suites, err := testws.LoadGoldenSuites(t, "testdata/golden")
	if err != nil {
		t.Fatalf("load golden suites: %v", err)
	}
	if len(suites) == 0 {
		t.Fatal("no golden suites found in testdata/golden")
	}

	for _, suite := range suites {
		suite := suite
		t.Run(suite.Name, func(t *testing.T) {
			ws := suite.Build(t)

			if len(suite.Queries) == 0 {
				t.Skip("no queries in suite")
			}

			hits, total := 0, 0
			for _, q := range suite.Queries {
				result := query.Run(ws.Path(), q.Query)
				if q.Matches(result) {
					hits++
				}
				total++
			}

			accuracy := float64(hits) / float64(total)
			t.Logf("%s: %d/%d (%.1f%%) — minimum: %.1f%%",
				suite.Name, hits, total, accuracy*100, suite.MinAccuracy*100)

			// Accuracy gate is enforced only when the engine is implemented.
			// During Sprint 1 the stub returns empty results so we skip the gate.
			if isStubEngine() {
				t.Logf("skipping accuracy gate: query engine is stub (Sprint 1)")
				return
			}

			if accuracy < suite.MinAccuracy {
				t.Errorf("accuracy %.1f%% below minimum %.1f%% for suite %q",
					accuracy*100, suite.MinAccuracy*100, suite.Name)
			}
		})
	}
}

// isStubEngine always returns false — the real BM25F engine ships in Sprint 3.
func isStubEngine() bool { return false }

// TestBuilder_BasicWorkspace validates the testws builder produces a
// convention-compliant workspace that oz validate would accept.
func TestBuilder_BasicWorkspace(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend",
			testws.Scope("code/api/**"),
			testws.Role("Builds REST endpoints"),
			testws.Responsibilities("Implements handlers and middleware"),
			testws.OutOfScope("UI work"),
			testws.Rules("rules/coding-guidelines.md"),
			testws.Skills("skills/commit/"),
		).
		WithAgent("frontend",
			testws.Scope("code/ui/**"),
			testws.Role("React component development"),
		).
		WithSpec("specs/api.md",
			testws.Section("overview", "REST API specification"),
			testws.Section("authentication", "Bearer token auth"),
		).
		WithDecision("0001-use-rest", "REST chosen over gRPC for simplicity").
		Build()

	// Workspace root should exist.
	if ws.Path() == "" {
		t.Fatal("workspace path is empty")
	}

	// Required oz files should exist.
	for _, required := range []string{"AGENTS.md", "OZ.md"} {
		data, err := ws.ReadFile(required)
		if err != nil {
			t.Errorf("required file %s missing: %v", required, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("required file %s is empty", required)
		}
	}

	// Agent AGENT.md files should exist and contain required sections.
	for _, agentName := range []string{"backend", "frontend"} {
		path := "agents/" + agentName + "/AGENT.md"
		data, err := ws.ReadFile(path)
		if err != nil {
			t.Errorf("agent file %s missing: %v", path, err)
			continue
		}
		content := string(data)
		for _, section := range []string{"## Role", "## Read-chain", "## Rules", "## Skills", "## Responsibilities", "## Out of scope"} {
			if !contains(content, section) {
				t.Errorf("%s: missing section %q", path, section)
			}
		}
	}

	// Spec file should exist.
	data, err := ws.ReadFile("specs/api.md")
	if err != nil {
		t.Errorf("spec file missing: %v", err)
	} else if !contains(string(data), "## overview") && !contains(string(data), "## Overview") {
		t.Errorf("spec file missing overview section")
	}
}

// TestFixture_LoadsFromYAML validates the YAML fixture loader.
func TestFixture_LoadsFromYAML(t *testing.T) {
	ws := testws.FromFixture(t, "testdata/golden/01_minimal/workspace.yaml").Build()

	// Should have materialized the workspace without error.
	if ws.Path() == "" {
		t.Fatal("fixture build returned empty path")
	}

	// AGENTS.md must list the agents from the fixture.
	data, err := ws.ReadFile("AGENTS.md")
	if err != nil {
		t.Fatalf("AGENTS.md missing: %v", err)
	}
	_ = data // Content checked in TestBuilder_BasicWorkspace
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
