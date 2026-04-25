# Language Packages

This document normatively defines the `LanguagePackage` interface, the global registry, and the `CodeConcept` type produced during `oz context build`.

---

## Overview

A **language package** is a self-contained Go package that teaches oz how to index one programming language (and optionally one framework). Language packages are compiled into the `oz` binary and self-register via `package init()`. Adding support for a new language requires no changes to the builder or registry — only a new package and a blank import.

The system has three components:

- **`LanguagePackage` interface** (`internal/codeindex`) — the contract each language package must satisfy.
- **Registry** (`internal/codeindex/registry.go`) — a global, mutex-guarded store of registered packages; populated at startup via `init()`.
- **Builder** (`internal/context/builder.go`) — calls `codeindex.Detect(root)` to find active packages, then drives the index and semantics pipeline generically.

---

## `LanguagePackage` interface

Defined in `internal/codeindex/codeindex.go`.

```go
type LanguagePackage interface {
    Language()        string
    Extensions()      []string
    Detect(root string) DetectResult
    IndexFile(f DiscoveredCodeFile, ctx ProjectContext) (*Result, error)
    ExtractSemantics(f DiscoveredCodeFile, ctx ProjectContext) ([]CodeConcept, error)
}
```

### `Language() string`

Returns the canonical language name. MUST be lowercase, stable, and unique across all registered packages (e.g. `"go"`, `"typescript"`, `"python"`). The registry deduplicates by this value — a second `Register` call with the same `Language()` is silently ignored.

### `Extensions() []string`

Returns the file extensions this package handles (e.g. `[]string{".go"}`). Used by `WalkCode` to assign `DiscoveredCodeFile.Lang`. If two registered packages declare the same extension, the first registered wins.

### `Detect(root string) DetectResult`

Called **once per `oz context build` run** at the workspace root. Returns a `DetectResult`:

```go
type DetectResult struct {
    Confidence float64 // 0.0–1.0; 0 means the language is absent
    Framework  string  // detected framework name, e.g. "gin", "nextjs", "" if none
    Manifest   string  // workspace-relative path to the manifest file
}
```

**Invariants:**

- `Confidence == 0` means the language is absent; the registry MUST NOT include this package in active results.
- `Confidence > 0` means the language is present. Values between 0 and 1 are reserved for future multi-language heuristics; implementations SHOULD return `1.0` when the language is definitively detected.
- `Framework` MUST be `""` when no framework is detectable. It MUST NOT be guessed.
- `Manifest` MUST be a workspace-relative slash-delimited path (e.g. `"code/oz/go.mod"`).

Detection MUST be based on manifest files (e.g. `go.mod`, `package.json`, `requirements.txt`) found under `root/code/`. Do not infer language from file extensions alone — `Extensions()` already handles that.

### `IndexFile(f DiscoveredCodeFile, ctx ProjectContext) (*Result, error)`

Called once per discovered source file. `ctx` is populated from `DetectResult`:

```go
type ProjectContext struct {
    Root      string // absolute path to the workspace root
    Framework string // from DetectResult.Framework
    Manifest  string // from DetectResult.Manifest
}
```

The returned `Result` MUST include:

```go
type Result struct {
    FileNode    graph.Node   // code_file node for this file
    Symbols     []graph.Node // code_symbol nodes (exported/public symbols only)
    PackageNode *graph.Node  // optional; code_package grouping node
    Edges       []graph.Edge // contains edges: file → symbol
}
```

**`FileNode` invariants:**
- `ID` MUST be `"code_file:" + f.Path`.
- `Type` MUST be `graph.NodeTypeCodeFile` (`"code_file"`).
- `Language` MUST match `Language()`.

**`Symbols` invariants:**
- Each symbol `ID` MUST be `"code_symbol:" + qualifiedName` where `qualifiedName` uniquely identifies the symbol within the package (e.g. the Go import path + `.` + symbol name).
- `SymbolKind` MUST be one of `"func"`, `"type"`, `"value"`.
- Only **exported** (public) symbols SHOULD be emitted. Private symbols are not useful to LLM consumers.

