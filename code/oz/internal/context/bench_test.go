package context_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/scaffold"
	"github.com/oz-tools/oz/internal/testws"
)

// BenchmarkBuild_50Files benchmarks oz context build against a workspace with
// ~50 convention files (the S7-01 performance target is < 500ms).
//
// Run with:
//
//	go test -bench=BenchmarkBuild_50Files ./internal/context/ -benchtime=5s
func BenchmarkBuild_50Files(b *testing.B) {
	root := buildLargeWorkspaceOnDisk(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ozcontext.Build(root)
		if err != nil {
			b.Fatalf("Build: %v", err)
		}
	}
}

// BenchmarkBuild_50Files_WithSerialize benchmarks the full build+serialize path.
func BenchmarkBuild_50Files_WithSerialize(b *testing.B) {
	root := buildLargeWorkspaceOnDisk(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ozcontext.Build(root)
		if err != nil {
			b.Fatalf("Build: %v", err)
		}
		if err := ozcontext.Serialize(root, result.Graph); err != nil {
			b.Fatalf("Serialize: %v", err)
		}
	}
}

// TestBuild_50Files_UnderTarget verifies the build completes in < 500ms on
// a ~50-file workspace. Uses a single run — call the full benchmark with
// -bench for statistical confidence.
func TestBuild_50Files_UnderTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing test in short mode")
	}
	ws := buildLargeWorkspace(t)

	start := time.Now()
	_, err := ozcontext.Build(ws.Path())
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	const targetMs = 500
	t.Logf("oz context build on ~50-file workspace: %dms (target: <%dms)", elapsed, targetMs)
	if elapsed >= targetMs {
		t.Errorf("build took %dms, target is <%dms — profile and optimise before shipping", elapsed, targetMs)
	}
}

// buildLargeWorkspace creates a workspace with roughly 50 convention files
// using the testws builder (test context only, not benchmarks).
func buildLargeWorkspace(t *testing.T) *testws.Workspace {
	t.Helper()

	agents := []struct {
		name  string
		scope string
		role  string
	}{
		{"backend", "code/api/**", "Builds the REST API and business logic layer"},
		{"frontend", "code/ui/**", "Builds the React web application"},
		{"mobile", "code/ios/**", "Builds iOS and Android mobile clients"},
		{"infra", "infra/**", "Manages cloud infrastructure and CI/CD pipelines"},
		{"data", "code/pipelines/**", "Builds data pipelines and analytics"},
		{"auth", "code/auth/**", "Owns authentication and session management"},
		{"docs-agent", "docs/**", "Maintains product documentation and user guides"},
		{"design", "design/**", "Owns the design system and visual language"},
		{"qa", "tests/**", "Owns test automation and quality assurance"},
		{"security", "security/**", "Owns security review and vulnerability management"},
	}

	b := testws.New(t)
	for _, a := range agents {
		b.WithAgent(a.name,
			testws.Role(a.role),
			testws.Scope(a.scope),
			testws.Responsibilities("Implements features in scope. Reviews PRs. Maintains documentation."),
			testws.OutOfScope("Work outside declared scope paths."),
			testws.Rules("rules/coding-guidelines.md"),
			testws.Skills("skills/oz/"),
		)
	}
	addLargeWorkspaceContent(b)
	return b.Build()
}

