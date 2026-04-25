package codeindex

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/joaoajmatos/oz/internal/graph"
)

// DetectResult is returned by LanguagePackage.Detect.
// Using a struct lets callers ignore unknown fields and lets the interface grow
// without breaking existing implementations.
type DetectResult struct {
	Confidence float64 // 0.0–1.0; 0 means the language is absent
	Framework  string  // e.g. "gin", "cobra", "nextjs", "" if none detected
	Manifest   string  // workspace-relative path to manifest (e.g. "code/oz/go.mod")
}

// ProjectContext is populated once per Build from DetectResult and passed into
// IndexFile and ExtractSemantics for every file of that language.
type ProjectContext struct {
	Root      string // absolute path to workspace root
	Framework string // e.g. "gin", "cobra", "" — from DetectResult.Framework
	Manifest  string // workspace-relative path to manifest — from DetectResult.Manifest
}

// CodeConcept is a framework-specific semantic unit extracted by static analysis.
// It is intentionally separate from semantic.Concept, which is LLM-derived and
// carries probabilistic fields (confidence, reviewed) that do not apply here.
type CodeConcept struct {
	Name string            // human-readable, e.g. "GET /api/users"
	Kind string            // dot-namespaced kind, e.g. "gin.route", "cobra.command"
	File string            // workspace-relative
	Line int
	Refs []string          // node IDs this concept annotates, e.g. ["code_symbol:...Handler"]
	Meta map[string]string // framework-specific key-value pairs
}

// DiscoveredCodeFile is a code file found under code/.
type DiscoveredCodeFile struct {
	Path    string // workspace-relative
	AbsPath string
	Lang    string // resolved from extension via language package registry
}

// Result holds the nodes produced by indexing one file.
type Result struct {
	FileNode    graph.Node   // code_file node
	Symbols     []graph.Node // code_symbol nodes
	Edges       []graph.Edge // contains edges (file → symbol)
	PackageNode *graph.Node  // optional code_package node; builder de-dupes by ID
}

// LanguagePackage is the extensible interface for language integrations.
// Each implementation handles one language (and optionally one framework)
// and self-registers via its package init() function.
type LanguagePackage interface {
	// Language returns the canonical language name (e.g. "go", "typescript").
	Language() string

	// Extensions returns the file extensions this package handles (e.g. ".go").
	Extensions() []string

	// Detect reports whether this language is present in the project rooted at root.
	// Called once per Build run. Confidence == 0 means the language is absent.
	Detect(root string) DetectResult

	// IndexFile extracts graph nodes from one source file.
	// ctx carries the root, framework, and manifest path from Detect.
	// PackageNode in the returned Result is optional; the builder de-dupes by ID.
	IndexFile(f DiscoveredCodeFile, ctx ProjectContext) (*Result, error)

	// ExtractSemantics extracts framework-specific concepts from one source file.
	// Called during Build immediately after IndexFile for the same file.
	// Returns nil when no framework-specific concepts are found.
	ExtractSemantics(f DiscoveredCodeFile, ctx ProjectContext) ([]CodeConcept, error)
}

// WalkOpts configures WalkCode behaviour.
type WalkOpts struct {
	// IncludeTestGo, when true, includes *_test.go files in the walk result.
	// When false (zero value), *_test.go files are skipped — the default for
	// context/graph.json builds. Drift uses IncludeTestGo: true only when
	// merging supplemental test symbols (see internal/audit/drift).
	IncludeTestGo bool
}

// WalkCode walks root/code/ and returns all files handled by the provided
// language packages. Skips: vendor/, testdata/, files with no registered package.
// By default (opts.IncludeTestGo == false) *_test.go files are skipped.
func WalkCode(root string, pkgs []LanguagePackage, opts WalkOpts) ([]DiscoveredCodeFile, error) {
	codeDir := filepath.Join(root, "code")
	if st, err := os.Stat(codeDir); err != nil || !st.IsDir() {
		return nil, nil
	}

	pkgsByExt := map[string]LanguagePackage{}
	for _, pkg := range pkgs {
		for _, ext := range pkg.Extensions() {
			pkgsByExt[ext] = pkg
		}
	}

	var files []DiscoveredCodeFile
	err := filepath.WalkDir(codeDir, func(path string, de os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if de.IsDir() {
			name := de.Name()
			if name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		base := filepath.Base(path)
		if !opts.IncludeTestGo && strings.HasSuffix(base, "_test.go") {
			return nil
		}

		pkg := pkgsByExt[filepath.Ext(base)]
		if pkg == nil {
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, DiscoveredCodeFile{
			Path:    filepath.ToSlash(rel),
			AbsPath: path,
			Lang:    pkg.Language(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(files, func(a, b DiscoveredCodeFile) int {
		return strings.Compare(a.Path, b.Path)
	})
	return files, nil
}
