package drift

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/audit"
	"github.com/joaoajmatos/oz/internal/audit/drift/specscan"
	"github.com/joaoajmatos/oz/internal/graph"
)

// hasCode reports whether any finding in fs has the given code.
func hasCode(fs []audit.Finding, code string) bool {
	for _, f := range fs {
		if f.Code == code {
			return true
		}
	}
	return false
}

// countCode returns how many findings have the given code.
func countCode(fs []audit.Finding, code string) int {
	n := 0
	for _, f := range fs {
		if f.Code == code {
			n++
		}
	}
	return n
}

// --- DRIFT001: missing code/ paths ---

func TestRunCheck_DRIFT001_PathDoesNotExist(t *testing.T) {
	root := t.TempDir()

	candidates := []specscan.Candidate{
		{File: "specs/api.md", Line: 3, Text: "code/oz/internal/missing/pkg.go", Kind: "path"},
	}

	fs := runCheck(root, nil, candidates)

	if !hasCode(fs, "DRIFT001") {
		t.Fatal("expected DRIFT001 for non-existent path")
	}
	for _, f := range fs {
		if f.Code != "DRIFT001" {
			continue
		}
		if f.File != "specs/api.md" {
			t.Errorf("file = %q, want specs/api.md", f.File)
		}
		if f.Line != 3 {
			t.Errorf("line = %d, want 3", f.Line)
		}
		if len(f.Refs) == 0 || f.Refs[0] != "code/oz/internal/missing/pkg.go" {
			t.Errorf("refs = %v", f.Refs)
		}
	}
}

func TestRunCheck_DRIFT001_PathExists_NoFinding(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "code", "oz", "internal", "audit"), 0755); err != nil {
		t.Fatal(err)
	}

	candidates := []specscan.Candidate{
		{File: "specs/api.md", Line: 1, Text: "code/oz/internal/audit", Kind: "path"},
	}

	fs := runCheck(root, nil, candidates)

	if hasCode(fs, "DRIFT001") {
		t.Error("unexpected DRIFT001 for existing path")
	}
}

func TestRunCheck_DRIFT001_DeduplicatedByPath(t *testing.T) {
	root := t.TempDir()

	// Same missing path referenced from two different spec files — emit once.
	candidates := []specscan.Candidate{
		{File: "specs/a.md", Line: 1, Text: "code/oz/gone.go", Kind: "path"},
		{File: "specs/b.md", Line: 5, Text: "code/oz/gone.go", Kind: "path"},
	}

	fs := runCheck(root, nil, candidates)

	if n := countCode(fs, "DRIFT001"); n != 1 {
		t.Errorf("expected 1 DRIFT001 (deduplicated), got %d", n)
	}
}

// --- DRIFT002: missing identifiers ---

func TestRunCheck_DRIFT002_IdentifierNotInSymbolSet(t *testing.T) {
	root := t.TempDir()

	symbols := []Symbol{
		{Pkg: "example.com/audit", Name: "RunAll", Kind: "func", File: "code/audit/audit.go", Line: 10},
	}
	candidates := []specscan.Candidate{
		{File: "specs/api.md", Line: 2, Text: "DeletedFunc", Kind: "identifier"},
	}

	fs := runCheck(root, symbols, candidates)

	if !hasCode(fs, "DRIFT002") {
		t.Fatal("expected DRIFT002 for unknown identifier")
	}
	for _, f := range fs {
		if f.Code != "DRIFT002" {
			continue
		}
		if f.File != "specs/api.md" {
			t.Errorf("file = %q, want specs/api.md", f.File)
		}
		if f.Line != 2 {
			t.Errorf("line = %d, want 2", f.Line)
		}
	}
}

func TestRunCheck_DRIFT002_KnownUnqualified_NoFinding(t *testing.T) {
	root := t.TempDir()

	symbols := []Symbol{
		{Pkg: "example.com/audit", Name: "RunAll", Kind: "func"},
	}
	candidates := []specscan.Candidate{
		{File: "specs/api.md", Line: 1, Text: "RunAll", Kind: "identifier"},
	}

	fs := runCheck(root, symbols, candidates)

	if hasCode(fs, "DRIFT002") {
		t.Error("unexpected DRIFT002 for known unqualified symbol")
	}
}

