package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// scaffoldValidWorkspace creates a minimal workspace that passes oz validate.
func scaffoldValidWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	for _, d := range []string{"agents", "specs/decisions", "docs", "context", "skills", "rules", "notes"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}

	writeFile(t, filepath.Join(dir, "AGENTS.md"), "# AGENTS.md\n")
	writeFile(t, filepath.Join(dir, "OZ.md"), "oz standard: v0.1\nproject: test\ndescription: test\n")
	writeFile(t, filepath.Join(dir, "README.md"), "# test\n")

	agentDir := filepath.Join(dir, "agents", "coding")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(agentDir, "AGENT.md"),
		"# coding Agent\n\n## Role\n\n## Read-chain\n\n## Responsibilities\n")

	return dir
}

func TestValidateCmd_ValidWorkspace(t *testing.T) {
	dir := scaffoldValidWorkspace(t)
	err := runValidate(validateCmd, []string{dir})
	if err != nil {
		t.Errorf("expected nil error for valid workspace, got: %v", err)
	}
}

func TestValidateCmd_InvalidWorkspace_ReturnsValidationError(t *testing.T) {
	dir := t.TempDir() // empty — not a valid workspace
	err := runValidate(validateCmd, []string{dir})
	if !errors.Is(err, errValidationFailed) {
		t.Errorf("expected errValidationFailed, got: %v", err)
	}
}

func TestValidateCmd_InvalidWorkspace_PrintsErrorsToStderr(t *testing.T) {
	dir := t.TempDir()

	stderr := captureStderr(t, func() {
		_ = runValidate(validateCmd, []string{dir})
	})

	if len(stderr) == 0 {
		t.Error("expected error output on stderr for invalid workspace")
	}
}

func TestValidateCmd_WarningOnly_ReturnsNil(t *testing.T) {
	dir := scaffoldValidWorkspace(t)
	os.Remove(filepath.Join(dir, "README.md")) // recommended, not required

	err := runValidate(validateCmd, []string{dir})
	if err != nil {
		t.Errorf("expected nil error for warning-only workspace, got: %v", err)
	}
}

func TestValidateCmd_NestedPath_UsesWorkspaceRoot(t *testing.T) {
	dir := scaffoldValidWorkspace(t)
	nested := filepath.Join(dir, "code", "oz")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	err := runValidate(validateCmd, []string{nested})
	if err != nil {
		t.Errorf("expected nil error for nested workspace path, got: %v", err)
	}
}

// captureStderr redirects os.Stderr for the duration of fn and returns what was written.
func captureStderr(t *testing.T, fn func()) []byte {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.Bytes()
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
