package testfixture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFixtureLoadReadsFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	name := "sample case"
	fixtureDir := filepath.Join(dir, FixtureName(name))
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "input.txt"), []byte("input"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "expected.txt"), []byte("expected"), 0o644); err != nil {
		t.Fatalf("write expected: %v", err)
	}

	f := New(dir)
	input, expected := f.Load(t, name)
	if input != "input" || expected != "expected" {
		t.Fatalf("unexpected load result: input=%q expected=%q", input, expected)
	}
}

func TestFixtureAssertMatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	name := "match"
	fixtureDir := filepath.Join(dir, FixtureName(name))
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "input.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "expected.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write expected: %v", err)
	}

	New(dir).Assert(t, name, "ok")
}

func TestFixtureAssertUpdateGolden(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	name := "update me"
	fixtureDir := filepath.Join(dir, FixtureName(name))
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "input.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "expected.txt"), []byte("before"), 0o644); err != nil {
		t.Fatalf("write expected: %v", err)
	}

	original := os.Getenv("UPDATE_GOLDEN")
	t.Cleanup(func() {
		if original == "" {
			_ = os.Unsetenv("UPDATE_GOLDEN")
		} else {
			_ = os.Setenv("UPDATE_GOLDEN", original)
		}
	})
	if err := os.Setenv("UPDATE_GOLDEN", "1"); err != nil {
		t.Fatalf("set env: %v", err)
	}

	New(dir).Assert(t, name, "after")

	got, err := os.ReadFile(filepath.Join(fixtureDir, "expected.txt"))
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	if string(got) != "after" {
		t.Fatalf("expected golden updated to %q, got %q", "after", string(got))
	}
}

func TestFixtureName(t *testing.T) {
	t.Parallel()

	if got := FixtureName("go test run"); got != "go_test_run" {
		t.Fatalf("FixtureName space sanitize=%q", got)
	}
	if got := FixtureName("a/b"); got != "a_b" {
		t.Fatalf("FixtureName slash sanitize=%q", got)
	}
}
