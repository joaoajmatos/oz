package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oz-tools/oz/internal/workspace"
)

func TestNew_ResolvesRelativePath(t *testing.T) {
	ws, err := workspace.New(".")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(ws.Root) {
		t.Errorf("expected absolute path, got %q", ws.Root)
	}
}

func TestValid(t *testing.T) {
	dir := t.TempDir()

	ws, _ := workspace.New(dir)
	if ws.Valid() {
		t.Fatal("expected invalid workspace with no files")
	}

	write(t, filepath.Join(dir, "AGENTS.md"), "")
	if ws.Valid() {
		t.Fatal("expected invalid without OZ.md")
	}

	write(t, filepath.Join(dir, "OZ.md"), "")
	if !ws.Valid() {
		t.Fatal("expected valid once both required files exist")
	}
}

func TestReadManifest(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "OZ.md"), "oz standard: v0.1\nproject: myproject\ndescription: a test project\n")

	ws, _ := workspace.New(dir)
	m, err := ws.ReadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "myproject" {
		t.Errorf("Name: got %q, want %q", m.Name, "myproject")
	}
	if m.Description != "a test project" {
		t.Errorf("Description: got %q, want %q", m.Description, "a test project")
	}
}

func TestReadManifest_MissingFields(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "OZ.md"), "# no fields here\n")

	ws, _ := workspace.New(dir)
	m, err := ws.ReadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "" || m.Description != "" {
		t.Errorf("expected zero values, got name=%q description=%q", m.Name, m.Description)
	}
}

func TestAgents(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, filepath.Join(dir, "agents", "coding"))
	mkdir(t, filepath.Join(dir, "agents", "maintainer"))
	// a file should be ignored
	write(t, filepath.Join(dir, "agents", "notadir.md"), "")

	ws, _ := workspace.New(dir)
	agents, err := ws.Agents()
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d: %v", len(agents), agents)
	}
}

func TestAgents_Empty(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, filepath.Join(dir, "agents"))

	ws, _ := workspace.New(dir)
	agents, err := ws.Agents()
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
}

func TestHierarchyLayers(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, filepath.Join(dir, "specs"))
	// docs intentionally missing

	ws, _ := workspace.New(dir)
	layers := ws.HierarchyLayers()

	byName := make(map[string]bool)
	for _, l := range layers {
		byName[l.Name] = l.Exists
	}

	if !byName["specs"] {
		t.Error("expected specs to exist")
	}
	if byName["docs"] {
		t.Error("expected docs to not exist")
	}
}

// helpers

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}
