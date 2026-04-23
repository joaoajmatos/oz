package goindexer

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaoajmatos/oz/internal/codeindex"
	"github.com/joaoajmatos/oz/internal/graph"
)

// Indexer extracts exported symbols from Go files.
type Indexer struct {
	moduleCache map[string]string // go.mod dir -> module path
}

// New returns a Go AST-based code indexer.
func New() *Indexer {
	return &Indexer{
		moduleCache: map[string]string{},
	}
}

func (i *Indexer) Language() string { return "go" }

func (i *Indexer) Extensions() []string { return []string{".go"} }

func (i *Indexer) IndexFile(f codeindex.DiscoveredCodeFile) (*codeindex.Result, error) {
	pkgPath, err := i.resolvePackagePath(f.AbsPath)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	parsed, parseErr := parser.ParseFile(fset, f.AbsPath, nil, parser.ParseComments)

	fileNode := graph.Node{
		ID:       "code_file:" + f.Path,
		Type:     graph.NodeTypeCodeFile,
		File:     f.Path,
		Name:     filepath.Base(f.Path),
		Language: i.Language(),
		Package:  pkgPath,
	}
	if parseErr == nil && parsed.Doc != nil {
		fileNode.DocComment = strings.TrimSpace(parsed.Doc.Text())
	}

	if parseErr != nil {
		log.Printf("warning: goindexer parse failed for %s: %v", f.Path, parseErr)
		return &codeindex.Result{FileNode: fileNode}, nil
	}

	var symbols []graph.Node
	var edges []graph.Edge
	for _, decl := range parsed.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name == nil || !d.Name.IsExported() {
				continue
			}
			symbolID := fmt.Sprintf("code_symbol:%s.%s", pkgPath, d.Name.Name)
			symbols = append(symbols, graph.Node{
				ID:         symbolID,
				Type:       graph.NodeTypeCodeSymbol,
				File:       f.Path,
				Name:       d.Name.Name,
				Language:   i.Language(),
				SymbolKind: "func",
				Package:    pkgPath,
				Line:       fset.Position(d.Pos()).Line,
			})
			edges = append(edges, graph.Edge{
				From: fileNode.ID,
				To:   symbolID,
				Type: graph.EdgeTypeContains,
			})
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name == nil || !s.Name.IsExported() {
						continue
					}
					symbolID := fmt.Sprintf("code_symbol:%s.%s", pkgPath, s.Name.Name)
					symbols = append(symbols, graph.Node{
						ID:         symbolID,
						Type:       graph.NodeTypeCodeSymbol,
						File:       f.Path,
						Name:       s.Name.Name,
						Language:   i.Language(),
						SymbolKind: "type",
						Package:    pkgPath,
						Line:       fset.Position(s.Pos()).Line,
					})
					edges = append(edges, graph.Edge{
						From: fileNode.ID,
						To:   symbolID,
						Type: graph.EdgeTypeContains,
					})
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if n == nil || !n.IsExported() {
							continue
						}
						symbolID := fmt.Sprintf("code_symbol:%s.%s", pkgPath, n.Name)
						symbols = append(symbols, graph.Node{
							ID:         symbolID,
							Type:       graph.NodeTypeCodeSymbol,
							File:       f.Path,
							Name:       n.Name,
							Language:   i.Language(),
							SymbolKind: "value",
							Package:    pkgPath,
							Line:       fset.Position(n.Pos()).Line,
						})
						edges = append(edges, graph.Edge{
							From: fileNode.ID,
							To:   symbolID,
							Type: graph.EdgeTypeContains,
						})
					}
				}
			}
		}
	}

	return &codeindex.Result{
		FileNode: fileNode,
		Symbols:  symbols,
		Edges:    edges,
	}, nil
}

func (i *Indexer) resolvePackagePath(absPath string) (string, error) {
	dir := filepath.Dir(absPath)
	goModDir, modulePath, err := i.findNearestModule(dir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(goModDir, dir)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return modulePath, nil
	}
	return modulePath + "/" + rel, nil
}

func (i *Indexer) findNearestModule(startDir string) (string, string, error) {
	dir := startDir
	for {
		if modulePath, ok := i.moduleCache[dir]; ok {
			return dir, modulePath, nil
		}

		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			modulePath, parseErr := readModulePath(goModPath)
			if parseErr != nil {
				return "", "", parseErr
			}
			i.moduleCache[dir] = modulePath
			return dir, modulePath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", fmt.Errorf("no go.mod found for %s", startDir)
		}
		dir = parent
	}
}

func readModulePath(goModPath string) (string, error) {
	f, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			module := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if module != "" {
				return module, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("module directive not found in %s", goModPath)
}
