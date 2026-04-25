package context

import (
	"fmt"
	"os"
	"slices"

	"github.com/joaoajmatos/oz/internal/codeindex"
	_ "github.com/joaoajmatos/oz/internal/codeindex/goindexer" // registers Go language package
	"github.com/joaoajmatos/oz/internal/graph"
)

// BuildResult is the output of a Build call.
type BuildResult struct {
	Graph     *graph.Graph
	NodeCount int
	EdgeCount int
	Concepts  []codeindex.CodeConcept // framework-specific concepts; may be empty
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
	activePkgs := codeindex.Detect(root)

	// Build a ProjectContext for each active language (one Detect call per package).
	type pkgEntry struct {
		pkg codeindex.LanguagePackage
		ctx codeindex.ProjectContext
	}
	langMap := make(map[string]pkgEntry, len(activePkgs))
	for _, p := range activePkgs {
		dr := p.Detect(root)
		langMap[p.Language()] = pkgEntry{
			pkg: p,
			ctx: codeindex.ProjectContext{Root: root, Framework: dr.Framework, Manifest: dr.Manifest},
		}
	}

	codeFiles, err := codeindex.WalkCode(root, activePkgs, codeindex.WalkOpts{})
	if err != nil {
		return nil, fmt.Errorf("walk code files: %w", err)
	}

	// seenPkg de-dupes code_package nodes by ID.
	// For DocComment: keep the first non-empty value seen across files in the package.
	type seenEntry struct {
		node   graph.Node
		hasDoc bool
	}
	seenPkg := map[string]seenEntry{}
	var allConcepts []codeindex.CodeConcept

	for _, cf := range codeFiles {
		entry, ok := langMap[cf.Lang]
		if !ok {
			continue
		}
		res, indexErr := entry.pkg.IndexFile(cf, entry.ctx)
		if indexErr != nil {
			return nil, fmt.Errorf("index code file %s: %w", cf.Path, indexErr)
		}
		nodes = append(nodes, res.FileNode)
		nodes = append(nodes, res.Symbols...)
		edges = append(edges, res.Edges...)

		if pn := res.PackageNode; pn != nil {
			if e, seen := seenPkg[pn.ID]; !seen {
				seenPkg[pn.ID] = seenEntry{*pn, pn.DocComment != ""}
			} else if !e.hasDoc && pn.DocComment != "" {
				updated := e.node
				updated.DocComment = pn.DocComment
				seenPkg[pn.ID] = seenEntry{updated, true}
			}
		}

		concepts, _ := entry.pkg.ExtractSemantics(cf, entry.ctx)
		allConcepts = append(allConcepts, concepts...)
	}

	// Emit code_package nodes in sorted order for determinism.
	pkgIDs := make([]string, 0, len(seenPkg))
	for id := range seenPkg {
		pkgIDs = append(pkgIDs, id)
	}
	slices.Sort(pkgIDs)
	for _, id := range pkgIDs {
		nodes = append(nodes, seenPkg[id].node)
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
		Concepts:  allConcepts,
	}, nil
}

// readFile is a thin wrapper around os.ReadFile used by extractor.go.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
