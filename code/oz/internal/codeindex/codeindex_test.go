package codeindex_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oz-tools/oz/internal/codeindex"
	"github.com/oz-tools/oz/internal/codeindex/goindexer"
)

func TestWalkCode_SkipsAndFilters(t *testing.T) {
	root := t.TempDir()

	mkdirAll(t, filepath.Join(root, "code", "pkg", "nested"))
	mkdirAll(t, filepath.Join(root, "code", "vendor", "dep"))
	mkdirAll(t, filepath.Join(root, "code", "testdata", "fixtures"))

	writeFile(t, filepath.Join(root, "code", "pkg", "main.go"), "package pkg\n")
	writeFile(t, filepath.Join(root, "code", "pkg", "main_test.go"), "package pkg\n")
	writeFile(t, filepath.Join(root, "code", "pkg", "nested", "util.go"), "package nested\n")
	writeFile(t, filepath.Join(root, "code", "pkg", "nested", "readme.md"), "# not code\n")
	writeFile(t, filepath.Join(root, "code", "vendor", "dep", "dep.go"), "package dep\n")
	writeFile(t, filepath.Join(root, "code", "testdata", "fixtures", "fixture.go"), "package fixture\n")

	files, err := codeindex.WalkCode(root, []codeindex.Indexer{goindexer.New()})
	if err != nil {
		t.Fatalf("WalkCode: %v", err)
	}

	got := make([]string, 0, len(files))
	for _, f := range files {
		got = append(got, f.Path)
		if f.Lang != "go" {
			t.Fatalf("file %s lang = %q, want go", f.Path, f.Lang)
		}
	}

	want := []string{
		"code/pkg/main.go",
		"code/pkg/nested/util.go",
	}
	if len(got) != len(want) {
		t.Fatalf("walk result count = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("walk result[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
