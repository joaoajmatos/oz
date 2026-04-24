package validate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/testws"
	"github.com/joaoajmatos/oz/internal/validate"
	"github.com/joaoajmatos/oz/internal/workspace"
)

// validWorkspace scaffolds a minimal workspace that passes all checks.
func validWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	for _, d := range []string{"agents", "specs/decisions", "docs", "context", "skills", "rules", "notes"} {
		mkdir(t, filepath.Join(dir, d))
	}

	write(t, filepath.Join(dir, "AGENTS.md"), "# AGENTS.md\n")
	write(t, filepath.Join(dir, "OZ.md"), "oz standard: v0.1\nproject: test\ndescription: test\n")
	write(t, filepath.Join(dir, "README.md"), "# test\n")

	agentDir := filepath.Join(dir, "agents", "coding")
	mkdir(t, agentDir)
	write(t, filepath.Join(agentDir, "AGENT.md"), "# coding Agent\n\n## Role\n\n## Read-chain\n\n## Rules\n\n## Skills\n\n## Responsibilities\n\n## Out of scope\n\n## Context topics\n")

	return dir
}

func runValidate(t *testing.T, dir string) *validate.Result {
	t.Helper()
	ws, err := workspace.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	return validate.Validate(ws)
}

func TestValidate_ValidWorkspace(t *testing.T) {
	result := runValidate(t, validWorkspace(t))
	if !result.Valid() {
		t.Errorf("expected valid workspace, got findings: %v", result.Findings)
	}
}

func TestValidate_MissingRequiredFile_AGENTSmd(t *testing.T) {
	dir := validWorkspace(t)
	os.Remove(filepath.Join(dir, "AGENTS.md"))
	requireError(t, runValidate(t, dir), "AGENTS.md")
}

func TestValidate_MissingRequiredFile_OZmd(t *testing.T) {
	dir := validWorkspace(t)
	os.Remove(filepath.Join(dir, "OZ.md"))
	requireError(t, runValidate(t, dir), "OZ.md")
}

func TestValidate_MissingRecommendedFile_IsWarning(t *testing.T) {
	dir := validWorkspace(t)
	os.Remove(filepath.Join(dir, "README.md"))
	result := runValidate(t, dir)
	if !result.Valid() {
		t.Error("missing recommended file should not fail validation")
	}
	requireWarning(t, result, "README.md")
}

func TestValidate_MissingRequiredDirectories(t *testing.T) {
	for _, d := range []string{"agents", "specs/decisions", "docs", "context", "skills", "rules", "notes"} {
		t.Run(d, func(t *testing.T) {
			dir := validWorkspace(t)
			os.RemoveAll(filepath.Join(dir, d))
			requireError(t, runValidate(t, dir), d)
		})
	}
}

func TestValidate_MissingRecommendedDirectory_IsWarning(t *testing.T) {
	dir := validWorkspace(t)
	// code/ is recommended and was never created in validWorkspace
	result := runValidate(t, dir)
	requireWarning(t, result, "code/")
	for _, f := range result.Findings {
		if f.Severity == validate.Error {
			t.Errorf("unexpected error finding: %s", f.Message)
		}
	}
}

func TestValidate_OZmd_MissingVersionField(t *testing.T) {
	dir := validWorkspace(t)
	write(t, filepath.Join(dir, "OZ.md"), "project: test\ndescription: no version\n")
	requireError(t, runValidate(t, dir), "oz standard")
}

func TestValidate_OZmd_EmptyVersionField(t *testing.T) {
	dir := validWorkspace(t)
	write(t, filepath.Join(dir, "OZ.md"), "oz standard: \nproject: test\n")
	requireError(t, runValidate(t, dir), "oz standard")
}

func TestValidate_Agent_MissingAGENTmd(t *testing.T) {
	dir := validWorkspace(t)
	os.Remove(filepath.Join(dir, "agents", "coding", "AGENT.md"))
	requireError(t, runValidate(t, dir), "AGENT.md")
}

func TestValidate_Agent_MissingRequiredSections(t *testing.T) {
	allSections := []string{
		"## Role",
		"## Read-chain",
		"## Rules",
		"## Skills",
		"## Responsibilities",
		"## Out of scope",
		"## Context topics",
	}
	full := "# coding Agent\n\n## Role\n\n## Read-chain\n\n## Rules\n\n## Skills\n\n## Responsibilities\n\n## Out of scope\n\n## Context topics\n"
	for _, section := range allSections {
		t.Run(section, func(t *testing.T) {
			dir := validWorkspace(t)
			write(t, filepath.Join(dir, "agents", "coding", "AGENT.md"),
				strings.ReplaceAll(full, section+"\n", ""))
			requireError(t, runValidate(t, dir), section)
		})
	}
}

func TestResult_Valid_OnlyWarnings(t *testing.T) {
	r := &validate.Result{Findings: []validate.Finding{
		{Severity: validate.Warning, Message: "some warning"},
	}}
	if !r.Valid() {
		t.Error("expected Valid() == true with only warnings")
	}
}