func TestRunCheck_DRIFT002_KnownQualified_NoFinding(t *testing.T) {
	root := t.TempDir()

	symbols := []Symbol{
		{Pkg: "example.com/audit", Name: "RunAll", Kind: "func"},
	}
	candidates := []specscan.Candidate{
		{File: "specs/api.md", Line: 1, Text: "audit.RunAll", Kind: "identifier"},
	}

	fs := runCheck(root, symbols, candidates)

	if hasCode(fs, "DRIFT002") {
		t.Error("unexpected DRIFT002 for known qualified symbol")
	}
}

func TestRunCheck_DRIFT002_UnknownQualified(t *testing.T) {
	root := t.TempDir()

	symbols := []Symbol{
		{Pkg: "example.com/audit", Name: "RunAll", Kind: "func"},
	}
	// Package known but symbol was removed.
	candidates := []specscan.Candidate{
		{File: "specs/api.md", Line: 1, Text: "audit.OldFunc", Kind: "identifier"},
	}

	fs := runCheck(root, symbols, candidates)

	if !hasCode(fs, "DRIFT002") {
		t.Error("expected DRIFT002 for unknown qualified identifier")
	}
}

func TestRunCheck_DRIFT002_DeduplicatedPerFileAndText(t *testing.T) {
	root := t.TempDir()

	// Same identifier in the same file twice → one finding for that file.
	// Same identifier in a different file → separate finding.
	candidates := []specscan.Candidate{
		{File: "specs/a.md", Line: 1, Text: "GoneFunc", Kind: "identifier"},
		{File: "specs/a.md", Line: 8, Text: "GoneFunc", Kind: "identifier"},
		{File: "specs/b.md", Line: 2, Text: "GoneFunc", Kind: "identifier"},
	}

	fs := runCheck(root, nil, candidates)

	if n := countCode(fs, "DRIFT002"); n != 2 {
		t.Errorf("expected 2 DRIFT002 findings (one per file), got %d", n)
	}
}

// --- DRIFT003: unmentioned symbols ---

func TestRunCheck_DRIFT003_SymbolNeverMentioned(t *testing.T) {
	root := t.TempDir()

	symbols := []Symbol{
		{Pkg: "example.com/audit", Name: "HiddenHelper", Kind: "func", File: "code/audit/audit.go", Line: 5},
	}

	fs := runCheck(root, symbols, nil)

	if !hasCode(fs, "DRIFT003") {
		t.Fatal("expected DRIFT003 for unmentioned symbol")
	}
	for _, f := range fs {
		if f.Code != "DRIFT003" {
			continue
		}
		if f.File != "code/audit/audit.go" {
			t.Errorf("file = %q, want code/audit/audit.go", f.File)
		}
		if f.Line != 5 {
			t.Errorf("line = %d, want 5", f.Line)
		}
	}
}

func TestRunCheck_DRIFT003_MentionedUnqualified_NoFinding(t *testing.T) {
	root := t.TempDir()

	symbols := []Symbol{
		{Pkg: "example.com/audit", Name: "RunAll", Kind: "func"},
	}
	candidates := []specscan.Candidate{
		{File: "specs/api.md", Line: 1, Text: "RunAll", Kind: "identifier"},
	}

	fs := runCheck(root, symbols, candidates)

	if hasCode(fs, "DRIFT003") {
		t.Error("unexpected DRIFT003: RunAll is mentioned via unqualified form")
	}
}

func TestRunCheck_DRIFT003_MentionedQualified_NoFinding(t *testing.T) {
	root := t.TempDir()

	symbols := []Symbol{
		{Pkg: "example.com/audit", Name: "RunAll", Kind: "func"},
	}
	candidates := []specscan.Candidate{
		{File: "specs/api.md", Line: 1, Text: "audit.RunAll", Kind: "identifier"},
	}

	fs := runCheck(root, symbols, candidates)

	if hasCode(fs, "DRIFT003") {
		t.Error("unexpected DRIFT003: RunAll is mentioned via qualified form")
	}
}

