package context

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/joaoajmatos/oz/internal/graph"
)

// Serialize writes g to context/graph.json inside workspaceRoot.
// Nodes and edges are sorted by stable keys before writing.
// The ContentHash field is set to the SHA-256 of the canonical
// nodes+edges payload (excluding ContentHash itself).
//
// Running Serialize twice with identical input produces byte-identical output.
func Serialize(workspaceRoot string, g *graph.Graph) error {
	Normalise(g) // sort + hash in place

	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal graph: %w", err)
	}

	outPath := filepath.Join(workspaceRoot, "context", "graph.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("mkdir context/: %w", err)
	}

	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("write graph.json: %w", err)
	}

	return nil
}

// Normalise sorts nodes and edges into a stable order and sets ContentHash.
// It modifies g in place. Call this before any serialization or comparison.
func Normalise(g *graph.Graph) {
	sortNodes(g.Nodes)
	sortEdges(g.Edges)
	g.ContentHash = computeHash(g.Nodes, g.Edges)
}

// sortNodes sorts nodes by ID (lexicographic).
func sortNodes(nodes []graph.Node) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
}

// sortEdges sorts edges by (From, To, Type).
func sortEdges(edges []graph.Edge) {
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		return a.Type < b.Type
	})
}

// computeHash returns the SHA-256 of the canonical JSON encoding of nodes+edges.
// The hash is computed before ContentHash is set, so it captures only content.
func computeHash(nodes []graph.Node, edges []graph.Edge) string {
	payload := struct {
		Nodes []graph.Node `json:"nodes"`
		Edges []graph.Edge `json:"edges"`
	}{nodes, edges}

	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// LoadGraph reads and parses context/graph.json from workspaceRoot.
func LoadGraph(workspaceRoot string) (*graph.Graph, error) {
	path := filepath.Join(workspaceRoot, "context", "graph.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read graph.json: %w", err)
	}
	var g graph.Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parse graph.json: %w", err)
	}
	return &g, nil
}
