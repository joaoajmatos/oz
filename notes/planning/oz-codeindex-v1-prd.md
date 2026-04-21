# Product Requirements Document: oz code indexing (V1)

**Author**: oz-spec
**Date**: 2026-04-20
**Status**: Draft
**Stakeholders**: oz-coding, oz-spec, oz-maintainer
**Supersedes**: the "code is recommended under `code/`" gap in `oz context` V1 where code files are not represented in `context/graph.json`.

---

## 1. Executive Summary

`oz context build` will be extended to index source code files under `code/` into `context/graph.json`. V1 adds two graph node types (`code_file`, `code_symbol`) and one edge type (`contains`) so code joins convention artifacts in a single structural graph.

The indexing design is language-agnostic via a new `Indexer` interface. V1 ships a Go indexer implementation; additional languages can be added later by implementing one new package and registering it in the builder.

After this feature, `oz context build` is the single command that produces a complete workspace picture (agents, specs, docs, notes, and code). Drift work planned for Sprint A4 is re-scoped from a standalone extraction pass to graph queries over the indexed code nodes.

---

## 2. Background & Context

Current `context/graph.json` knows only markdown convention artifacts and their links. It currently contains six node types and no representation of source code files or symbols under `code/`.

This creates four concrete issues:

- Agent `owns` edges to code paths often terminate in `path:...` pseudo-IDs rather than real nodes.
- Spec/doc backtick references to `code/*.go` resolve to nothing in the graph.
- `oz context query` and MCP cannot answer ownership questions like "what code does this agent own?" from graph primitives.
- Planned drift extraction (Sprint A4) was forced toward a parallel Go AST pass that duplicates information the graph should already contain.

The correction is to extend `oz context build` itself so the structural graph is code-aware, and drift checks can query the same graph artifact as the rest of the system.

---

## 3. Objectives & Success Metrics

### Goals

1. `oz context build` produces `code_file` and `code_symbol` nodes for all non-test exported Go symbols under `code/`.
2. Agent `owns` edges for direct code file paths resolve to real graph nodes.
3. Spec/doc backtick references to `code/*.go` paths produce `references` edges to indexed code files.
4. The `Indexer` interface enables adding a second language by implementing and registering one new indexer package, without interface changes.
5. Revised drift Sprint A4 can run as graph queries over existing nodes and edges, with no standalone extraction pipeline.

### Non-Goals

1. Multi-language indexing in V1. Go only; TypeScript/Python/etc. are V2.
2. Indexing test files by default. `*_test.go` files are excluded.
3. Indexing vendored code. `vendor/` is excluded.
4. Indexing artifacts outside `code/` (for example `go.sum`, generated artifacts, build output).

### Success Metrics

| Metric | Target |
|---|---|
| No `STALE001` after fresh build | `oz audit staleness` exits 0 immediately after `oz context build` |
| Code nodes in graph | `oz context build` on this repo produces >= 1 `code_file` node per Go package and >= 1 `code_symbol` node per exported symbol |
| `references` edges for code | A spec section containing `` `code/oz/cmd/audit.go` `` produces a `references` edge to the corresponding `code_file` node |
| Query unaffected | `oz context query` still routes to agents correctly; code nodes do not participate in BM25F scoring |
| Determinism | Two consecutive `oz context build` runs produce byte-identical `graph.json` |
| Schema version | `graph.json` carries `"schema_version": "2"` |

---

## 4. Target Users & Segments

**Primary: oz workspace developers (humans + CI)**
Developers and CI pipelines need one deterministic graph build that includes both convention files and code so relationship checks do not split into separate indexing systems.

**Secondary: LLMs working in oz workspaces**
Agents and MCP clients use the graph to answer ownership and reference questions. Code-aware nodes allow these answers to come from first-party graph structure instead of path heuristics.

---

## 5. Solution Overview

### Graph schema changes

V1 adds two node types and one edge type in `internal/graph/graph.go`:

- `code_file` — a source file under `code/`
- `code_symbol` — an exported/public symbol declared in a source file
- `contains` — edge from `code_file` to `code_symbol`

Node ID formats:

- `code_file:code/oz/internal/audit/audit.go`
- `code_symbol:github.com/joaoajmatos/oz/internal/audit.RunAll`