// buildLargeWorkspaceOnDisk creates the same large workspace using scaffold
// directly (safe for use in benchmarks where no *testing.T is available).
func buildLargeWorkspaceOnDisk(tb testing.TB) string {
	tb.Helper()

	root := tb.TempDir()

	agentCfgs := []scaffold.AgentConfig{
		{Name: "backend", Description: "Builds the REST API", Type: "coding"},
		{Name: "frontend", Description: "Builds the React web application", Type: "coding"},
		{Name: "mobile", Description: "Builds mobile clients", Type: "coding"},
		{Name: "infra", Description: "Manages infrastructure", Type: "coding"},
		{Name: "data", Description: "Builds data pipelines", Type: "coding"},
		{Name: "auth", Description: "Owns authentication", Type: "coding"},
		{Name: "docs-agent", Description: "Maintains documentation", Type: "coding"},
		{Name: "design", Description: "Owns design system", Type: "coding"},
		{Name: "qa", Description: "Owns quality assurance", Type: "coding"},
		{Name: "security", Description: "Owns security review", Type: "coding"},
	}

	cfg := scaffold.Config{
		Name:        "bench-workspace",
		Description: "Large workspace for performance benchmarking",
		CodeMode:    "inline",
		Agents:      agentCfgs,
	}
	if err := scaffold.Scaffold(root, cfg); err != nil {
		tb.Fatalf("scaffold: %v", err)
	}

	// Write spec files.
	specFiles := map[string]string{
		"specs/api-design.md": "## Overview\n\nREST API following OpenAPI 3.0.\n\n" +
			"## Authentication\n\nBearer token auth.\n\n" +
			"## Endpoints\n\nCRUD resources: users, projects, tasks.\n\n" +
			"## Pagination\n\nCursor-based pagination.\n\n",
		"specs/auth-spec.md": "## Overview\n\nJWT tokens with RS256 signing.\n\n" +
			"## OAuth\n\nGoogle, GitHub, and Okta OAuth 2.0.\n\n" +
			"## RBAC\n\nRole-based access control.\n\n",
		"specs/data-model.md": "## Overview\n\nPostgreSQL primary, Redis for cache.\n\n" +
			"## Entities\n\nUsers, Organizations, Projects, Tasks.\n\n",
		"specs/security-policy.md": "## Overview\n\nSecurity policy last reviewed 2026-01.\n\n" +
			"## Vulnerabilities\n\nCVEs triaged within 24h.\n\n",
		"specs/design-system.md": "## Overview\n\nDesign system based on Radix UI.\n\n" +
			"## Tokens\n\nColor, typography, spacing tokens in CSS variables.\n\n",
	}
	for path, content := range specFiles {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			tb.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			tb.Fatalf("write %s: %v", path, err)
		}
	}

	// Write decision files.
	for i := 1; i <= 5; i++ {
		path := filepath.Join(root, "specs", "decisions", fmt.Sprintf("000%d-decision.md", i))
		content := fmt.Sprintf("## Decision\n\nArchitecture decision #%d.\n\n", i)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			tb.Fatalf("write decision %d: %v", i, err)
		}
	}

	// Write context snapshots.
	for i := 1; i <= 3; i++ {
		dir := filepath.Join(root, "context", fmt.Sprintf("topic-%d", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			tb.Fatalf("mkdir context/topic-%d: %v", i, err)
		}
		content := fmt.Sprintf("## Summary\n\nContext snapshot for topic %d.\n\n", i)
		if err := os.WriteFile(filepath.Join(dir, "summary.md"), []byte(content), 0644); err != nil {
			tb.Fatalf("write context snapshot %d: %v", i, err)
		}
	}

	// Write note files.
	notesDir := filepath.Join(root, "notes")
	for i := 1; i <= 2; i++ {
		content := fmt.Sprintf("## Idea %d\n\nRaw thinking not yet crystallized.\n\n", i)
		if err := os.WriteFile(filepath.Join(notesDir, fmt.Sprintf("idea-%d.md", i)), []byte(content), 0644); err != nil {
			tb.Fatalf("write note %d: %v", i, err)
		}
	}

	return root
}

// addLargeWorkspaceContent adds the non-agent content (specs, decisions,
// context snapshots, notes) to a testws builder.
func addLargeWorkspaceContent(b *testws.Builder) {
	specPairs := []struct{ path, heading, content string }{
		{"specs/api-design.md", "Overview", "REST API following OpenAPI 3.0 conventions"},
		{"specs/api-design.md", "Authentication", "Bearer token auth via auth service"},
		{"specs/api-design.md", "Endpoints", "CRUD resources: users, projects, tasks"},
		{"specs/auth-spec.md", "Overview", "JWT tokens with RS256 signing"},
		{"specs/auth-spec.md", "OAuth", "Google, GitHub, and Okta OAuth 2.0"},
		{"specs/data-model.md", "Overview", "PostgreSQL primary, Redis for cache"},
		{"specs/data-model.md", "Entities", "Users, Organizations, Projects, Tasks"},
		{"specs/security-policy.md", "Overview", "Security policy last reviewed 2026-01"},
		{"specs/design-system.md", "Overview", "Design system based on Radix UI"},
	}
	specsByPath := map[string][]testws.SpecOption{}
	for _, s := range specPairs {
		specsByPath[s.path] = append(specsByPath[s.path], testws.Section(s.heading, s.content))
	}
	for path, opts := range specsByPath {
		b.WithSpec(path, opts...)
	}

	for i := 1; i <= 5; i++ {
		b.WithDecision(
			fmt.Sprintf("000%d-decision", i),
			fmt.Sprintf("Architecture decision #%d recorded for traceability", i),
		)
	}

	for i := 1; i <= 3; i++ {
		b.WithContextSnapshot(
			fmt.Sprintf("topic-%d", i),
			"summary.md",
			fmt.Sprintf("## Summary\n\nContext snapshot for topic %d.", i),
		)
	}

	for i := 1; i <= 2; i++ {
		b.WithNote(
			fmt.Sprintf("idea-%d.md", i),
			fmt.Sprintf("## Idea %d\n\nRaw thinking — not yet crystallized.", i),
		)
	}
}
