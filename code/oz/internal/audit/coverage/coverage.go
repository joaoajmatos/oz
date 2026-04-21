// Package coverage implements the oz audit coverage check.
// It reports dangling scope paths (COV001), unowned code directories (COV002),
// overlapping agent scope paths (COV003), and agents with scope but no
// responsibilities description (COV004).
package coverage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaoajmatos/oz/internal/audit"
	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/graph"
)

// Check implements audit.Check for coverage checks.
type Check struct{}

// Name returns the check name.
func (c *Check) Name() string { return "coverage" }

// Codes returns the finding codes this check may produce.
func (c *Check) Codes() []string { return []string{"COV001", "COV002", "COV003", "COV004"} }

// Run loads the graph from root and returns coverage findings.
func (c *Check) Run(root string, _ audit.Options) ([]audit.Finding, error) {
	g, err := ozcontext.LoadGraph(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("graph.json not found — run 'oz context build' first")
		}
		return nil, fmt.Errorf("coverage: load graph: %w", err)
	}
	return runCheck(root, g)
}

// runCheck runs all coverage sub-checks against a pre-loaded graph.
// Separated from Run for testability.
func runCheck(root string, g *graph.Graph) ([]audit.Finding, error) {
	agentNodes := buildAgentIndex(g)

	var findings []audit.Finding

	findings = append(findings, checkDanglingPaths(root, g, agentNodes)...)

	cov002, err := checkUnownedCodeDirs(root, g)
	if err != nil {
		return nil, err
	}
	findings = append(findings, cov002...)

	findings = append(findings, checkOverlappingScopes(g)...)
	findings = append(findings, checkScopeWithoutResponsibilities(g)...)

	return findings, nil
}

// buildAgentIndex returns a map from agent node ID to node.
func buildAgentIndex(g *graph.Graph) map[string]graph.Node {
	m := make(map[string]graph.Node)
	for _, n := range g.Nodes {
		if n.Type == graph.NodeTypeAgent {
			m[n.ID] = n
		}
	}
	return m
}

// checkDanglingPaths emits COV001 for each owns edge whose target is a
// path: pseudo-id that does not exist on disk.
// Glob paths (containing * or ?) are skipped because they cannot be stat'd.
func checkDanglingPaths(root string, g *graph.Graph, agentNodes map[string]graph.Node) []audit.Finding {
	var findings []audit.Finding
	seen := make(map[string]bool)

	for _, e := range g.Edges {
		if e.Type != graph.EdgeTypeOwns {
			continue
		}
		if !strings.HasPrefix(e.To, "path:") {
			continue
		}
		scopePath := strings.TrimPrefix(e.To, "path:")

		// Skip glob patterns — they cannot be stat'd meaningfully.
		if strings.ContainsAny(scopePath, "*?") {
			continue
		}

		key := e.From + "|" + scopePath
		if seen[key] {
			continue
		}
		seen[key] = true

		diskPath := filepath.Join(root, filepath.FromSlash(scopePath))
		if _, err := os.Stat(diskPath); os.IsNotExist(err) {
			agentFile := ""
			if a, ok := agentNodes[e.From]; ok {
				agentFile = a.File
			}
			findings = append(findings, audit.Finding{
				Check:    "coverage",
				Code:     "COV001",
				Severity: audit.SeverityError,
				Message:  fmt.Sprintf("%s owns %q which does not exist on disk", e.From, scopePath),
				File:     agentFile,
				Hint:     "remove the scope entry from the agent's AGENT.md or create the missing path",
				Refs:     []string{e.From, scopePath},
			})
		}
	}
	return findings
}

// checkUnownedCodeDirs emits COV002 for each top-level directory under code/
// that is not owned by any agent's scope declaration.
func checkUnownedCodeDirs(root string, g *graph.Graph) ([]audit.Finding, error) {
	codeDir := filepath.Join(root, "code")
	entries, err := os.ReadDir(codeDir)
	if os.IsNotExist(err) {
		return nil, nil // no code/ directory; nothing to check
	}
	if err != nil {
		return nil, fmt.Errorf("coverage: read code/: %w", err)
	}

	// Collect all scope paths across agents.
	var allScopes []string
	for _, n := range g.Nodes {
		if n.Type == graph.NodeTypeAgent {
			allScopes = append(allScopes, n.Scope...)
		}
	}

	var findings []audit.Finding
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := e.Name()
		if isOwnedByAny(dir, allScopes) {
			continue
		}
		findings = append(findings, audit.Finding{
			Check:    "coverage",
			Code:     "COV002",
			Severity: audit.SeverityWarn,
			Message:  fmt.Sprintf("code/%s/ is not owned by any agent", dir),
			File:     "code/" + dir,
			Hint:     fmt.Sprintf("add 'code/%s/**' to an agent's scope in AGENT.md", dir),
		})
	}
	return findings, nil
}

