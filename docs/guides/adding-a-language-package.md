# Adding a Language Package

Add first-class support for a new programming language or framework to oz's code indexer.

## Goal

Implement a `LanguagePackage` that teaches oz to discover files, extract symbols, detect the active framework, and produce framework-specific concepts — all without touching the builder or registry.

## Preconditions

- Familiarity with the normative spec: `specs/language-packages.md`.
- A Go parser or AST library for the target language (pure Go preferred; CGO is not available in the oz build).
- The target language must have files under `code/` in the workspace.

---

## Steps

### 1. Create the package

```
code/oz/internal/codeindex/<lang>indexer/<lang>indexer.go
```

Follow the same layout as `goindexer/`:

```
internal/codeindex/
├── codeindex.go          # interface + types (do not modify)
├── registry.go           # registry (do not modify)
└── tsindexer/            # example: TypeScript
    ├── tsindexer.go
    └── tsindexer_test.go
```

### 2. Implement the struct

```go
package tsindexer

import "github.com/joaoajmatos/oz/internal/codeindex"

type Indexer struct{}

func New() *Indexer { return &Indexer{} }

func (i *Indexer) Language() string     { return "typescript" }
func (i *Indexer) Extensions() []string { return []string{".ts", ".tsx"} }
```

### 3. Implement `Detect`

Read the manifest file that proves the language is present. For TypeScript this is `package.json`; for Python it is `requirements.txt` or `pyproject.toml`.

```go
func (i *Indexer) Detect(root string) codeindex.DetectResult {
    codeDir := filepath.Join(root, "code")
    var manifest string
    _ = filepath.WalkDir(codeDir, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            return nil
        }
        if !d.IsDir() && d.Name() == "package.json" {
            rel, _ := filepath.Rel(root, path)
            manifest = filepath.ToSlash(rel)
            return filepath.SkipAll
        }
        return nil
    })
    if manifest == "" {
        return codeindex.DetectResult{}
    }
    framework := detectFramework(filepath.Join(root, manifest))
    return codeindex.DetectResult{Confidence: 1.0, Framework: framework, Manifest: manifest}
}
```

#### Detecting the framework

Read the manifest to find known dependencies. For TypeScript/JavaScript:

```go
func detectFramework(packageJSONPath string) string {
    data, err := os.ReadFile(packageJSONPath)
    if err != nil {
        return ""
    }
    // Simple substring check; a full JSON parse is fine too.
    content := string(data)
    switch {
    case strings.Contains(content, `"next"`):
        return "nextjs"
    case strings.Contains(content, `"react"`):
        return "react"
    case strings.Contains(content, `"vue"`):
        return "vue"
    case strings.Contains(content, `"express"`):
        return "express"
    default:
        return ""
    }
}
```

For Go, scan `go.mod` require lines:

```go
case strings.Contains(line, "github.com/gin-gonic/gin"):
    return "gin"
case strings.Contains(line, "github.com/spf13/cobra"):
    return "cobra"
```

### 4. Implement `IndexFile`

Parse one source file and return symbol nodes. Aim for **exported/public symbols only**.

```go
func (i *Indexer) IndexFile(f codeindex.DiscoveredCodeFile, ctx codeindex.ProjectContext) (*codeindex.Result, error) {
    fileNode := graph.Node{
        ID:       "code_file:" + f.Path,
        Type:     graph.NodeTypeCodeFile,
        File:     f.Path,
        Name:     filepath.Base(f.Path),
        Language: i.Language(),
    }

    // Parse the file (use your language's AST library here).
    symbols, edges, err := parseSymbols(f.AbsPath, f.Path, i.Language())
    if err != nil {
        // Log and return file-only result — don't fail the whole build.
        log.Printf("warning: %s parse failed: %v", f.Path, err)
        return &codeindex.Result{FileNode: fileNode}, nil
    }

    // Emit a package/module node if the language has one.
    var pkgNode *graph.Node
    if moduleName := resolveModule(f.AbsPath); moduleName != "" {
        n := graph.Node{
            ID:       "code_package:" + moduleName,
            Type:     graph.NodeTypeCodePackage,
            File:     f.Path,
            Name:     lastSegment(moduleName),
            Package:  moduleName,
            Language: i.Language(),
        }
        pkgNode = &n
    }

    return &codeindex.Result{
        FileNode:    fileNode,
        Symbols:     symbols,
        PackageNode: pkgNode,
        Edges:       edges,
    }, nil
}
```

