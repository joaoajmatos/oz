package goindexer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oz-tools/oz/internal/codeindex"
	"github.com/oz-tools/oz/internal/codeindex/goindexer"
	"github.com/oz-tools/oz/internal/graph"
)

func TestIndexFile_ExtractsExportedSymbols(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n\ngo 1.21\n")
	writeFile(t, filepath.Join(root, "pkg", "sample.go"), `package pkg

const ExportedConst = 1
var ExportedVar = 2
var internalVar = 3

type ExportedType struct{}
type internalType struct{}

func ExportedFunc() {}
func internalFunc() {}
`)

	idx := goindexer.New()
	res, err := idx.IndexFile(codeindex.DiscoveredCodeFile{
		Path:    "code/project/pkg/sample.go",
		AbsPath: filepath.Join(root, "pkg", "sample.go"),
		Lang:    "go",
	})
	if err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	if res.FileNode.Type != graph.NodeTypeCodeFile {
		t.Fatalf("file node type = %q, want %q", res.FileNode.Type, graph.NodeTypeCodeFile)
	}
	if res.FileNode.Package != "example.com/project/pkg" {
		t.Fatalf("file node package = %q, want %q", res.FileNode.Package, "example.com/project/pkg")
	}

	if len(res.Symbols) != 4 {
		t.Fatalf("symbol count = %d, want 4", len(res.Symbols))
	}
	if len(res.Edges) != 4 {
		t.Fatalf("contains edge count = %d, want 4", len(res.Edges))
	}

	want := map[string]string{
		"ExportedConst": "value",
		"ExportedVar":   "value",
		"ExportedType":  "type",
		"ExportedFunc":  "func",
	}
	for _, sym := range res.Symbols {
		kind, ok := want[sym.Name]
		if !ok {
			t.Fatalf("unexpected symbol %q", sym.Name)
		}
		if sym.SymbolKind != kind {
			t.Fatalf("symbol %s kind = %q, want %q", sym.Name, sym.SymbolKind, kind)
		}
		if sym.Type != graph.NodeTypeCodeSymbol {
			t.Fatalf("symbol %s type = %q, want %q", sym.Name, sym.Type, graph.NodeTypeCodeSymbol)
		}
		if sym.Package != "example.com/project/pkg" {
			t.Fatalf("symbol %s package = %q", sym.Name, sym.Package)
		}
		if sym.Line <= 0 {
			t.Fatalf("symbol %s line = %d, want > 0", sym.Name, sym.Line)
		}
	}
}

func TestIndexFile_ParseFailureReturnsFileNodeOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n\ngo 1.21\n")
	writeFile(t, filepath.Join(root, "pkg", "broken.go"), "package pkg\nfunc Broken( {\n")

	idx := goindexer.New()
	res, err := idx.IndexFile(codeindex.DiscoveredCodeFile{
		Path:    "code/project/pkg/broken.go",
		AbsPath: filepath.Join(root, "pkg", "broken.go"),
		Lang:    "go",
	})
	if err != nil {
		t.Fatalf("IndexFile returned error on parse failure: %v", err)
	}
	if res.FileNode.Type != graph.NodeTypeCodeFile {
		t.Fatalf("file node type = %q, want %q", res.FileNode.Type, graph.NodeTypeCodeFile)
	}
	if len(res.Symbols) != 0 {
		t.Fatalf("symbols = %d, want 0 on parse failure", len(res.Symbols))
	}
	if len(res.Edges) != 0 {
		t.Fatalf("edges = %d, want 0 on parse failure", len(res.Edges))
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
