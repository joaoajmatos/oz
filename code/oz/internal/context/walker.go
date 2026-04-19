// Package context implements oz context build: workspace file discovery,
// parsing, cross-reference extraction, and deterministic graph serialization.
package context

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileKind classifies a discovered workspace file.
type FileKind string

const (
	KindAgentMD         FileKind = "agent_md"
	KindSpec            FileKind = "spec"
	KindDecision        FileKind = "decision"
	KindDoc             FileKind = "doc"
	KindContextSnapshot FileKind = "context_snapshot"
	KindNote            FileKind = "note"
)

// DiscoveredFile is a single file found during workspace traversal.
type DiscoveredFile struct {
	// Path is the file path relative to the workspace root (forward slashes).
	Path string
	// AbsPath is the absolute filesystem path.
	AbsPath string
	// Kind classifies the file within the oz convention.
	Kind FileKind
	// Agent is the agent name for KindAgentMD files; empty otherwise.
	Agent string
}

// Walk traverses root and returns all oz-convention files.
// It respects .ozignore if present. Files are returned in a stable order:
// agents first (sorted by agent name), then specs, docs, context, notes.
func Walk(root string) ([]DiscoveredFile, error) {
	ign, err := loadIgnore(root)
	if err != nil {
		return nil, err
	}

	var files []DiscoveredFile

	// 1. agents/*/AGENT.md — sorted by agent directory name.
	agentsDir := filepath.Join(root, "agents")
	if entries, err := os.ReadDir(agentsDir); err == nil {
		var names []string
		for _, e := range entries {
			if e.IsDir() {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			rel := filepath.Join("agents", name, "AGENT.md")
			abs := filepath.Join(root, rel)
			if _, err := os.Stat(abs); err != nil {
				continue
			}
			if ign.ignored(rel) {
				continue
			}
			files = append(files, DiscoveredFile{
				Path:    filepath.ToSlash(rel),
				AbsPath: abs,
				Kind:    KindAgentMD,
				Agent:   name,
			})
		}
	}

	// 2. Markdown files in each tier directory, sorted by path within each tier.
	type tierSpec struct {
		dir  string
		kind FileKind
	}
	tiers := []tierSpec{
		{"specs", KindSpec},
		{"docs", KindDoc},
		{"context", KindContextSnapshot},
		{"notes", KindNote},
	}

	for _, ts := range tiers {
		dirPath := filepath.Join(root, ts.dir)
		var tier []DiscoveredFile

		_ = filepath.WalkDir(dirPath, func(path string, de os.DirEntry, err error) error {
			if err != nil || de.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".md") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if ign.ignored(rel) {
				return nil
			}

			kind := ts.kind
			if ts.dir == "specs" && strings.HasPrefix(rel, "specs/decisions/") {
				kind = KindDecision
			}

			tier = append(tier, DiscoveredFile{
				Path:    rel,
				AbsPath: path,
				Kind:    kind,
			})
			return nil
		})

		sort.Slice(tier, func(i, j int) bool {
			return tier[i].Path < tier[j].Path
		})
		files = append(files, tier...)
	}

	return files, nil
}

// ignoreList holds patterns loaded from .ozignore.
type ignoreList struct {
	prefixes []string
}

func loadIgnore(root string) (ignoreList, error) {
	var ign ignoreList
	path := filepath.Join(root, ".ozignore")
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return ign, nil
	}
	if err != nil {
		return ign, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ign.prefixes = append(ign.prefixes, line)
	}
	return ign, scanner.Err()
}

// ignored reports whether rel matches any ignore pattern.
// Patterns are treated as path prefixes (forward-slash separated).
func (ign ignoreList) ignored(rel string) bool {
	for _, prefix := range ign.prefixes {
		if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
			return true
		}
	}
	return false
}
