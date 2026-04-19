package testws_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oz-tools/oz/internal/testws"
)

// TestBuilder_CreatesRequiredOZFiles verifies that Build() produces a
// convention-compliant workspace: AGENTS.md, OZ.md, and all required dirs.
func TestBuilder_CreatesRequiredOZFiles(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("Builds the API")).
		Build()

	for _, f := range []string{"AGENTS.md", "OZ.md"} {
		if _, err := os.Stat(filepath.Join(ws.Path(), f)); err != nil {
			t.Errorf("required file %s missing: %v", f, err)
		}
	}

	for _, dir := range []string{"agents", "specs", "docs", "context", "rules", "notes"} {
		if _, err := os.Stat(filepath.Join(ws.Path(), dir)); err != nil {
			t.Errorf("required directory %s missing: %v", dir, err)
		}
	}
}

// TestBuilder_AgentMDHasAllSections verifies each AGENT.md contains the
// seven required convention sections.
func TestBuilder_AgentMDHasAllSections(t *testing.T) {
	ws := testws.New(t).
		WithAgent("alpha",
			testws.Role("Does alpha things"),
			testws.Scope("code/alpha/**"),
			testws.Responsibilities("Owns alpha subsystem"),
			testws.OutOfScope("Beta work"),
			testws.Rules("rules/coding-guidelines.md"),
			testws.Skills("skills/commit/"),
		).
		Build()

	data, err := ws.ReadFile("agents/alpha/AGENT.md")
	if err != nil {
		t.Fatalf("AGENT.md missing: %v", err)
	}
	content := string(data)

	required := []string{
		"## Role",
		"## Read-chain",
		"## Rules",
		"## Skills",
		"## Responsibilities",
		"## Out of scope",
		"## Context topics",
	}
	for _, section := range required {
		if !strings.Contains(content, section) {
			t.Errorf("AGENT.md missing section %q", section)
		}
	}
}

// TestBuilder_AgentScopeInResponsibilities verifies scope paths land in
// the Responsibilities section so the query engine can find them.
func TestBuilder_AgentScopeInResponsibilities(t *testing.T) {
	ws := testws.New(t).
		WithAgent("infra",
			testws.Scope("infra/**", "deploy/**"),
		).
		Build()

	data, err := ws.ReadFile("agents/infra/AGENT.md")
	if err != nil {
		t.Fatalf("AGENT.md missing: %v", err)
	}
	content := string(data)

	for _, scope := range []string{"infra/**", "deploy/**"} {
		if !strings.Contains(content, scope) {
			t.Errorf("scope path %q not found in AGENT.md", scope)
		}
	}
}

// TestBuilder_SpecFileWritten verifies WithSpec writes the file with sections.
func TestBuilder_SpecFileWritten(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("API")).
		WithSpec("specs/api.md",
			testws.Section("Overview", "REST API over HTTP"),
			testws.Section("Authentication", "Bearer token"),
		).
		Build()

	data, err := ws.ReadFile("specs/api.md")
	if err != nil {
		t.Fatalf("spec file missing: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "REST API over HTTP") {
		t.Error("spec content missing from written file")
	}
	if !strings.Contains(content, "Bearer token") {
		t.Error("spec section content missing from written file")
	}
}

// TestBuilder_SemanticOverlayWritten verifies WithSemanticOverlay writes
// context/semantic.json with the correct structure.
func TestBuilder_SemanticOverlayWritten(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("API")).
		WithSemanticOverlay(testws.SemanticOverlay{
			GraphHash: "abc123",
			Concepts: []testws.OverlayConcept{
				{Name: "authentication", OwnedBy: "backend"},
				{Name: "pagination", OwnedBy: "backend"},
			},
		}).
		Build()

	data, err := ws.ReadFile("context/semantic.json")
	if err != nil {
		t.Fatalf("semantic.json missing: %v", err)
	}

	var overlay testws.SemanticOverlay
	if err := json.Unmarshal(data, &overlay); err != nil {
		t.Fatalf("semantic.json is not valid JSON: %v", err)
	}

	if overlay.GraphHash != "abc123" {
		t.Errorf("graph_hash: got %q, want %q", overlay.GraphHash, "abc123")
	}
	if len(overlay.Concepts) != 2 {
		t.Errorf("concepts count: got %d, want 2", len(overlay.Concepts))
	}
	if overlay.Concepts[0].Name != "authentication" {
		t.Errorf("concept name: got %q, want %q", overlay.Concepts[0].Name, "authentication")
	}
}

// TestBuilder_CleanupIsIdempotent verifies Cleanup() can be called safely
// even though t.TempDir() already handles removal.
func TestBuilder_CleanupIsIdempotent(t *testing.T) {
	ws := testws.New(t).
		WithAgent("a", testws.Role("test")).
		Build()

	// First call should succeed.
	ws.Cleanup()
	// Second call should not panic.
	ws.Cleanup()
}

// TestFixture_RoundTrip verifies that a workspace.yaml fixture produces a
// workspace whose AGENT.md files contain content from the fixture.
func TestFixture_RoundTrip(t *testing.T) {
	// Write a temporary fixture file.
	dir := t.TempDir()
	fixtureYAML := `
agents:
  - name: roundtrip
    role: "Round-trip test agent"
    scope: ["code/rt/**"]
    responsibilities: "Tests the fixture loader"
    out_of_scope: "Everything else"
    rules: ["rules/coding-guidelines.md"]
    skills: ["skills/commit/"]
specs:
  - path: "specs/rt.md"
    sections:
      - heading: "Overview"
        content: "Round-trip spec content"
`
	fixturePath := filepath.Join(dir, "workspace.yaml")
	if err := os.WriteFile(fixturePath, []byte(fixtureYAML), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	ws := testws.FromFixture(t, fixturePath).Build()

	agentData, err := ws.ReadFile("agents/roundtrip/AGENT.md")
	if err != nil {
		t.Fatalf("AGENT.md missing: %v", err)
	}
	if !strings.Contains(string(agentData), "Round-trip test agent") {
		t.Error("agent role not in AGENT.md")
	}

	specData, err := ws.ReadFile("specs/rt.md")
	if err != nil {
		t.Fatalf("spec file missing: %v", err)
	}
	if !strings.Contains(string(specData), "Round-trip spec content") {
		t.Error("spec content not in spec file")
	}
}

// TestBuilder_MultipleAgents verifies multiple agents are all materialized.
func TestBuilder_MultipleAgents(t *testing.T) {
	agents := []string{"alpha", "beta", "gamma"}
	b := testws.New(t)
	for _, name := range agents {
		b.WithAgent(name, testws.Role("Agent "+name))
	}
	ws := b.Build()

	for _, name := range agents {
		path := "agents/" + name + "/AGENT.md"
		if _, err := ws.ReadFile(path); err != nil {
			t.Errorf("agent %s: AGENT.md missing: %v", name, err)
		}
	}
}
