package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogger_AppendCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "context", "crystallize.log")
	l := New(path)

	if err := l.Append("notes/foo.md", "specs/decisions/0001-foo.md", "adr"); err != nil {
		t.Fatalf("Append: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	line := string(data)
	if !strings.Contains(line, "notes/foo.md") {
		t.Errorf("source not in log: %q", line)
	}
	if !strings.Contains(line, "specs/decisions/0001-foo.md") {
		t.Errorf("target not in log: %q", line)
	}
	if !strings.Contains(line, "[adr]") {
		t.Errorf("type not in log: %q", line)
	}
	if !strings.Contains(line, "→") {
		t.Errorf("arrow separator not in log: %q", line)
	}
}

func TestLogger_AppendIsAppendOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crystallize.log")
	l := New(path)

	entries := []struct{ source, target, typ string }{
		{"notes/a.md", "specs/a.md", "spec"},
		{"notes/b.md", "docs/b.md", "guide"},
		{"notes/c.md", "specs/decisions/0001-c.md", "adr"},
	}
	for _, e := range entries {
		if err := l.Append(e.source, e.target, e.typ); err != nil {
			t.Fatalf("Append %s: %v", e.source, err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) != len(entries) {
		t.Errorf("want %d lines, got %d:\n%s", len(entries), len(lines), content)
	}
	for i, e := range entries {
		if !strings.Contains(lines[i], e.source) {
			t.Errorf("line %d missing source %q: %q", i, e.source, lines[i])
		}
		if !strings.Contains(lines[i], e.target) {
			t.Errorf("line %d missing target %q: %q", i, e.target, lines[i])
		}
	}
}

func TestLogger_AppendFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crystallize.log")
	l := New(path)

	if err := l.Append("notes/plan.md", "specs/plan.md", "spec"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	line := strings.TrimRight(string(data), "\n")
	// Expected format: <RFC3339>\t<source>\t→\t<target>\t[<type>]
	parts := strings.Split(line, "\t")
	if len(parts) != 5 {
		t.Fatalf("want 5 tab-separated fields, got %d: %q", len(parts), line)
	}
	if parts[1] != "notes/plan.md" {
		t.Errorf("source field = %q, want %q", parts[1], "notes/plan.md")
	}
	if parts[2] != "→" {
		t.Errorf("arrow field = %q, want \"→\"", parts[2])
	}
	if parts[3] != "specs/plan.md" {
		t.Errorf("target field = %q, want %q", parts[3], "specs/plan.md")
	}
	if parts[4] != "[spec]" {
		t.Errorf("type field = %q, want \"[spec]\"", parts[4])
	}
}
