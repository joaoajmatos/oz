package coverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/audit"
	"github.com/joaoajmatos/oz/internal/graph"
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

// countCode returns how many findings have the given code.
func countCode(fs []audit.Finding, code string) int {
	n := 0
	for _, f := range fs {
		if f.Code == code {
			n++
		}
	}
	return n
}

// run is a test helper that calls runCheck and panics on error.
func run(t *testing.T, root string, g *graph.Graph) []audit.Finding {
	t.Helper()
	fs, err := runCheck(root, g, nil)
	if err != nil {
		t.Fatalf("runCheck: %v", err)
	}
	return fs
}

// --- COV001: dangling scope paths ---

func TestCOV001_DanglingPath(t *testing.T) {
	root := t.TempDir()
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, File: "agents/coding/AGENT.md"},
		},
		Edges: []graph.Edge{
			{From: "agent:coding", To: "path:code/nonexistent", Type: graph.EdgeTypeOwns},
		},
	}
	fs := run(t, root, g)
	if !hasCode(fs, "COV001") {
		t.Error("expected COV001 for dangling non-glob scope path")
	}
}

func TestCOV001_ExistingPath_NoFinding(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "code", "oz"), 0755); err != nil {
		t.Fatal(err)
	}
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, File: "agents/coding/AGENT.md"},
		},
		Edges: []graph.Edge{
			{From: "agent:coding", To: "path:code/oz", Type: graph.EdgeTypeOwns},
		},
	}
	fs := run(t, root, g)
	if hasCode(fs, "COV001") {
		t.Error("expected no COV001 for existing path")
	}
}

func TestCOV001_GlobPath_Skipped(t *testing.T) {
	// Glob patterns should never generate COV001.
	root := t.TempDir()
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, File: "agents/coding/AGENT.md"},
		},
		Edges: []graph.Edge{
			{From: "agent:coding", To: "path:code/oz/**", Type: graph.EdgeTypeOwns},
		},
	}
	fs := run(t, root, g)
	if hasCode(fs, "COV001") {
		t.Error("expected no COV001 for glob scope path")
	}
}

func TestCOV001_NonOwnsEdge_Skipped(t *testing.T) {
	root := t.TempDir()
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent},
		},
		Edges: []graph.Edge{
			// reads edge, not owns — should not trigger COV001.
			{From: "agent:coding", To: "path:code/nonexistent", Type: graph.EdgeTypeReads},
		},
	}
	fs := run(t, root, g)
	if hasCode(fs, "COV001") {
		t.Error("expected no COV001 for non-owns edge")
	}
}

// --- COV002: unowned code/ directories ---

func TestCOV002_UnownedCodeDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "code", "oz"), 0755); err != nil {
		t.Fatal(err)
	}
	// Agent exists but owns a different path.
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, Scope: []string{"code/other/**"}},
		},
	}
	fs := run(t, root, g)
	if !hasCode(fs, "COV002") {
		t.Error("expected COV002 for unowned code/ directory")
	}
}

func TestCOV002_OwnedByExact_NoFinding(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "code", "oz"), 0755); err != nil {
		t.Fatal(err)
	}
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, Scope: []string{"code/oz/**"}},
		},
	}
	fs := run(t, root, g)
	if hasCode(fs, "COV002") {
		t.Error("expected no COV002 when agent owns the directory exactly")
	}
}

func TestCOV002_OwnedByWildcard_NoFinding(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "code", "oz"), 0755); err != nil {
		t.Fatal(err)
	}
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, Scope: []string{"code/**"}},
		},
	}
	fs := run(t, root, g)
	if hasCode(fs, "COV002") {
		t.Error("expected no COV002 when agent owns all of code/ via wildcard")
	}
}

func TestCOV002_NoCodeDir_NoFinding(t *testing.T) {
	root := t.TempDir()
	// No code/ directory at all — check should pass silently.
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent},
		},
	}
	fs := run(t, root, g)
	if hasCode(fs, "COV002") {
		t.Error("expected no COV002 when there is no code/ directory")
	}
}

func TestCOV002_MultipleUnownedDirs(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"alpha", "beta"} {
		if err := os.MkdirAll(filepath.Join(root, "code", d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, Scope: []string{"code/other/**"}},
		},
	}
	fs := run(t, root, g)
	if n := countCode(fs, "COV002"); n != 2 {
		t.Errorf("expected 2 COV002 findings, got %d", n)
	}
}

// --- COV003: overlapping non-glob scope paths ---

