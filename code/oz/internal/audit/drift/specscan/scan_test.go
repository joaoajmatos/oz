package specscan_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/joaoajmatos/oz/internal/audit/drift/specscan"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestScan_ExportedIdentifierInBackticks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "specs", "api.md"), "Use `RunAll` to aggregate findings.\n")

	candidates, err := specscan.Scan(root, specscan.Options{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d: %+v", len(candidates), candidates)
	}
	c := candidates[0]
	if c.Text != "RunAll" {
		t.Errorf("text: got %q, want %q", c.Text, "RunAll")
	}
	if c.Kind != "identifier" {
		t.Errorf("kind: got %q, want %q", c.Kind, "identifier")
	}
	if c.Line != 1 {
		t.Errorf("line: got %d, want 1", c.Line)
	}
	if c.File != "specs/api.md" {
		t.Errorf("file: got %q, want %q", c.File, "specs/api.md")
	}
}

func TestScan_QualifiedIdentifier(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "specs", "api.md"), "See `audit.RunAll` for details.\n")

	candidates, err := specscan.Scan(root, specscan.Options{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Text != "audit.RunAll" {
		t.Errorf("text: got %q", candidates[0].Text)
	}
	if candidates[0].Kind != "identifier" {
		t.Errorf("kind: got %q", candidates[0].Kind)
	}
}

func TestScan_CodePathInBackticks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "specs", "api.md"), "Implementation: `code/oz/internal/audit/audit.go`\n")

	candidates, err := specscan.Scan(root, specscan.Options{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Kind != "path" {
		t.Errorf("kind: got %q, want path", candidates[0].Kind)
	}
	if candidates[0].Text != "code/oz/internal/audit/audit.go" {
		t.Errorf("text: got %q", candidates[0].Text)
	}
}

func TestScan_CodePathInMarkdownLink(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "specs", "api.md"), "See [audit package](code/oz/internal/audit/).\n")

	candidates, err := specscan.Scan(root, specscan.Options{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Kind != "path" {
		t.Errorf("kind: got %q, want path", candidates[0].Kind)
	}
}

func TestScan_UnexportedIdentifierNotCaptured(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "specs", "api.md"), "Use `runAll` internally; `myType` is private.\n")

	candidates, err := specscan.Scan(root, specscan.Options{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates (unexported names), got %d: %+v", len(candidates), candidates)
	}
}

func TestScan_IncludeDocsWalksDocsDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "specs", "api.md"), "See `RunAll`.\n")
	writeFile(t, filepath.Join(root, "docs", "arch.md"), "See `BuildGraph` in `code/oz/internal/context/builder.go`.\n")

	withDocs, err := specscan.Scan(root, specscan.Options{IncludeDocs: true})
	if err != nil {
		t.Fatalf("Scan with docs: %v", err)
	}
	withoutDocs, err := specscan.Scan(root, specscan.Options{IncludeDocs: false})
	if err != nil {
		t.Fatalf("Scan without docs: %v", err)
	}

	if len(withDocs) <= len(withoutDocs) {
		t.Errorf("expected more candidates with IncludeDocs=true; got %d vs %d", len(withDocs), len(withoutDocs))
	}
	// docs/arch.md should contribute at least 2 candidates (identifier + path).
	if len(withDocs)-len(withoutDocs) < 2 {
		t.Errorf("expected >=2 extra candidates from docs/arch.md, got %d extra", len(withDocs)-len(withoutDocs))
	}
}

func TestScan_NoSpecsDir(t *testing.T) {
	root := t.TempDir()
	candidates, err := specscan.Scan(root, specscan.Options{})
	if err != nil {
		t.Fatalf("Scan on workspace with no specs/: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates, got %d", len(candidates))
	}
}

func TestScan_IdentPatterns_CustomOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "specs", "api.md"), "See `OnlyThis` and `RunAll`.\n")

	only := regexp.MustCompile(`^OnlyThis$`)
	candidates, err := specscan.Scan(root, specscan.Options{
		IdentPatterns: []*regexp.Regexp{only},
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate with custom pattern, got %d: %+v", len(candidates), candidates)
	}
	if candidates[0].Text != "OnlyThis" || candidates[0].Kind != "identifier" {
		t.Fatalf("got %+v", candidates[0])
	}
}

func TestScan_IdentPatterns_AppendAlongsideGoDefault(t *testing.T) {
	root := t.TempDir()
	// Not matched by Go default (lowercase); matched by extra pattern.
	writeFile(t, filepath.Join(root, "specs", "api.md"), "Use `ts` for TypeScript.\n")

	goRe := regexp.MustCompile(specscan.DefaultGoExportedIdentPattern)
	tsRe := regexp.MustCompile(`^ts$`)
	candidates, err := specscan.Scan(root, specscan.Options{
		IdentPatterns: []*regexp.Regexp{goRe, tsRe},
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Text != "ts" {
		t.Fatalf("expected single identifier ts, got %+v", candidates)
	}
}

func TestScan_MultipleMatchesPerLine(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "specs", "api.md"),
		"Call `RunAll` then `Serialize` and check `code/oz/cmd/audit.go`.\n")

	candidates, err := specscan.Scan(root, specscan.Options{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates on one line, got %d: %+v", len(candidates), candidates)
	}
	for _, c := range candidates {
		if c.Line != 1 {
			t.Errorf("all candidates should be on line 1, got line %d", c.Line)
		}
	}
}
