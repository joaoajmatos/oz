package codeindex

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/oz-tools/oz/internal/graph"
)

// DiscoveredCodeFile is a code file found under code/.
type DiscoveredCodeFile struct {
	Path    string // workspace-relative
	AbsPath string
	Lang    string // resolved from extension via indexer registry
}

// Result holds the nodes produced by indexing one file.
type Result struct {
	FileNode graph.Node   // the code_file node
	Symbols  []graph.Node // code_symbol nodes
	Edges    []graph.Edge // contains edges
}

// Indexer extracts graph nodes from a source file.
type Indexer interface {
	Language() string
	Extensions() []string
	IndexFile(f DiscoveredCodeFile) (*Result, error)
}

// WalkCode walks root/code/ and returns all files handled by provided indexers.
// Skips: vendor/, testdata/, *_test.go, files with no registered indexer.
func WalkCode(root string, indexers []Indexer) ([]DiscoveredCodeFile, error) {
	codeDir := filepath.Join(root, "code")
	if st, err := os.Stat(codeDir); err != nil || !st.IsDir() {
		return nil, nil
	}

	indexersByExt := map[string]Indexer{}
	for _, idx := range indexers {
		for _, ext := range idx.Extensions() {
			indexersByExt[ext] = idx
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
		if strings.HasSuffix(base, "_test.go") {
			return nil
		}

		idx := indexersByExt[filepath.Ext(base)]
		if idx == nil {
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, DiscoveredCodeFile{
			Path:    filepath.ToSlash(rel),
			AbsPath: path,
			Lang:    idx.Language(),
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
