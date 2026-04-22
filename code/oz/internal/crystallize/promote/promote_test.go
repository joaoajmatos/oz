package promote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixedDate = time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)

// makeRoot creates a minimal workspace directory structure under t.TempDir().
func makeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{
		"specs/decisions",
		"docs/guides",
		"docs",
		"notes",
	} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	return root
}

// --- ADRNumber ---

func TestADRNumber_EmptyDir(t *testing.T) {
	root := makeRoot(t)
	n, err := ADRNumber(filepath.Join(root, "specs", "decisions"))
	if err != nil {
		t.Fatalf("ADRNumber: %v", err)
	}
	if n != 1 {
		t.Errorf("want 1, got %d", n)
	}
}

func TestADRNumber_MissingDir(t *testing.T) {
	n, err := ADRNumber(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("ADRNumber: %v", err)
	}
	if n != 1 {
		t.Errorf("want 1, got %d", n)
	}
}

func TestADRNumber_WithGaps(t *testing.T) {
	root := makeRoot(t)
	dir := filepath.Join(root, "specs", "decisions")
	for _, name := range []string{
		"0001-first.md",
		"0003-third.md", // gap at 0002
		"0007-seventh.md",
		"_template.md", // should be ignored (no leading digits)
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	n, err := ADRNumber(dir)
	if err != nil {
		t.Fatalf("ADRNumber: %v", err)
	}
	if n != 8 {
		t.Errorf("want 8, got %d", n)
	}
}

func TestADRNumber_Deterministic(t *testing.T) {
	root := makeRoot(t)
	dir := filepath.Join(root, "specs", "decisions")
	if err := os.WriteFile(filepath.Join(dir, "0005-something.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	n1, _ := ADRNumber(dir)
	n2, _ := ADRNumber(dir)
	if n1 != n2 || n1 != 6 {
		t.Errorf("ADRNumber not deterministic: %d vs %d (want 6)", n1, n2)
	}
}

// --- slugify ---

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Auth Rewrite", "auth-rewrite"},
		{"ADR-0002: My Decision", "adr-0002-my-decision"},
		{"  leading spaces  ", "leading-spaces"},
		{"hello_world", "hello-world"},
		{"already-slug", "already-slug"},
		{"Foo & Bar -- Baz", "foo-bar-baz"},
	}
	for _, tc := range cases {
		got := slugify(tc.in)
		if got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// --- injectADRTemplate ---

func TestInjectADRTemplate(t *testing.T) {
	content := []byte("We decided to rewrite auth.")
	result := injectADRTemplate(content, 3, "Auth Rewrite", fixedDate)
	got := string(result)

	wantPrefix := "---\nstatus: accepted\ndate: 2026-04-22\n---\n\n# ADR-0003: Auth Rewrite\n\n"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("unexpected prefix:\ngot:  %q\nwant: %q", got, wantPrefix)
	}
	if !strings.HasSuffix(got, string(content)) {
		t.Errorf("original content not preserved at end of result")
	}
}

// --- Promote: all five types ---

func TestPromote_ADR(t *testing.T) {
	root := makeRoot(t)
	content := []byte("We decided to use Go.")
	opts := Options{Today: fixedDate}
	res, err := Promote(root, "notes/decision.md", content, TypeADR, "Use Go", opts)
	if err != nil {
		t.Fatalf("Promote ADR: %v", err)
	}
	wantPath := filepath.Join(root, "specs", "decisions", "0001-use-go.md")
	if res.TargetPath != wantPath {
		t.Errorf("target = %q, want %q", res.TargetPath, wantPath)
	}
	if res.Appended {
		t.Error("Appended should be false for ADR")
	}
	data, _ := os.ReadFile(res.TargetPath)
	if !strings.Contains(string(data), "# ADR-0001: Use Go") {
		t.Errorf("ADR header not injected; got:\n%s", data)
	}
	if !strings.Contains(string(data), "status: accepted") {
		t.Error("ADR frontmatter not injected")
	}
	if !strings.Contains(string(data), "date: 2026-04-22") {
		t.Error("ADR date not injected")
	}
}

func TestPromote_Spec(t *testing.T) {
	root := makeRoot(t)
	content := []byte("All agents MUST read AGENTS.md.")
	res, err := Promote(root, "notes/agent-contract.md", content, TypeSpec, "Agent Contract", Options{})
	if err != nil {
		t.Fatalf("Promote Spec: %v", err)
	}
	wantPath := filepath.Join(root, "specs", "agent-contract.md")
	if res.TargetPath != wantPath {
		t.Errorf("target = %q, want %q", res.TargetPath, wantPath)
	}
	data, _ := os.ReadFile(res.TargetPath)
	if string(data) != string(content) {
		t.Error("spec content not written verbatim")
	}
}

func TestPromote_Guide(t *testing.T) {
	root := makeRoot(t)
	content := []byte("Step 1: install. Step 2: run.")
	res, err := Promote(root, "notes/setup.md", content, TypeGuide, "Setup Guide", Options{})
	if err != nil {
		t.Fatalf("Promote Guide: %v", err)
	}
	wantPath := filepath.Join(root, "docs", "guides", "setup-guide.md")
	if res.TargetPath != wantPath {
		t.Errorf("target = %q, want %q", res.TargetPath, wantPath)
	}
}

func TestPromote_Arch(t *testing.T) {
	root := makeRoot(t)
	content := []byte("Three-layer architecture...")
	res, err := Promote(root, "notes/layers.md", content, TypeArch, "Graph Layers", Options{})
	if err != nil {
		t.Fatalf("Promote Arch: %v", err)
	}
	wantPath := filepath.Join(root, "docs", "graph-layers.md")
	if res.TargetPath != wantPath {
		t.Errorf("target = %q, want %q", res.TargetPath, wantPath)
	}
}

func TestPromote_OpenItem_CreateFile(t *testing.T) {
	root := makeRoot(t)
	content := []byte("We need to decide on the storage format.")
	res, err := Promote(root, "notes/open-q.md", content, TypeOpenItem, "Storage Format", Options{})
	if err != nil {
		t.Fatalf("Promote OpenItem: %v", err)
	}
	if !res.Appended {
		t.Error("Appended should be true for open-item")
	}
	wantPath := filepath.Join(root, "docs", "open-items.md")
	if res.TargetPath != wantPath {
		t.Errorf("target = %q, want %q", res.TargetPath, wantPath)
	}
	data, _ := os.ReadFile(res.TargetPath)
	got := string(data)
	if !strings.HasPrefix(got, "# Open Items\n") {
		t.Errorf("missing header; got:\n%s", got)
	}
	if !strings.Contains(got, "## Storage Format") {
		t.Errorf("section header not present; got:\n%s", got)
	}
	if !strings.Contains(got, string(content)) {
		t.Errorf("content not present; got:\n%s", got)
	}
}

func TestPromote_OpenItem_AppendToExisting(t *testing.T) {
	root := makeRoot(t)
	existing := "# Open Items\n\n## First Item\n\nSome content.\n"
	target := filepath.Join(root, "docs", "open-items.md")
	if err := os.WriteFile(target, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Promote(root, "notes/q2.md", []byte("Second question."), TypeOpenItem, "Second Item", Options{})
	if err != nil {
		t.Fatalf("Promote OpenItem append: %v", err)
	}
	data, _ := os.ReadFile(target)
	got := string(data)
	if !strings.Contains(got, "## First Item") {
		t.Error("first item lost after append")
	}
	if !strings.Contains(got, "## Second Item") {
		t.Error("second item not appended")
	}
	// Existing content must come before the new section.
	if strings.Index(got, "## Second Item") < strings.Index(got, "## First Item") {
		t.Error("new item appears before existing item")
	}
}

func TestPromote_UnsupportedType(t *testing.T) {
	root := makeRoot(t)
	_, err := Promote(root, "notes/x.md", []byte("x"), "bogus-type", "X", Options{})
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

// --- Collision guard ---

func TestPromote_CollisionGuard(t *testing.T) {
	root := makeRoot(t)
	content := []byte("spec content")
	// First promotion succeeds.
	_, err := Promote(root, "notes/s.md", content, TypeSpec, "My Spec", Options{})
	if err != nil {
		t.Fatalf("first promote: %v", err)
	}
	// Second promotion on the same target returns CollisionError.
	_, err = Promote(root, "notes/s2.md", []byte("new content"), TypeSpec, "My Spec", Options{})
	if err == nil {
		t.Fatal("expected CollisionError, got nil")
	}
	var ce *CollisionError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *CollisionError, got %T: %v", err, err)
	}
	if string(ce.Existing) != string(content) {
		t.Errorf("CollisionError.Existing = %q, want %q", ce.Existing, content)
	}
}

// --- TargetPath ---

func TestTargetPath(t *testing.T) {
	root := makeRoot(t)
	cases := []struct {
		artifactType string
		title        string
		wantSuffix   string
	}{
		{TypeSpec, "Agent Contract", "specs/agent-contract.md"},
		{TypeGuide, "Setup Guide", "docs/guides/setup-guide.md"},
		{TypeArch, "Graph Layers", "docs/graph-layers.md"},
		{TypeOpenItem, "Anything", "docs/open-items.md"},
	}
	for _, tc := range cases {
		got, err := TargetPath(root, tc.artifactType, tc.title)
		if err != nil {
			t.Errorf("TargetPath(%q, %q): %v", tc.artifactType, tc.title, err)
			continue
		}
		want := filepath.Join(root, filepath.FromSlash(tc.wantSuffix))
		if got != want {
			t.Errorf("TargetPath(%q, %q) = %q, want %q", tc.artifactType, tc.title, got, want)
		}
	}
}

func TestTargetPath_ADR(t *testing.T) {
	root := makeRoot(t)
	// Place two existing ADRs so the next number is 3.
	dir := filepath.Join(root, "specs", "decisions")
	for _, f := range []string{"0001-first.md", "0002-second.md"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := TargetPath(root, TypeADR, "My Decision")
	if err != nil {
		t.Fatalf("TargetPath ADR: %v", err)
	}
	want := filepath.Join(root, "specs", "decisions", "0003-my-decision.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- Kill-mid-write atomicity ---

// TestAtomicWrite_KillMidWrite simulates a process kill between the tmp write
// and the rename. The target must not exist after the simulated kill.
// A subsequent call must succeed and clean up the stale .tmp.
func TestAtomicWrite_KillMidWrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.md")
	content := []byte("hello atomicity")

	// Install hook that panics to simulate an abrupt kill.
	afterWriteHook = func() {
		panic("simulated kill")
	}
	defer func() { afterWriteHook = nil }()

	func() {
		defer func() { recover() }() //nolint:errcheck
		_ = atomicWrite(target, content)
	}()

	// Target must NOT exist — rename never happened.
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Error("target exists after simulated kill — atomicity violated")
	}
	// .tmp will exist (written before the hook panicked).
	if _, err := os.Stat(target + ".tmp"); errors.Is(err, os.ErrNotExist) {
		t.Error(".tmp does not exist — cannot verify cleanup on next call")
	}

	// A second call must succeed, cleaning up the stale .tmp.
	afterWriteHook = nil
	if err := atomicWrite(target, content); err != nil {
		t.Fatalf("second atomicWrite failed: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target does not exist after successful write: %v", err)
	}
	// Stale .tmp must be gone after a successful rename.
	if _, err := os.Stat(target + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Error("stale .tmp still exists after successful second write")
	}
}

// Run TestAtomicWrite_KillMidWrite five times to catch flakiness.
func TestAtomicWrite_KillMidWrite_Repeated(t *testing.T) {
	for i := range 5 {
		t.Run(fmt.Sprintf("run%d", i+1), func(t *testing.T) {
			TestAtomicWrite_KillMidWrite(t)
		})
	}
}