func TestCOV003_OverlappingPaths(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:backend", Type: graph.NodeTypeAgent, Scope: []string{"code/api"}},
			{ID: "agent:frontend", Type: graph.NodeTypeAgent, Scope: []string{"code/api/handlers"}},
		},
	}
	fs, _ := runCheck("", g, nil)
	if !hasCode(fs, "COV003") {
		t.Error("expected COV003 for overlapping scope paths")
	}
}

func TestCOV003_IdenticalPaths(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:backend", Type: graph.NodeTypeAgent, Scope: []string{"code/api"}},
			{ID: "agent:frontend", Type: graph.NodeTypeAgent, Scope: []string{"code/api"}},
		},
	}
	fs, _ := runCheck("", g, nil)
	if !hasCode(fs, "COV003") {
		t.Error("expected COV003 for identical scope paths")
	}
}

func TestCOV003_DisjointPaths_NoFinding(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:backend", Type: graph.NodeTypeAgent, Scope: []string{"code/api"}},
			{ID: "agent:frontend", Type: graph.NodeTypeAgent, Scope: []string{"code/ui"}},
		},
	}
	fs, _ := runCheck("", g, nil)
	if hasCode(fs, "COV003") {
		t.Error("expected no COV003 for disjoint paths")
	}
}

func TestCOV003_GlobPathsSkipped(t *testing.T) {
	// Glob paths should not participate in overlap detection.
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:backend", Type: graph.NodeTypeAgent, Scope: []string{"code/api/**"}},
			{ID: "agent:frontend", Type: graph.NodeTypeAgent, Scope: []string{"code/api/handlers"}},
		},
	}
	fs, _ := runCheck("", g, nil)
	if hasCode(fs, "COV003") {
		t.Error("expected no COV003 when glob paths are involved")
	}
}

func TestCOV003_SameAgent_NoFinding(t *testing.T) {
	// An agent with two overlapping scope paths should not get COV003.
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:backend", Type: graph.NodeTypeAgent, Scope: []string{"code/api", "code/api/handlers"}},
		},
	}
	fs, _ := runCheck("", g, nil)
	if hasCode(fs, "COV003") {
		t.Error("expected no COV003 for overlapping paths within the same agent")
	}
}

// --- COV004: scope without responsibilities ---

func TestCOV004_ScopeWithEmptyResponsibilities(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, Scope: []string{"code/oz/**"}, Responsibilities: ""},
		},
	}
	fs, _ := runCheck("", g, nil)
	if !hasCode(fs, "COV004") {
		t.Error("expected COV004 for agent with scope but empty responsibilities")
	}
}

func TestCOV004_ScopeWithResponsibilities_NoFinding(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, Scope: []string{"code/oz/**"}, Responsibilities: "Builds the oz toolset."},
		},
	}
	fs, _ := runCheck("", g, nil)
	if hasCode(fs, "COV004") {
		t.Error("expected no COV004 when responsibilities are set")
	}
}

func TestCOV004_NoScope_NoFinding(t *testing.T) {
	// An agent with no scope paths should not get COV004.
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, Scope: nil, Responsibilities: ""},
		},
	}
	fs, _ := runCheck("", g, nil)
	if hasCode(fs, "COV004") {
		t.Error("expected no COV004 when agent has no scope")
	}
}

// --- ownsCodeDir helper ---

func TestOwnsCodeDir(t *testing.T) {
	tests := []struct {
		scope string
		dir   string
		want  bool
	}{
		{"code/oz/**", "oz", true},
		{"code/oz", "oz", true},
		{"code/**", "oz", true},
		{"code/*", "oz", true},
		{"code/other/**", "oz", false},
		{"specs/**", "oz", false},
		{"code/oz/cmd/**", "oz", true}, // starts with code/oz
	}
	for _, tc := range tests {
		got := ownsCodeDir(tc.scope, tc.dir)
		if got != tc.want {
			t.Errorf("ownsCodeDir(%q, %q) = %v, want %v", tc.scope, tc.dir, got, tc.want)
		}
	}
}

// --- pathsOverlap helper ---

func TestPathsOverlap(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"code/api", "code/api/handlers", true},
		{"code/api/handlers", "code/api", true},
		{"code/api", "code/api", true},
		{"code/api", "code/ui", false},
		{"code/api", "code/apis", false}, // prefix match requires /
	}
	for _, tc := range tests {
		got := pathsOverlap(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("pathsOverlap(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
