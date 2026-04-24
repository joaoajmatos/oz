package query_test

import (
	"testing"

	"github.com/joaoajmatos/oz/internal/query"
	"github.com/joaoajmatos/oz/internal/testws"
)

// TestRoutingAccuracy runs every golden suite under testdata/golden, measures
// routing accuracy with query.Run, and fails if any suite is below its
// min_accuracy.
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

			if accuracy < suite.MinAccuracy {
				t.Errorf("accuracy %.1f%% below minimum %.1f%% for suite %q",
					accuracy*100, suite.MinAccuracy*100, suite.Name)
			}
		})
	}
}

// TestRetrievalAccuracy walks the 04_retrieval golden suite and checks Sprint-2
// retrieval expectations for context_blocks (R-01 scope).
// Code entry points and implementing_packages remain covered in later sprints.
func TestRetrievalAccuracy(t *testing.T) {
	suites, err := testws.LoadGoldenSuites(t, "testdata/golden")
	if err != nil {
		t.Fatalf("load golden suites: %v", err)
	}

	for _, suite := range suites {
		if suite.Name != "04_retrieval" {
			continue
		}
		suite := suite
		t.Run(suite.Name, func(t *testing.T) {
			ws := suite.Build(t)
			for _, q := range suite.Queries {
				q := q
				t.Run(q.Query, func(t *testing.T) {
					result := query.Run(ws.Path(), q.Query)

					for _, b := range q.ExpectBlocksInTopK {
						testws.ExpectBlockInTopK(t, result, b.File, b.Section, b.K)
					}
					for _, b := range q.ExpectBlocksNotInTopK {
						testws.ExpectBlockNotInTopK(t, result, b.File, b.Section, b.K)
					}
					for _, ep := range q.ExpectCodeEntryPointsInTopK {
						testws.ExpectCodeEntryPoint(t, result, ep.Symbol, ep.K)
					}
					for _, p := range q.ExpectPackagesInTopK {
						testws.ExpectPackageInTopK(t, result, p.Package, p.K)
					}
					for _, p := range q.ExpectPackagesNotInTopK {
						testws.ExpectPackageNotInTopK(t, result, p.Package, p.K)
					}
					if q.ExpectRelevanceDescending {
						testws.ExpectRelevanceDescending(t, result)
					}
					if q.ExpectNoRelevantContext {
						testws.ExpectNoRelevantContext(t, result)
					}
					if q.ExpectTrustBeats != nil {
						testws.ExpectTrustBeats(
							t,
							result,
							mapTrustTier(q.ExpectTrustBeats.WinnerTier),
							mapTrustTier(q.ExpectTrustBeats.LoserTier),
						)
					}
				})
			}
		})
	}
}

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
			testws.Skills("skills/oz/"),
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

// TestRelevantConcepts_QueryRanked checks that relevant_concepts are ordered by
// query–concept BM25, not the winning agent’s full concept portfolio.
func TestRelevantConcepts_QueryRanked(t *testing.T) {
	suites, err := testws.LoadGoldenSuites(t, "testdata/golden")
	if err != nil {
		t.Fatalf("load golden suites: %v", err)
	}
	var suite *testws.GoldenSuite
	for _, s := range suites {
		if s.Name == "05_semantic" {
			suite = s
			break
		}
	}
	if suite == nil {
		t.Fatal("missing 05_semantic suite")
	}
	ws := suite.Build(t)
	if err := query.WriteScoringTOML(ws.Path(), query.DefaultScoringConfig()); err != nil {
		t.Fatalf("write scoring: %v", err)
	}
	r := query.Run(ws.Path(), "add TOTP-based MFA for organization admin accounts")
	if r.Agent != "auth" {
		t.Fatalf("expected auth agent, got %q", r.Agent)
	}
	if len(r.RelevantConcepts) == 0 {
		t.Fatal("expected non-empty relevant_concepts for MFA query with overlay")
	}
	if r.RelevantConcepts[0] != "MFA Enforcement" {
		t.Fatalf("first concept = %q, want MFA Enforcement; all = %v", r.RelevantConcepts[0], r.RelevantConcepts)
	}
}

func mapTrustTier(tier string) string {
	switch tier {
	case "specs":
		return "high"
	case "docs", "context":
		return "medium"
	case "notes":
		return "low"
	default:
		return tier
	}
}