`graph.Node` gains optional fields (all `omitempty`):

```go
Language   string // "go", "typescript", ...
SymbolKind string // "func" | "type" | "value"
Package    string // full import path, e.g. "github.com/joaoajmatos/oz/internal/audit"
Line       int    // source line of symbol declaration
```

`SchemaVersion` is bumped from `"1"` to `"2"`. Existing graphs become intentionally stale and should trigger `STALE001` until rebuilt.

### Extensibility interface

New package: `internal/codeindex/`

```go
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
    Language()   string   // "go"
    Extensions() []string // [".go"]
    IndexFile(f DiscoveredCodeFile) (*Result, error)
}

// WalkCode walks root/code/ and returns all files handled by provided indexers.
// Skips: vendor/, testdata/, and (by default) *_test.go; set opts.IncludeTestGo to include tests.
func WalkCode(root string, indexers []Indexer, opts WalkOpts) ([]DiscoveredCodeFile, error)
```

New package: `internal/codeindex/goindexer/`

- Implements `Indexer` for `.go` files.
- Uses `go/parser` + `go/ast` to extract exported `FuncDecl`, `TypeSpec`, and `ValueSpec`.
- Detects module root by walking up to nearest `go.mod`, reads module path, computes full import path, and caches by directory.
- On parse failure, returns file node only (no symbols), logs warning, and continues (no hard failure).

### Integration path

`internal/context/builder.go` adds a code-index pass after markdown indexing:

```go
codeindex.WalkCode(root, []codeindex.Indexer{goindexer.New()}, codeindex.WalkOpts{})
```

Collected file/symbol nodes and `contains` edges are appended before graph normalization.

No change is required in `extractor.go`: `buildFileNodeMap` already maps file paths generically, so indexed `code_file` nodes automatically participate in `resolveFileRef`, enabling `references` edges for `code/...` mentions in specs/docs.

---

## 6. Decisions

| Question | Decision |
|---|---|
| Where should code indexing happen? | Inside `oz context build`, not as a separate audit extraction phase. |
| What schema granularity is V1? | File + exported symbol granularity (`code_file`, `code_symbol`) with `contains` edges. |
| How is extensibility handled? | A language-agnostic `Indexer` interface with per-language implementations. |
| Which language ships in V1? | Go only (`goindexer`). |
| What about parse failures? | Soft-fail: keep `code_file` node, skip symbols for that file, log warning. |
| Do code nodes influence query routing? | No. BM25F corpus remains agent-focused; code nodes stay out of scoring. |
| What schema version ships? | `"2"` with explicit rebuild requirement for existing graphs. |

---

## 7. Impact on Sprint A4 (drift)

Sprint A4 was previously planned as a standalone Go AST extractor. With code indexed into the graph, that extraction phase is replaced by graph-level checks:

- `DRIFT003` ("symbol never mentioned"): `code_symbol` nodes with no inbound `references` edges from spec/doc nodes.
- `DRIFT001/DRIFT002` ("spec cites missing code"): unresolved `references` targets that still point to `path:code/...` pseudo IDs after resolution.

A4 therefore becomes drift orchestration against graph state, not a parallel indexing subsystem.

---

## 8. Timeline & Phasing

### Phase 1 — CI-0 (Scoping & alignment)

- Lock PRD, pre-mortem, and sprint plan for code indexing.
- Update the existing audit sprint plan to mark legacy A4 as paused pending code indexing.

### Phase 2 — CI-1 (Full implementation)

- Ship schema updates (`code_file`, `code_symbol`, `contains`, schema `"2"`).
- Ship `internal/codeindex` + `internal/codeindex/goindexer`.
- Integrate indexing pass into `context.Builder`.
- Validate determinism, staleness behavior, and self-repo node counts.
- Update architecture docs to include code-index flow and graph schema extension.

### Phase 3 — Drift re-scope handoff

- Resume audit drift sprint as graph-query orchestration using indexed code nodes.
- Remove dependency on any separate Go symbol extraction step.

---

*This PRD is paired with `notes/planning/oz-codeindex-v1-premortem.md` (risk register) and `notes/planning/oz-codeindex-v1-sprints.md` (sprint plan). Implementation starts after CI-0 go/no-go.*
