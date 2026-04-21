package context

import (
	"regexp"
	"strings"

	"github.com/joaoajmatos/oz/internal/graph"
)

// extractEdges produces edges by analysing the relationships between nodes.
//
// Edge types produced:
//   - reads:       agent → node for each file in the agent's read-chain
//   - owns:        agent → scope-path string (target may not be a graph node)
//   - references:  any document → node for each backtick/link file-path reference
//   - supports:    doc → spec/decision node when a doc references a spec file
func extractEdges(nodes []graph.Node, files []DiscoveredFile) []graph.Edge {
	// Build a lookup: file path → node ID (use the first node for that file).
	fileToNode := buildFileNodeMap(nodes)

	var edges []graph.Edge
	seen := map[string]bool{}

	add := func(from, to, edgeType string) {
		key := edgeType + ":" + from + "→" + to
		if seen[key] {
			return
		}
		seen[key] = true
		edges = append(edges, graph.Edge{From: from, To: to, Type: edgeType})
	}

	for _, n := range nodes {
		if n.Type != graph.NodeTypeAgent {
			continue
		}
		agentID := n.ID

		// reads edges — from read-chain file paths.
		for _, rcFile := range n.ReadChain {
			targetID := resolveFileRef(rcFile, fileToNode)
			if targetID != "" {
				add(agentID, targetID, graph.EdgeTypeReads)
			}
		}

		// owns edges — from scope paths.
		// We record owns edges even when the path is a glob pattern.
		for _, scopePath := range n.Scope {
			// Use a synthetic node ID for scope paths that aren't in the graph.
			// If the path resolves to a real node, use that; otherwise use
			// a "path:<scope>" pseudo-ID so the edge is still recorded.
			targetID := resolveFileRef(scopePath, fileToNode)
			if targetID == "" {
				targetID = "path:" + scopePath
			}
			add(agentID, targetID, graph.EdgeTypeOwns)
		}
	}

	// references and supports edges — scan all markdown files for file-path mentions.
	for _, f := range files {
		if f.Kind == KindAgentMD {
			// Agent reads/owns edges already handled above.
			// Still scan for inline references to other docs.
			nodeID := "agent:" + f.Agent
			extractRefEdges(f.AbsPath, nodeID, fileToNode, seen, &edges)
			continue
		}

		// Find the primary node ID for this file.
		fromID := fileToNode[f.Path]
		if fromID == "" {
			continue
		}

		extractRefEdges(f.AbsPath, fromID, fileToNode, seen, &edges)
	}

	return edges
}

// extractRefEdges scans a file for file-path references and appends edges.
func extractRefEdges(absPath, fromID string, fileToNode map[string]string, seen map[string]bool, edges *[]graph.Edge) {
	content, err := readFileContent(absPath)
	if err != nil {
		return
	}

	add := func(to, edgeType string) {
		key := edgeType + ":" + fromID + "→" + to
		if seen[key] {
			return
		}
		seen[key] = true
		*edges = append(*edges, graph.Edge{From: fromID, To: to, Type: edgeType})
	}

	for _, ref := range extractFileRefs(content) {
		toID := resolveFileRef(ref, fileToNode)
		if toID == "" || toID == fromID {
			continue
		}
		// Determine edge type.
		edgeType := graph.EdgeTypeReferences
		if isSpecRef(toID) && isDocNode(fromID) {
			edgeType = graph.EdgeTypeSupports
		}
		add(toID, edgeType)
	}
}

// buildFileNodeMap returns a map from file path → node ID.
// When multiple nodes share a file (e.g., multiple sections), the file maps
// to the first node's ID (lexicographically by node ID).
func buildFileNodeMap(nodes []graph.Node) map[string]string {
	m := make(map[string]string, len(nodes))
	for _, n := range nodes {
		if n.File == "" {
			continue
		}
		if existing, ok := m[n.File]; !ok || n.ID < existing {
			m[n.File] = n.ID
		}
	}
	return m
}

// resolveFileRef resolves a file reference string (which may include trailing
// annotations or be a backtick-wrapped path) to a node ID.
func resolveFileRef(ref string, fileToNode map[string]string) string {
	// Strip backticks.
	ref = strings.Trim(ref, "`")
	// Strip trailing annotation " — ...".
	if idx := strings.Index(ref, " — "); idx >= 0 {
		ref = ref[:idx]
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	// Direct match.
	if id, ok := fileToNode[ref]; ok {
		return id
	}
	// Try with forward-slash normalisation.
	normalised := strings.ReplaceAll(ref, "\\", "/")
	if id, ok := fileToNode[normalised]; ok {
		return id
	}
	return ""
}

// markdownLinkRe matches [text](path) where path looks like a file path.
var markdownLinkRe = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)

// backtickFileRe matches backtick-wrapped tokens that look like file paths
// (contain at least one / or a . followed by letters, excluding pure code).
var backtickFileRe = regexp.MustCompile("`([^`]*(?:/|\\.[a-zA-Z]{1,6})[^`]*)`")

// extractFileRefs returns all file-path-like strings found in content.
func extractFileRefs(content string) []string {
	seen := map[string]bool{}
	var refs []string

	addRef := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		// Skip URLs.
		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
			return
		}
		seen[s] = true
		refs = append(refs, s)
	}

	for _, m := range markdownLinkRe.FindAllStringSubmatch(content, -1) {
		addRef(m[2])
	}
	for _, m := range backtickFileRe.FindAllStringSubmatch(content, -1) {
		addRef(m[1])
	}

	return refs
}

// isSpecRef returns true if the node ID belongs to the specs tier.
func isSpecRef(id string) bool {
	return strings.HasPrefix(id, "spec_section:") || strings.HasPrefix(id, "decision:")
}

// isDocNode returns true if the from ID belongs to the docs tier.
func isDocNode(id string) bool {
	return strings.HasPrefix(id, "doc:")
}

// readFileContent reads the entire file contents as a string.
// Returns empty string on error (tolerant).
func readFileContent(path string) (string, error) {
	data, err := readFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
