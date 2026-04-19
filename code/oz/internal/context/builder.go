package context

import (
	"fmt"
	"os"

	"github.com/oz-tools/oz/internal/graph"
)

// BuildResult is the output of a Build call.
type BuildResult struct {
	Graph     *graph.Graph
	NodeCount int
	EdgeCount int
}

// Build constructs the structural graph for the oz workspace at root.
// walker → parsers → indexer → extractor → graph
//
// The graph is returned but NOT written to disk; call Serialize to persist it.
func Build(root string) (*BuildResult, error) {
	// 1. Discover files.
	files, err := Walk(root)
	if err != nil {
		return nil, fmt.Errorf("walk workspace: %w", err)
	}

	var nodes []graph.Node

	// 2. Parse agent AGENT.md files → agent nodes.
	for _, f := range files {
		if f.Kind != KindAgentMD {
			continue
		}
		agent, err := ParseAgentMD(f.AbsPath, f.Agent)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", f.Path, err)
		}
		nodes = append(nodes, AgentNode(agent, f.Path))
	}

	// 3. Index spec / decision / doc / context / note files → section/file nodes.
	for _, f := range files {
		if f.Kind == KindAgentMD {
			continue
		}
		fileNodes, err := IndexMarkdownFile(f)
		if err != nil {
			return nil, fmt.Errorf("index %s: %w", f.Path, err)
		}
		nodes = append(nodes, fileNodes...)
	}

	// 4. Extract edges.
	edges := extractEdges(nodes, files)

	g := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Nodes:         nodes,
		Edges:         edges,
	}

	return &BuildResult{
		Graph:     g,
		NodeCount: len(nodes),
		EdgeCount: len(edges),
	}, nil
}

// readFile is a thin wrapper around os.ReadFile used by extractor.go.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