**Node ID conventions:**

| Node type | ID format |
|-----------|-----------|
| `code_file` | `code_file:<workspace-relative/path.ts>` |
| `code_symbol` | `code_symbol:<module>.<SymbolName>` |
| `code_package` | `code_package:<canonical-module-name>` |

### 5. Implement `ExtractSemantics`

Return framework-specific concepts. Start with a stub and fill it in per framework:

```go
func (i *Indexer) ExtractSemantics(f codeindex.DiscoveredCodeFile, ctx codeindex.ProjectContext) ([]codeindex.CodeConcept, error) {
    switch ctx.Framework {
    case "nextjs":
        return extractNextJSConcepts(f)
    case "express":
        return extractExpressConcepts(f)
    default:
        return nil, nil
    }
}
```

A concept for a Next.js page route:

```go
codeindex.CodeConcept{
    Name: "/dashboard",
    Kind: "nextjs.page",
    File: f.Path,
    Line: 1,
    Refs: []string{"code_symbol:" + modulePrefix + ".default"},
    Meta: map[string]string{"route": "/dashboard"},
}
```

`Kind` MUST be dot-namespaced: `<framework>.<concept-type>`.

### 6. Self-register in `init()`

```go
func init() { codeindex.Register(New()) }
```

### 7. Activate in the builder and drift checker

Add a blank import in **both** files:

```go
// internal/context/builder.go
import _ "github.com/joaoajmatos/oz/internal/codeindex/tsindexer"

// internal/audit/drift/check.go
import _ "github.com/joaoamatos/oz/internal/codeindex/tsindexer"
```

---

## Write tests

`tsindexer/tsindexer_test.go`:

```go
func TestDetect_PackageJSONFound(t *testing.T) { ... }
func TestDetect_NoPackageJSON(t *testing.T)    { ... }
func TestDetect_FrameworkNextJS(t *testing.T)  { ... }
func TestIndexFile_ExtractsExports(t *testing.T) { ... }
func TestIndexFile_PackageNodeSet(t *testing.T)  { ... }
func TestExtractSemantics_NextJSPage(t *testing.T) { ... }
```

Minimum viable test: create a temp dir, write one source file and a manifest, call `Detect` and `IndexFile`, assert `Confidence > 0` and at least one symbol node.

---

## Verify

```bash
# From code/oz/
go build ./...
go test ./internal/codeindex/<lang>indexer/...
go test ./internal/codeindex/...
go test ./internal/context/...

# End-to-end: build graph in a workspace that uses the new language
oz context build
# Verify new code_file, code_symbol, code_package nodes appear
cat context/graph.json | jq '[.nodes[] | select(.language == "typescript")] | length'
```

---

## Common pitfalls

- **Emitting private/unexported symbols.** LLM consumers only benefit from the public API surface; unexported symbols inflate the graph without adding routing value.
- **Failing the build on parse errors.** Return `Result{FileNode: fileNode}` and log a warning instead of returning an error — one unparseable file should not abort indexing of the whole project.
- **Forgetting the blank import.** Without it, `init()` never runs and the package is never registered, even though it compiles cleanly.
- **Hardcoding the language in `builder.go`.** The builder is generic. If you find yourself adding language-specific logic there, it belongs in the language package instead.
- **`Kind` without a namespace.** Use `"nextjs.page"`, not `"page"`. Un-namespaced kinds will collide across language packages.
