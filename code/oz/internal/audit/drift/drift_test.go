package drift_test

import (
	"testing"

	"github.com/oz-tools/oz/internal/audit/drift"
	"github.com/oz-tools/oz/internal/graph"
)

func TestLoadSymbols_Empty(t *testing.T) {
	g := &graph.Graph{}
	syms := drift.LoadSymbols(g)
	if len(syms) != 0 {
		t.Fatalf("expected 0 symbols from empty graph, got %d", len(syms))
	}
}

func TestLoadSymbols_FiltersNonSymbolNodes(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coder", Type: graph.NodeTypeAgent, Name: "coder"},
			{ID: "spec_section:specs/api.md:Overview", Type: graph.NodeTypeSpecSection, Name: "Overview"},
			{
				ID:         "code_symbol:example.com/pkg.Foo",
				Type:       graph.NodeTypeCodeSymbol,
				Name:       "Foo",
				SymbolKind: "func",
				Package:    "example.com/pkg",
				File:       "code/pkg/foo.go",
				Line:       10,
			},
		},
	}

	syms := drift.LoadSymbols(g)
	if len(syms) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(syms))
	}
	s := syms[0]
	if s.Name != "Foo" {
		t.Errorf("name: got %q, want %q", s.Name, "Foo")
	}
	if s.Pkg != "example.com/pkg" {
		t.Errorf("pkg: got %q, want %q", s.Pkg, "example.com/pkg")
	}
	if s.Kind != "func" {
		t.Errorf("kind: got %q, want %q", s.Kind, "func")
	}
	if s.File != "code/pkg/foo.go" {
		t.Errorf("file: got %q, want %q", s.File, "code/pkg/foo.go")
	}
	if s.Line != 10 {
		t.Errorf("line: got %d, want %d", s.Line, 10)
	}
}

func TestLoadSymbols_SortedByPkgThenName(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "code_symbol:b/pkg.Zebra", Type: graph.NodeTypeCodeSymbol, Name: "Zebra", Package: "b/pkg", SymbolKind: "type"},
			{ID: "code_symbol:a/pkg.Moo", Type: graph.NodeTypeCodeSymbol, Name: "Moo", Package: "a/pkg", SymbolKind: "func"},
			{ID: "code_symbol:a/pkg.Alpha", Type: graph.NodeTypeCodeSymbol, Name: "Alpha", Package: "a/pkg", SymbolKind: "value"},
			{ID: "code_symbol:b/pkg.Apple", Type: graph.NodeTypeCodeSymbol, Name: "Apple", Package: "b/pkg", SymbolKind: "func"},
		},
	}

	syms := drift.LoadSymbols(g)
	want := []struct{ pkg, name string }{
		{"a/pkg", "Alpha"},
		{"a/pkg", "Moo"},
		{"b/pkg", "Apple"},
		{"b/pkg", "Zebra"},
	}
	if len(syms) != len(want) {
		t.Fatalf("symbol count = %d, want %d", len(syms), len(want))
	}
	for i, w := range want {
		if syms[i].Pkg != w.pkg || syms[i].Name != w.name {
			t.Errorf("[%d] got (%q, %q), want (%q, %q)", i, syms[i].Pkg, syms[i].Name, w.pkg, w.name)
		}
	}
}

func TestLoadSymbols_AllKinds(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "code_symbol:pkg.F", Type: graph.NodeTypeCodeSymbol, Name: "F", SymbolKind: "func", Package: "pkg"},
			{ID: "code_symbol:pkg.T", Type: graph.NodeTypeCodeSymbol, Name: "T", SymbolKind: "type", Package: "pkg"},
			{ID: "code_symbol:pkg.V", Type: graph.NodeTypeCodeSymbol, Name: "V", SymbolKind: "value", Package: "pkg"},
		},
	}

	syms := drift.LoadSymbols(g)
	if len(syms) != 3 {
		t.Fatalf("expected 3 symbols, got %d", len(syms))
	}
	kinds := map[string]bool{}
	for _, s := range syms {
		kinds[s.Kind] = true
	}
	for _, k := range []string{"func", "type", "value"} {
		if !kinds[k] {
			t.Errorf("kind %q missing from results", k)
		}
	}
}
