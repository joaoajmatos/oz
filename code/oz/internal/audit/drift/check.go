// Package drift provides spec-code drift detection for oz workspaces.
//
// Sprint A5: the orchestrator wires LoadSymbols (from graph.json) and specscan
// together to produce DRIFT001/002/003 findings.
package drift

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oz-tools/oz/internal/audit"
	"github.com/oz-tools/oz/internal/audit/drift/specscan"
	"github.com/oz-tools/oz/internal/codeindex"
	"github.com/oz-tools/oz/internal/codeindex/goindexer"
	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/graph"
)

// Check implements audit.Check for spec-code drift detection.
type Check struct{}

// Name returns the check name.
func (c *Check) Name() string { return "drift" }

// Codes returns the finding codes this check may produce.
func (c *Check) Codes() []string {
	return []string{"DRIFT001", "DRIFT002", "DRIFT003"}
}

// Run loads graph.json and scans specs/ (and optionally docs/) to produce drift findings.
func (c *Check) Run(root string, opts audit.Options) ([]audit.Finding, error) {
	g, err := ozcontext.LoadGraph(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("graph.json not found — run 'oz context build' first")
		}
		return nil, fmt.Errorf("drift: load graph: %w", err)
	}

	symbols, err := loadDriftSymbols(root, g, opts.IncludeTests)
	if err != nil {
		return nil, fmt.Errorf("drift: load symbols: %w", err)
	}

	candidates, err := specscan.Scan(root, specscan.Options{IncludeDocs: opts.IncludeDocs})
	if err != nil {
		return nil, fmt.Errorf("drift: scan specs: %w", err)
	}

	return runCheck(root, symbols, candidates), nil
}

// loadDriftSymbols returns exported symbols from the graph (production Go files).
// When includeTests is true, symbols from *_test.go under code/ are merged in
// (the graph build skips test files by default; see codeindex.WalkOpts).
func loadDriftSymbols(root string, g *graph.Graph, includeTests bool) ([]Symbol, error) {
	symbols := LoadSymbols(g)
	if !includeTests {
		return symbols, nil
	}

	goIdx := goindexer.New()
	files, err := codeindex.WalkCode(root, []codeindex.Indexer{goIdx}, codeindex.WalkOpts{IncludeTestGo: true})
	if err != nil {
		return nil, err
	}
	for _, cf := range files {
		if !strings.HasSuffix(cf.Path, "_test.go") {
			continue
		}
		res, err := goIdx.IndexFile(cf)
		if err != nil {
			return nil, err
		}
		for _, n := range res.Symbols {
			symbols = append(symbols, symbolFromGraphNode(n))
		}
	}
	sortSymbols(symbols)
	return symbols, nil
}

// runCheck is the pure orchestration logic, separated from Run for testability.
func runCheck(root string, symbols []Symbol, candidates []specscan.Candidate) []audit.Finding {
	var findings []audit.Finding
	findings = append(findings, checkMissingPaths(root, candidates)...)
	findings = append(findings, checkMissingIdentifiers(symbols, candidates)...)
	findings = append(findings, checkUnmentionedSymbols(symbols, candidates)...)
	return findings
}

// checkMissingPaths emits DRIFT001 for each code/ path candidate that does not
// exist on disk. One finding per unique path (first occurrence in the spec scan).
func checkMissingPaths(root string, candidates []specscan.Candidate) []audit.Finding {
	seen := make(map[string]bool)
	var findings []audit.Finding
	for _, c := range candidates {
		if c.Kind != "path" {
			continue
		}
		if seen[c.Text] {
			continue
		}
		seen[c.Text] = true
		absPath := filepath.Join(root, filepath.FromSlash(c.Text))
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			findings = append(findings, audit.Finding{
				Check:    "drift",
				Code:     "DRIFT001",
				Severity: audit.SeverityError,
				Message:  fmt.Sprintf("spec references code path %q which does not exist", c.Text),
				File:     c.File,
				Line:     c.Line,
				Hint:     "remove the reference or restore the missing file/directory",
				Refs:     []string{c.Text},
			})
		}
	}
	return findings
}