// isOwnedByAny reports whether any scope path declares ownership of code/<dir>.
func isOwnedByAny(dir string, scopePaths []string) bool {
	for _, sp := range scopePaths {
		if ownsCodeDir(sp, dir) {
			return true
		}
	}
	return false
}

// ownsCodeDir reports whether scopePath declares ownership of the named top-level
// directory under code/. It handles both exact and glob patterns.
//
// Examples:
//
//	ownsCodeDir("code/oz/**", "oz")    → true
//	ownsCodeDir("code/**", "oz")       → true
//	ownsCodeDir("code/*", "oz")        → true
//	ownsCodeDir("code/other/**", "oz") → false
//	ownsCodeDir("specs/**", "oz")      → false
func ownsCodeDir(scopePath, dir string) bool {
	parts := strings.SplitN(filepath.ToSlash(scopePath), "/", 3)
	if len(parts) < 2 || parts[0] != "code" {
		return false
	}
	second := parts[1]
	// "*" or "**" matches all directories under code/.
	return second == dir || second == "*" || second == "**"
}

// checkOverlappingScopes emits COV003 when two different agents declare
// non-glob scope paths where one is a prefix of the other.
func checkOverlappingScopes(g *graph.Graph) []audit.Finding {
	type agentPath struct {
		agentID string
		file    string
		path    string
	}

	var nonGlob []agentPath
	for _, n := range g.Nodes {
		if n.Type != graph.NodeTypeAgent {
			continue
		}
		for _, sp := range n.Scope {
			if !strings.ContainsAny(sp, "*?") {
				nonGlob = append(nonGlob, agentPath{n.ID, n.File, sp})
			}
		}
	}

	var findings []audit.Finding
	seen := make(map[string]bool)

	for i := 0; i < len(nonGlob); i++ {
		for j := i + 1; j < len(nonGlob); j++ {
			a, b := nonGlob[i], nonGlob[j]
			if a.agentID == b.agentID {
				continue
			}
			if !pathsOverlap(a.path, b.path) {
				continue
			}
			// Use a canonical key to deduplicate symmetric pairs.
			k1, k2 := a.agentID+"|"+a.path, b.agentID+"|"+b.path
			if k1 > k2 {
				k1, k2 = k2, k1
			}
			key := k1 + "||" + k2
			if seen[key] {
				continue
			}
			seen[key] = true
			findings = append(findings, audit.Finding{
				Check:    "coverage",
				Code:     "COV003",
				Severity: audit.SeverityWarn,
				Message:  fmt.Sprintf("%s and %s have overlapping scope paths %q and %q", a.agentID, b.agentID, a.path, b.path),
				Hint:     "narrow the scope paths so each path is owned by exactly one agent",
				Refs:     []string{a.agentID, b.agentID, a.path, b.path},
			})
		}
	}
	return findings
}

// pathsOverlap reports whether path a is a directory-prefix of path b or vice versa.
func pathsOverlap(a, b string) bool {
	a = strings.TrimRight(filepath.ToSlash(a), "/")
	b = strings.TrimRight(filepath.ToSlash(b), "/")
	return a == b ||
		strings.HasPrefix(b, a+"/") ||
		strings.HasPrefix(a, b+"/")
}

// checkScopeWithoutResponsibilities emits COV004 for each agent that has
// declared scope paths but has an empty Responsibilities description.
func checkScopeWithoutResponsibilities(g *graph.Graph) []audit.Finding {
	var findings []audit.Finding
	for _, n := range g.Nodes {
		if n.Type != graph.NodeTypeAgent {
			continue
		}
		if len(n.Scope) > 0 && strings.TrimSpace(n.Responsibilities) == "" {
			findings = append(findings, audit.Finding{
				Check:    "coverage",
				Code:     "COV004",
				Severity: audit.SeverityInfo,
				Message:  fmt.Sprintf("%s declares scope paths but has no responsibilities description", n.ID),
				File:     n.File,
				Hint:     "add a Responsibilities section to the agent's AGENT.md describing what it owns",
				Refs:     []string{n.ID},
			})
		}
	}
	return findings
}
