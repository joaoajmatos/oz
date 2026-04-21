package scaffold_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/scaffold"
)

func TestRepair_NothingMissing(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Repair(dir, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 0 {
		t.Fatalf("expected no created files, got %v", result.Created)
	}
	if len(result.Skipped) == 0 {
		t.Fatal("expected some skipped files")
	}
}

func TestRepair_MissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(dir, "docs", "open-items.md")
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Repair(dir, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if !containsPath(result.Created, "docs/open-items.md") {
		t.Fatalf("expected docs/open-items.md in Created, got %v", result.Created)
	}
}

func TestRepair_MissingSkillFile(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(dir, "skills", "workspace-management", "assets", "AGENT.md.tmpl")
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Repair(dir, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	want := "skills/workspace-management/assets/AGENT.md.tmpl"
	if !containsPath(result.Created, want) {
		t.Fatalf("expected %q in Created, got %v", want, result.Created)
	}
}

func TestRepair_NeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "docs", "architecture.md")
	sentinel := "SENTINEL_CONTENT_DO_NOT_REPLACE\n"
	if err := os.WriteFile(target, []byte(sentinel), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Repair(dir, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range result.Created {
		if p == "docs/architecture.md" {
			t.Fatal("repair must not recreate an existing docs/architecture.md")
		}
	}
	content := readFile(t, target)
	if content != sentinel {
		t.Fatalf("expected sentinel content unchanged, got %q", content)
	}
}

func containsPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}
