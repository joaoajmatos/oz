// Package drift provides primitives for spec-code drift detection.
//
// Sprint A4 re-scope: symbols are sourced from the structural graph produced
// by oz context build (code_symbol nodes), not by re-parsing source files.
// The standalone Go AST extractor was superseded by the CI-1 code-indexing sprint.
package drift

import (
	"sort"

	"github.com/oz-tools/oz/internal/graph"
)

// Symbol is an exported source symbol extracted from the structural graph.
type Symbol struct {
	// Pkg is the fully-qualified import path (e.g. github.com/oz-tools/oz/internal/audit).
	Pkg string

	// Name is the unqualified symbol name (e.g. RunAll).
	Name string

	// Kind is the symbol kind: "func", "type", or "value".
	Kind string

	// File is the workspace-relative source file path.
	File string

	// Line is the 1-based source line number.
	Line int
}

// LoadSymbols returns all exported symbols present in g as a sorted slice.
// Symbols are sorted by (Pkg, Name) for deterministic output.
func LoadSymbols(g *graph.Graph) []Symbol {
	var symbols []Symbol
	for _, n := range g.Nodes {
		if n.Type != graph.NodeTypeCodeSymbol {
			continue
		}
		symbols = append(symbols, symbolFromGraphNode(n))
	}
	sortSymbols(symbols)
	return symbols
}

func symbolFromGraphNode(n graph.Node) Symbol {
	return Symbol{
		Pkg:  n.Package,
		Name: n.Name,
		Kind: n.SymbolKind,
		File: n.File,
		Line: n.Line,
	}
}

func sortSymbols(symbols []Symbol) {
	sort.Slice(symbols, func(i, j int) bool {
		if symbols[i].Pkg != symbols[j].Pkg {
			return symbols[i].Pkg < symbols[j].Pkg
		}
		return symbols[i].Name < symbols[j].Name
	})
}
