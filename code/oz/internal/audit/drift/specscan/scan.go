// Package specscan extracts code references from oz workspace markdown files.
//
// It finds two kinds of candidates:
//   - Go-identifier-shaped tokens inside backticks (e.g. `MyFunc`, `pkg.MyType`)
//   - backtick strings or markdown link targets starting with "code/" (path refs)
//
// Results carry file path and 1-based line number for use in drift findings.
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

// Options controls which directories the scanner walks.
type Options struct {
	// IncludeDocs, when true, also scans files under docs/ in addition to specs/.
	IncludeDocs bool
}

// goIdentRe matches exported Go identifiers in two forms:
//   - Unqualified:  `RunAll`       — starts uppercase
//   - Qualified:    `audit.RunAll` — package (any case) dot exported symbol (uppercase)
//
// The qualified form uses lowercase-permissive package names to match real Go
// conventions (e.g. `graph.Node`, `audit.RunAll`).
var goIdentRe = regexp.MustCompile(`^(?:[A-Z][A-Za-z0-9_]*|[A-Za-z][A-Za-z0-9_]*\.[A-Z][A-Za-z0-9_]*)$`)

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
			found, scanErr := scanFile(filepath.ToSlash(rel), path)
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
func scanFile(relPath, absPath string) ([]Candidate, error) {
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
		candidates = append(candidates, extractFromLine(relPath, lineNum, line)...)
	}
	return candidates, scanner.Err()
}

// backtickRe matches content inside single backticks (inline code).
var backtickRe = regexp.MustCompile("`([^`]+)`")

// mdLinkRe matches the URL part of a markdown link: [text](url).
var mdLinkRe = regexp.MustCompile(`\]\(([^)]+)\)`)

func extractFromLine(file string, line int, text string) []Candidate {
	var out []Candidate

	// Extract inline backtick content.
	for _, m := range backtickRe.FindAllStringSubmatch(text, -1) {
		tok := strings.TrimSpace(m[1])
		if strings.HasPrefix(tok, "code/") {
			out = append(out, Candidate{File: file, Line: line, Text: tok, Kind: "path"})
		} else if goIdentRe.MatchString(tok) {
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