// checkMissingIdentifiers emits DRIFT002 for each identifier candidate that
// does not match any exported symbol in the graph. One finding per unique
// (spec-file, identifier) pair to avoid noise from repeated mentions.
func checkMissingIdentifiers(symbols []Symbol, candidates []specscan.Candidate) []audit.Finding {
	ss := buildSymbolSet(symbols)
	seen := make(map[string]bool)
	var findings []audit.Finding
	for _, c := range candidates {
		if c.Kind != "identifier" {
			continue
		}
		key := c.File + "\x00" + c.Text
		if seen[key] {
			continue
		}
		seen[key] = true
		if !ss.matches(c.Text) {
			findings = append(findings, audit.Finding{
				Check:    "drift",
				Code:     "DRIFT002",
				Severity: audit.SeverityError,
				Message:  fmt.Sprintf("spec references identifier %q which is not present in the codebase", c.Text),
				File:     c.File,
				Line:     c.Line,
				Hint:     "check if the symbol was renamed or removed; run 'oz context build' to refresh the graph",
				Refs:     []string{c.Text},
			})
		}
	}
	return findings
}

// checkUnmentionedSymbols emits DRIFT003 for each exported symbol that is
// never referenced by any identifier candidate across all scanned files.
func checkUnmentionedSymbols(symbols []Symbol, candidates []specscan.Candidate) []audit.Finding {
	mentioned := buildMentionSet(candidates)
	var findings []audit.Finding
	for _, s := range symbols {
		if mentioned[s.Name] {
			continue
		}
		findings = append(findings, audit.Finding{
			Check:    "drift",
			Code:     "DRIFT003",
			Severity: audit.SeverityWarn,
			Message:  fmt.Sprintf("exported symbol %s.%s is never mentioned in specs or docs", shortPackageName(s.Pkg), s.Name),
			File:     s.File,
			Line:     s.Line,
			Hint:     "document this symbol in specs/ or consider whether it should be exported",
			Refs:     []string{s.Pkg + "." + s.Name},
		})
	}
	return findings
}

// symbolSet supports fast lookup of known symbols by name or qualified form.
type symbolSet struct {
	byName      map[string]bool // "RunAll" → true
	byQualified map[string]bool // "audit.RunAll" → true
}

func buildSymbolSet(symbols []Symbol) symbolSet {
	ss := symbolSet{
		byName:      make(map[string]bool),
		byQualified: make(map[string]bool),
	}
	for _, s := range symbols {
		ss.byName[s.Name] = true
		if short := shortPackageName(s.Pkg); short != "" {
			ss.byQualified[short+"."+s.Name] = true
		}
	}
	return ss
}

// matches returns true if text refers to a known symbol.
// Qualified text (containing ".") is matched against the short-package form;
// unqualified text is matched against symbol names directly.
func (ss symbolSet) matches(text string) bool {
	if strings.Contains(text, ".") {
		return ss.byQualified[text]
	}
	return ss.byName[text]
}

// buildMentionSet returns the set of symbol Names that appear in any
// identifier candidate (qualified or unqualified).
func buildMentionSet(candidates []specscan.Candidate) map[string]bool {
	mentioned := make(map[string]bool)
	for _, c := range candidates {
		if c.Kind != "identifier" {
			continue
		}
		if dot := strings.LastIndex(c.Text, "."); dot >= 0 {
			// "audit.RunAll" → marks "RunAll" as mentioned
			mentioned[c.Text[dot+1:]] = true
		} else {
			mentioned[c.Text] = true
		}
	}
	return mentioned
}

// shortPackageName returns the last path segment of a Go import path.
// "github.com/oz-tools/oz/internal/audit" → "audit"
func shortPackageName(pkg string) string {
	if i := strings.LastIndex(pkg, "/"); i >= 0 {
		return pkg[i+1:]
	}
	return pkg
}