func TestResult_Valid_WithError(t *testing.T) {
	r := &validate.Result{Findings: []validate.Finding{
		{Severity: validate.Warning, Message: "warning"},
		{Severity: validate.Error, Message: "error"},
	}}
	if r.Valid() {
		t.Error("expected Valid() == false when an error is present")
	}
}

// helpers

func requireError(t *testing.T, result *validate.Result, substr string) {
	t.Helper()
	for _, f := range result.Findings {
		if f.Severity == validate.Error && strings.Contains(f.Message, substr) {
			return
		}
	}
	t.Errorf("expected an error finding containing %q\nfindings: %v", substr, result.Findings)
}

func requireWarning(t *testing.T, result *validate.Result, substr string) {
	t.Helper()
	for _, f := range result.Findings {
		if f.Severity == validate.Warning && strings.Contains(f.Message, substr) {
			return
		}
	}
	t.Errorf("expected a warning finding containing %q\nfindings: %v", substr, result.Findings)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

// TestValidate_TestwsFixture_Valid verifies that a testws-built workspace
// (which includes all 7 required AGENT.md sections) passes validation.
func TestValidate_TestwsFixture_Valid(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend",
			testws.Role("Builds REST endpoints"),
			testws.Scope("code/api/**"),
		).
		Build()

	result := runValidate(t, ws.Path())
	if !result.Valid() {
		t.Errorf("testws fixture should pass validation, got findings: %v", result.Findings)
	}
}

// TestValidate_SemanticOverlay_UnreviewedItems verifies that semantic.json with
// unreviewed items produces a warning (S6-04).
func TestValidate_SemanticOverlay_UnreviewedItems(t *testing.T) {
	unreviewed := false
	ws := testws.New(t).
		WithAgent("backend", testws.Role("Builds REST endpoints")).
		WithSemanticOverlay(testws.SemanticOverlay{
			Concepts: []testws.OverlayConcept{
				{Name: "REST API", OwnedBy: "backend", Reviewed: &unreviewed},
			},
		}).
		Build()

	result := runValidate(t, ws.Path())
	// Workspace must still be valid (warning does not affect exit code).
	if !result.Valid() {
		t.Errorf("unreviewed semantic items should not fail validation, got: %v", result.Findings)
	}
	requireWarning(t, result, "unreviewed")
}

// TestValidate_SemanticOverlay_AllReviewed verifies that semantic.json with all
// items already reviewed does NOT produce a warning (S6-04).
func TestValidate_SemanticOverlay_AllReviewed(t *testing.T) {
	dir := t.TempDir()

	for _, d := range []string{"agents", "specs/decisions", "docs", "context", "skills", "rules", "notes"} {
		mkdir(t, filepath.Join(dir, d))
	}
	write(t, filepath.Join(dir, "AGENTS.md"), "# AGENTS.md\n")
	write(t, filepath.Join(dir, "OZ.md"), "oz standard: v0.1\nproject: test\ndescription: test\n")
	write(t, filepath.Join(dir, "README.md"), "# test\n")
	agentDir := filepath.Join(dir, "agents", "coding")
	mkdir(t, agentDir)
	write(t, filepath.Join(agentDir, "AGENT.md"),
		"# coding Agent\n\n## Role\n\n## Read-chain\n\n## Rules\n\n## Skills\n\n## Responsibilities\n\n## Out of scope\n\n## Context topics\n")

	// Write a semantic.json where every item is reviewed.
	semJSON := `{
		"schema_version": "1",
		"graph_hash": "abc",
		"concepts": [{"id":"concept:foo","name":"Foo","tag":"EXTRACTED","confidence":1,"reviewed":true}],
		"edges": []
	}`
	write(t, filepath.Join(dir, "context", "semantic.json"), semJSON)

	result := runValidate(t, dir)
	for _, f := range result.Findings {
		if strings.Contains(f.Message, "unreviewed") {
			t.Errorf("unexpected unreviewed warning when all items are reviewed: %s", f.Message)
		}
	}
}

// TestValidate_SemanticOverlay_MissingFile passes cleanly when semantic.json is absent.
func TestValidate_SemanticOverlay_MissingFile(t *testing.T) {
	result := runValidate(t, validWorkspace(t))
	if !result.Valid() {
		t.Errorf("expected valid workspace with no semantic.json, got: %v", result.Findings)
	}
	for _, f := range result.Findings {
		if strings.Contains(f.Message, "unreviewed") {
			t.Errorf("unexpected unreviewed warning when semantic.json is absent: %s", f.Message)
		}
	}
}

// TestValidate_TestwsFixture_InvalidAgent confirms that an agent missing
// required sections produces error findings.
func TestValidate_TestwsFixture_InvalidAgent(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Role("Builds REST endpoints")).
		Build()

	// Overwrite the generated AGENT.md with one missing Rules and Skills.
	agentMD := filepath.Join(ws.Path(), "agents", "backend", "AGENT.md")
	write(t, agentMD, "# backend Agent\n\n## Role\n\n## Read-chain\n\n## Responsibilities\n\n## Out of scope\n\n## Context topics\n")

	result := runValidate(t, ws.Path())
	requireError(t, result, "## Rules")
	requireError(t, result, "## Skills")
}