**`PackageNode` invariants:**
- OPTIONAL. Set when the language has a meaningful grouping concept (Go packages, Python modules, JS/TS modules).
- `ID` MUST be `"code_package:" + canonicalPackageName`.
- `Type` MUST be `graph.NodeTypeCodePackage` (`"code_package"`).
- The builder de-duplicates package nodes across files by `ID`. When multiple files produce the same `ID`, the first non-empty `DocComment` wins.

**`Edges` invariants:**
- Each edge MUST be `EdgeTypeContains` (`"contains"`) from the `FileNode.ID` to a symbol `ID`.
- No other edge types should be emitted from `IndexFile`.

On **parse failure**, return a `Result` with only `FileNode` set and `nil` symbols/edges. Do not return an error for recoverable parse failures — log a warning and continue.

### `ExtractSemantics(f DiscoveredCodeFile, ctx ProjectContext) ([]CodeConcept, error)`

Called once per file immediately after `IndexFile`. Returns zero or more framework-specific concepts extracted by **static analysis only** — no LLM calls. Returning `nil, nil` is always valid.

---

## `CodeConcept` type

```go
type CodeConcept struct {
    Name string            // human-readable, e.g. "GET /api/users"
    Kind string            // dot-namespaced, e.g. "gin.route", "cobra.command"
    File string            // workspace-relative
    Line int
    Refs []string          // node IDs this concept annotates, e.g. ["code_symbol:...Handler"]
    Meta map[string]string // framework-specific key-value pairs
}
```

**Invariants:**

- `Kind` MUST be dot-namespaced: `<language-or-framework>.<concept>` (e.g. `"gin.route"`, `"cobra.command"`, `"django.model"`). This namespace prevents collisions across language packages.
- `Refs` SHOULD contain the `ID` of the symbol that best represents this concept. It MAY be empty when no clean mapping to a graph node exists.
- `CodeConcept` is **not** `semantic.Concept`. It is produced by deterministic static analysis and carries no probabilistic fields.
- Concepts are collected across all files by the builder and returned in `BuildResult.Concepts`. They are persisted to `context/code-concepts.json` by callers that choose to write them.

---

## Registry

Defined in `internal/codeindex/registry.go`.

```go
func Register(pkg LanguagePackage)            // called from package init()
func Detect(root string) []LanguagePackage    // returns packages with Confidence > 0
func All() []LanguagePackage                  // all registered; for tests
```

**Invariants:**

- `Register` is idempotent by `Language()`. A second call with the same language value is silently dropped.
- `Register` is safe for concurrent calls (e.g. parallel `init()` in a test binary) — guarded by `sync.RWMutex`.
- `Detect` calls `Detect(root)` on every registered package and returns those with `Confidence > 0`, in registration order.
- `All` returns all registered packages regardless of `Detect` result. Use this in tests when you want to assert what is registered, not what is active for a given root.

---

## `context/code-concepts.json`

The builder writes concepts to this file via `WriteConcepts(root, concepts, graphHash)` (defined in `internal/context/concepts.go`).

Schema:

```json
{
  "schema_version": "1",
  "graph_hash": "<same hash as context/graph.json>",
  "concepts": [
    {
      "name": "GET /api/users",
      "kind": "gin.route",
      "file": "code/api/handlers/users.go",
      "line": 14,
      "refs": ["code_symbol:github.com/acme/api/handlers.ListUsers"],
      "meta": { "method": "GET", "path": "/api/users" }
    }
  ]
}
```

`graph_hash` ties the concepts file to the graph build that produced it. When they diverge, the concepts file is stale and should be regenerated by running `oz context build`.

---

## Activation

To activate a language package, add a blank import in two places:

```go
// internal/context/builder.go
import _ "github.com/joaoajmatos/oz/internal/codeindex/goindexer"

// internal/audit/drift/check.go
import _ "github.com/joaoajmatos/oz/internal/codeindex/goindexer"
```

The blank import triggers the package's `init()`, which calls `codeindex.Register(New())`. No other wiring is required.
