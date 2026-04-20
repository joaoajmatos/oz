// Package specscan extracts code references from oz workspace markdown files.
//
// It finds two kinds of candidates:
//   - Identifier-shaped tokens inside backticks (default: exported Go shape)
//   - backtick strings or markdown link targets starting with "code/" (path refs)
//
// Results carry file path and 1-based line number for use in drift findings.
//
// Identifier detection is pluggable via Options.IdentPatterns for future
// multi-language indexers (PRD A-14); when unset, DefaultGoExportedIdentPattern applies.
package specscan

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Candidate is a single code reference extracted from a markdown file.
type Candidate struct {
	// File is the workspace-relative path to the markdown source.
	File string

	// Line is the 1-based line number where the candidate was found.
	Line int

	// Text is the raw candidate string (identifier or code/ path).
	Text string

	// Kind is "identifier" or "path".
	Kind string
}

// DefaultGoExportedIdentPattern matches exported Go identifiers in two forms:
//   - Unqualified:  `RunAll`       — starts uppercase, camelCase only
//   - Qualified:    `audit.RunAll` — package (any case) dot exported symbol (uppercase)
//
// Underscores are intentionally excluded: Go exported names are camelCase,
// so ALL_CAPS_UNDERSCORE patterns (e.g. env var names like OPENROUTER_API_KEY)
// are false positives and must be filtered out (AT-01 narrowing, Sprint A5-05).
//
// Callers may append compiled regexps built from this pattern to IdentPatterns
// alongside language-specific patterns.
const DefaultGoExportedIdentPattern = `^(?:[A-Z][A-Za-z0-9]*|[A-Za-z][A-Za-z0-9]*\.[A-Z][A-Za-z0-9]*)$`

// defaultGoExportedIdentRe is compiled from DefaultGoExportedIdentPattern.
var defaultGoExportedIdentRe = regexp.MustCompile(DefaultGoExportedIdentPattern)

// Options controls which directories the scanner walks and how identifiers are detected.
type Options struct {
	// IncludeDocs, when true, also scans files under docs/ in addition to specs/.
	IncludeDocs bool

	// IdentPatterns, when non-empty, defines which backtick tokens count as identifier
	// candidates: a token matches if any pattern's MatchString returns true (patterns
	// should be anchored with ^...$ for whole-token matching).
	//
	// When nil or empty, DefaultGoExportedIdentPattern is used alone. For V2
	// multi-language drift, append patterns for TypeScript, Rust, etc., or replace
	// the default by including a copy compiled from DefaultGoExportedIdentPattern.
	IdentPatterns []*regexp.Regexp
}

// isIdentifierToken reports whether tok should be recorded as an identifier candidate.
func (o Options) isIdentifierToken(tok string) bool {
	patterns := o.IdentPatterns
	if len(patterns) == 0 {
		return defaultGoExportedIdentRe.MatchString(tok)
	}
	for _, re := range patterns {
		if re != nil && re.MatchString(tok) {
			return true
		}
	}
	return false
}

// Scan walks specs/ (and optionally docs/) under root and returns all candidates.
func Scan(root string, opts Options) ([]Candidate, error) {
	dirs := []string{filepath.Join(root, "specs")}
	if opts.IncludeDocs {
		dirs = append(dirs, filepath.Join(root, "docs"))
	}

	var candidates []Candidate
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, de os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if de.IsDir() {
				return nil
			}
			if !strings.HasSuffix(de.Name(), ".md") {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			found, scanErr := scanFile(filepath.ToSlash(rel), path, opts)
			if scanErr != nil {
				return scanErr
			}
			candidates = append(candidates, found...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return candidates, nil
}

// scanFile extracts candidates from a single markdown file.
func scanFile(relPath, absPath string, opts Options) ([]Candidate, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var candidates []Candidate
	lineNum := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		candidates = append(candidates, extractFromLine(relPath, lineNum, line, opts)...)
	}
	return candidates, scanner.Err()
}

// backtickRe matches content inside single backticks (inline code).
var backtickRe = regexp.MustCompile("`([^`]+)`")

// mdLinkRe matches the URL part of a markdown link: [text](url).
var mdLinkRe = regexp.MustCompile(`\]\(([^)]+)\)`)

func extractFromLine(file string, line int, text string, opts Options) []Candidate {
	var out []Candidate

	// Extract inline backtick content.
	for _, m := range backtickRe.FindAllStringSubmatch(text, -1) {
		tok := strings.TrimSpace(m[1])
		if strings.HasPrefix(tok, "code/") {
			out = append(out, Candidate{File: file, Line: line, Text: tok, Kind: "path"})
		} else if opts.isIdentifierToken(tok) {
			out = append(out, Candidate{File: file, Line: line, Text: tok, Kind: "identifier"})
		}
	}

	// Extract markdown link targets.
	for _, m := range mdLinkRe.FindAllStringSubmatch(text, -1) {
		target := strings.TrimSpace(m[1])
		if strings.HasPrefix(target, "code/") {
			out = append(out, Candidate{File: file, Line: line, Text: target, Kind: "path"})
		}
	}

	return out
}
