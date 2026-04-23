package context

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/joaoajmatos/oz/internal/codeindex"
	"github.com/joaoajmatos/oz/internal/codeindex/goindexer"
	"github.com/joaoajmatos/oz/internal/graph"
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

	var edges []graph.Edge

	// 4. Index code files and symbols under code/.
	goIdx := goindexer.New()
	codeFiles, err := codeindex.WalkCode(root, []codeindex.Indexer{goIdx}, codeindex.WalkOpts{})
	if err != nil {
		return nil, fmt.Errorf("walk code files: %w", err)
	}
	// pkgDocs collects the first non-empty package doc comment seen per import path.
	pkgDocs := map[string]string{}
	for _, cf := range codeFiles {
		if cf.Lang != goIdx.Language() {
			continue
		}
		res, indexErr := goIdx.IndexFile(cf)
		if indexErr != nil {
			return nil, fmt.Errorf("index code file %s: %w", cf.Path, indexErr)
		}
		nodes = append(nodes, res.FileNode)
		nodes = append(nodes, res.Symbols...)
		edges = append(edges, res.Edges...)
		if res.FileNode.Package != "" {
			if _, seen := pkgDocs[res.FileNode.Package]; !seen && res.FileNode.DocComment != "" {
				pkgDocs[res.FileNode.Package] = res.FileNode.DocComment
			} else if _, seen := pkgDocs[res.FileNode.Package]; !seen {
				pkgDocs[res.FileNode.Package] = ""
			}
		}
	}
	// Emit one code_package node per unique Go package, sorted for determinism.
	pkgPaths := make([]string, 0, len(pkgDocs))
	for pkg := range pkgDocs {
		pkgPaths = append(pkgPaths, pkg)
	}
	slices.Sort(pkgPaths)
	for _, pkg := range pkgPaths {
		nodes = append(nodes, graph.Node{
			ID:         "code_package:" + pkg,
			Type:       graph.NodeTypeCodePackage,
			Name:       lastPathSegment(pkg),
			Package:    pkg,
			Language:   goIdx.Language(),
			DocComment: pkgDocs[pkg],
		})
	}

	// 5. Extract cross-reference and ownership edges now that code_file nodes exist.
	edges = append(edges, extractEdges(nodes, files)...)

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

// lastPathSegment returns the last slash-delimited segment of a path or import path.
func lastPathSegment(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
