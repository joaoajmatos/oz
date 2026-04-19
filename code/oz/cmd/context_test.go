package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindWorkspaceRoot_FromNestedDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# AGENTS\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "OZ.md"), []byte("oz standard: v0.1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "code", "oz")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(origWD); chdirErr != nil {
			t.Fatalf("restore working directory: %v", chdirErr)
		}
	}()

	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}

	got, err := findWorkspaceRoot()
	if err != nil {
		t.Fatalf("findWorkspaceRoot returned error: %v", err)
	}
	wantResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatal(err)
	}
	if gotResolved != wantResolved {
		t.Fatalf("expected workspace root %q, got %q", wantResolved, gotResolved)
	}
}
