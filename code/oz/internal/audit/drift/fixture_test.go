package drift

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/audit/drift/specscan"
)

// testdataPath returns the absolute path to a named testdata fixture.
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	// Resolved relative to the test binary's working directory (the package dir).
	return filepath.Join("testdata", name)
}

// TestFixture_Clean verifies that a fully-consistent spec+code workspace
// produces zero findings.
func TestFixture_Clean(t *testing.T) {
	root := testdataPath(t, "clean")

	// Create the code/ path that the spec references.
	codePkg := filepath.Join(root, "code", "pkg")
	if err := os.MkdirAll(codePkg, 0755); err != nil {
		t.Fatal(err)
	}
	fooGo := filepath.Join(codePkg, "foo.go")
	if err := os.WriteFile(fooGo, []byte("package pkg\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(filepath.Join(root, "code"))
	})

	symbols := []Symbol{
		{Pkg: "example.com/pkg", Name: "Foo", Kind: "func", File: "code/pkg/foo.go", Line: 1},
	}

	candidates, err := specscan.Scan(root, specscan.Options{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	fs := runCheck(root, symbols, candidates)

	if len(fs) != 0 {
		t.Errorf("clean fixture: expected no findings, got %+v", fs)
	}
}

// TestFixture_DriftedPath verifies DRIFT001 when a spec references a deleted code/ path.
func TestFixture_DriftedPath(t *testing.T) {
	root := testdataPath(t, "drifted-path")

	candidates, err := specscan.Scan(root, specscan.Options{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	fs := runCheck(root, nil, candidates)

	if !hasCode(fs, "DRIFT001") {
		t.Errorf("drifted-path fixture: expected DRIFT001, got %+v", fs)
	}
}

// TestFixture_DriftedSymbol verifies DRIFT002 when a spec references a removed function.
func TestFixture_DriftedSymbol(t *testing.T) {
	root := testdataPath(t, "drifted-symbol")

	// No symbols in the graph (simulate function was removed).
	candidates, err := specscan.Scan(root, specscan.Options{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	fs := runCheck(root, nil, candidates)

	if !hasCode(fs, "DRIFT002") {
		t.Errorf("drifted-symbol fixture: expected DRIFT002, got %+v", fs)
	}
}

// TestFixture_UnmentionedSymbol verifies DRIFT003 when an exported symbol
// has no corresponding mention in any spec.
func TestFixture_UnmentionedSymbol(t *testing.T) {
	root := testdataPath(t, "unmentioned-symbol")

	symbols := []Symbol{
		{Pkg: "example.com/pkg", Name: "Bar", Kind: "func", File: "code/pkg/bar.go", Line: 1},
	}
	candidates, err := specscan.Scan(root, specscan.Options{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	fs := runCheck(root, symbols, candidates)

	if !hasCode(fs, "DRIFT003") {
		t.Errorf("unmentioned-symbol fixture: expected DRIFT003, got %+v", fs)
	}
}
