package goindexer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/codeindex"
	"github.com/joaoajmatos/oz/internal/codeindex/goindexer"
	"github.com/joaoajmatos/oz/internal/graph"
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
	}, codeindex.ProjectContext{Root: root})
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
	}, codeindex.ProjectContext{Root: root})
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

func TestDetect_GoModFound(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "code", "myapp", "go.mod"), "module example.com/myapp\n\ngo 1.21\n")

	idx := goindexer.New()
	dr := idx.Detect(root)
	if dr.Confidence == 0 {
		t.Fatal("Detect: expected Confidence > 0 when go.mod exists")
	}
	if dr.Manifest == "" {
		t.Fatal("Detect: expected Manifest to be set")
	}
}

func TestDetect_NoGoMod(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "code", "myapp", "main.py"), "print('hello')\n")

	idx := goindexer.New()
	dr := idx.Detect(root)
	if dr.Confidence != 0 {
		t.Fatalf("Detect: expected Confidence == 0 when no go.mod, got %f", dr.Confidence)
	}
}

func TestIndexFile_PackageNodeSet(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n\ngo 1.21\n")
	writeFile(t, filepath.Join(root, "pkg", "sample.go"), "package pkg\n\nfunc Hello() {}\n")

	idx := goindexer.New()
	res, err := idx.IndexFile(codeindex.DiscoveredCodeFile{
		Path:    "code/project/pkg/sample.go",
		AbsPath: filepath.Join(root, "pkg", "sample.go"),
		Lang:    "go",
	}, codeindex.ProjectContext{Root: root})
	if err != nil {
		t.Fatalf("IndexFile: %v", err)
	}
	if res.PackageNode == nil {
		t.Fatal("expected PackageNode to be set")
	}
	if res.PackageNode.Type != "code_package" {
		t.Errorf("PackageNode.Type = %q, want code_package", res.PackageNode.Type)
	}
	if res.PackageNode.Language != "go" {
		t.Errorf("PackageNode.Language = %q, want go", res.PackageNode.Language)
	}
	wantID := "code_package:example.com/project/pkg"
	if res.PackageNode.ID != wantID {
		t.Errorf("PackageNode.ID = %q, want %q", res.PackageNode.ID, wantID)
	}
	wantName := "pkg"
	if res.PackageNode.Name != wantName {
		t.Errorf("PackageNode.Name = %q, want %q", res.PackageNode.Name, wantName)
	}
}

func TestExtractSemantics_Stub(t *testing.T) {
	root := t.TempDir()
	idx := goindexer.New()
	concepts, err := idx.ExtractSemantics(codeindex.DiscoveredCodeFile{
		Path: "code/project/pkg/main.go", Lang: "go",
	}, codeindex.ProjectContext{Root: root})
	if err != nil {
		t.Fatalf("ExtractSemantics: unexpected error: %v", err)
	}
	if len(concepts) != 0 {
		t.Fatalf("ExtractSemantics: expected nil/empty stub, got %d concepts", len(concepts))
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
