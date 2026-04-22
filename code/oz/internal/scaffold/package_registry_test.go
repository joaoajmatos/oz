package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestValidPackageIDs_IncludeCorePackages(t *testing.T) {
	ids := ValidPackageIDs()
	if len(ids) == 0 || !slices.Contains(ids, "pm") || !slices.Contains(ids, "maintainer") {
		t.Fatalf("expected valid package IDs to include pm and maintainer, got %v", ids)
	}
}

func TestInstallPackage_UnknownID(t *testing.T) {
	dir := t.TempDir()
	_, err := InstallPackage("not-a-package", dir, false)
	var unk *ErrUnknownPackage
	if !errors.As(err, &unk) {
		t.Fatalf("expected *ErrUnknownPackage, got %T: %v", err, err)
	}
	if unk.ID != "not-a-package" {
		t.Errorf("ID = %q", unk.ID)
	}
	if !strings.Contains(err.Error(), "pm") || !strings.Contains(err.Error(), "maintainer") {
		t.Errorf("error should mention valid ids: %v", err)
	}
}

func TestInstallPackage_PMHappyPath(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Name:        "acme",
		Description: "demo",
		CodeMode:    "inline",
		Agents: []AgentConfig{
			{Name: "coding", Type: "coding"},
		},
	}
	if err := Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}
	paths, err := InstallPackage("pm", dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 3 {
		t.Fatalf("expected several paths, got %v", paths)
	}
	agent := filepath.Join(dir, "agents", "pm", "AGENT.md")
	if _, err := os.Stat(agent); err != nil {
		t.Fatal(err)
	}
}

func TestInstallPackage_PMDuplicateFails(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Name: "x", Description: "y", CodeMode: "inline", Agents: []AgentConfig{{Name: "a", Type: "coding"}}}
	if err := Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallPackage("pm", dir, false); err != nil {
		t.Fatal(err)
	}
	_, err := InstallPackage("pm", dir, false)
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected overwrite refusal, got %v", err)
	}
}

func TestInstallPackage_MaintainerAddsAgentAndManifestRows(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Name:        "acme",
		Description: "demo",
		CodeMode:    "inline",
		Agents: []AgentConfig{
			{Name: "coding", Type: "coding"},
		},
	}
	if err := Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "agents", "maintainer", "AGENT.md")); err == nil {
		t.Fatal("maintainer should not exist before package install")
	}

	paths, err := InstallPackage("maintainer", dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("expected package install to report written files")
	}

	if _, err := os.Stat(filepath.Join(dir, "agents", "maintainer", "AGENT.md")); err != nil {
		t.Fatalf("maintainer AGENT.md missing: %v", err)
	}

	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "| **maintainer** |") {
		t.Fatal("AGENTS.md should register maintainer")
	}

	ozManifest, err := os.ReadFile(filepath.Join(dir, "OZ.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ozManifest), "| **maintainer** |") {
		t.Fatal("OZ.md should register maintainer")
	}
}