func TestRunCheck_DRIFT003_Severity_IsInfo(t *testing.T) {
	root := t.TempDir()

	symbols := []Symbol{
		{Pkg: "example.com/pkg", Name: "UnmentionedFunc", Kind: "func"},
	}

	fs := runCheck(root, symbols, nil)

	for _, f := range fs {
		if f.Code == "DRIFT003" && f.Severity != audit.SeverityInfo {
			t.Errorf("DRIFT003 severity = %q, want info", f.Severity)
		}
	}
}

// --- Clean: no findings when everything matches ---

func TestRunCheck_Clean(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "code", "oz"), 0755); err != nil {
		t.Fatal(err)
	}

	symbols := []Symbol{
		{Pkg: "example.com/audit", Name: "RunAll", Kind: "func"},
		{Pkg: "example.com/audit", Name: "Report", Kind: "type"},
	}
	candidates := []specscan.Candidate{
		{File: "specs/api.md", Line: 1, Text: "code/oz", Kind: "path"},
		{File: "specs/api.md", Line: 2, Text: "RunAll", Kind: "identifier"},
		{File: "specs/api.md", Line: 3, Text: "Report", Kind: "identifier"},
	}

	fs := runCheck(root, symbols, candidates)

	if len(fs) != 0 {
		t.Errorf("expected no findings for clean workspace, got %+v", fs)
	}
}

// --- symbolSet unit tests ---

func TestSymbolSet_UnqualifiedMatch(t *testing.T) {
	ss := buildSymbolSet([]Symbol{
		{Pkg: "github.com/oz/internal/audit", Name: "RunAll", Kind: "func"},
	})
	if !ss.matches("RunAll") {
		t.Error("expected unqualified match for RunAll")
	}
	if ss.matches("runAll") {
		t.Error("expected no match for unexported runAll")
	}
}

func TestSymbolSet_QualifiedMatch(t *testing.T) {
	ss := buildSymbolSet([]Symbol{
		{Pkg: "github.com/oz/internal/audit", Name: "RunAll", Kind: "func"},
	})
	if !ss.matches("audit.RunAll") {
		t.Error("expected qualified match for audit.RunAll")
	}
	if ss.matches("other.RunAll") {
		t.Error("expected no match for wrong package qualifier")
	}
}

func TestLoadDriftSymbols_IncludeTestsMergesTestExports(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "code", "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "code", "lib", "go.mod"), []byte("module testmod\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "code", "lib", "lib.go"), []byte("package lib\n\nfunc Production() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "code", "lib", "lib_test.go"), []byte("package lib\n\nimport \"testing\"\n\nfunc TestProduction(t *testing.T) {}\n\nfunc OnlyInTestExport() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	g := &graph.Graph{SchemaVersion: graph.SchemaVersion}

	syms, err := loadDriftSymbols(root, g, false)
	if err != nil {
		t.Fatalf("loadDriftSymbols includeTests=false: %v", err)
	}
	if len(syms) != 0 {
		t.Fatalf("want 0 symbols with empty graph and includeTests=false, got %d", len(syms))
	}

	syms, err = loadDriftSymbols(root, g, true)
	if err != nil {
		t.Fatalf("loadDriftSymbols includeTests=true: %v", err)
	}
	found := false
	for _, s := range syms {
		if s.Name == "OnlyInTestExport" {
			found = true
			if !strings.HasSuffix(s.File, "_test.go") {
				t.Errorf("OnlyInTestExport file = %q, want *_test.go", s.File)
			}
		}
	}
	if !found {
		t.Fatalf("expected OnlyInTestExport from _test.go, got %#v", syms)
	}
}

func TestShortPackageName(t *testing.T) {
	cases := []struct {
		pkg  string
		want string
	}{
		{"github.com/joaoajmatos/oz/internal/audit", "audit"},
		{"audit", "audit"},
		{"", ""},
	}
	for _, tc := range cases {
		got := shortPackageName(tc.pkg)
		if got != tc.want {
			t.Errorf("shortPackageName(%q) = %q, want %q", tc.pkg, got, tc.want)
		}
	}
}
