package codeindex_test

import (
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/codeindex"
	"github.com/joaoajmatos/oz/internal/codeindex/goindexer"
)

func TestWalkCode_IncludeTestGo(t *testing.T) {
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "code", "pkg"))
	writeFile(t, filepath.Join(root, "code", "pkg", "main.go"), "package pkg\n")
	writeFile(t, filepath.Join(root, "code", "pkg", "main_test.go"), "package pkg\n")

	files, err := codeindex.WalkCode(root, []codeindex.Indexer{goindexer.New()}, codeindex.WalkOpts{IncludeTestGo: true})
	if err != nil {
		t.Fatalf("WalkCode: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2: %#v", len(files), files)
	}
	want0, want1 := "code/pkg/main.go", "code/pkg/main_test.go"
	if files[0].Path != want0 || files[1].Path != want1 {
		t.Errorf("paths = %q, %q, want %q, %q", files[0].Path, files[1].Path, want0, want1)
	}
}
