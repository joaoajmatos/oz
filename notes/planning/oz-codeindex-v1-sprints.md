# oz code indexing V1 — Sprint Plan

**Author**: oz-spec
**Date**: 2026-04-20
**Status**: Draft
**PRD**: notes/planning/oz-codeindex-v1-prd.md
**Pre-mortem**: notes/planning/oz-codeindex-v1-premortem.md
**Format**: 1-week sprints, solo developer

---

## Sprint CI-0 — Scoping & alignment

**Goal**: Lock planning artifacts for code indexing. No implementation code changes.

### Stories

| # | Story | Acceptance Criteria |
|---|---|---|
| CI-0-01 | Draft and freeze `notes/planning/oz-codeindex-v1-prd.md` | PRD includes schema changes, indexer interface, integration path, and drift A4 impact section. |
| CI-0-02 | Draft and freeze `notes/planning/oz-codeindex-v1-premortem.md` | CT-01 and CT-02 are listed as launch-blocking tigers with mitigation/owner/deadline. |
| CI-0-03 | Draft and freeze `notes/planning/oz-codeindex-v1-sprints.md` | CI-0 and CI-1 are defined with story-level acceptance criteria and explicit DoD. |
| CI-0-04 | Update `notes/planning/oz-audit-v1-sprints.md` to pause legacy A4 | Sprint A4 section contains PAUSED notice and re-scope statement toward graph-query drift orchestration. |

### Definition of done

- All three code-index planning docs are committed.
- `notes/planning/oz-audit-v1-sprints.md` contains the A4 paused notice.
- Scope handoff to CI-1 is explicit and unambiguous.

---

## Sprint CI-1 — Full implementation

**Goal**: `oz context build` emits code nodes and containment edges for exported Go symbols under `code/`.

### Stories

| # | Story | Acceptance Criteria |
|---|---|---|
| CI-1-01 | Graph schema update: add `code_file`, `code_symbol`, `contains`; add `Language`, `SymbolKind`, `Package`, `Line`; bump schema `"1"` -> `"2"` | `internal/graph/graph.go` updated; schema constant is `"2"`; tests referencing schema version are updated. |
| CI-1-02 | `internal/codeindex/codeindex.go` with `Indexer`, `DiscoveredCodeFile`, `Result`, `WalkCode` | `WalkCode` skips `vendor/`, `testdata/`, and `*_test.go`; only files with registered indexers are returned; unit tests cover walk + skip behavior. |
| CI-1-03 | `internal/codeindex/goindexer/goindexer.go` Go AST indexer | Extracts exported `FuncDecl`, `TypeSpec`, `ValueSpec`; resolves package import path via nearest `go.mod`; parse failures return file node only and warning; table-driven tests with fixtures. |
| CI-1-04 | Integrate code indexing pass in `internal/context/builder.go` | After markdown indexing, builder calls `WalkCode(..., []Indexer{goindexer.New()}, WalkOpts{})`; appends nodes/edges before normalize; build remains deterministic. |
| CI-1-05 | Determinism + staleness verification | Two consecutive builds produce byte-identical `context/graph.json`; `oz audit staleness` is clean (`STALE001` absent) after rebuild. |
| CI-1-06 | Self-validation on this repository | Assert >= 1 `code_file` node per Go package and >= 1 `code_symbol` node per exported symbol; measure total code-node count; if > 500 apply CT-01 mitigation and document decision in `docs/open-items.md`. |
| CI-1-07 | Architecture documentation update | `docs/architecture.md` includes new node/edge types and the code-indexing pipeline in context build flow. |

### Definition of done

- `go test ./... -race` passes.
- `oz context build` on this repo produces code nodes.
- `oz audit staleness` is clean after rebuild.
- CT-01 threshold decision is made and documented.
- `docs/architecture.md` is updated.

---

## Summary

| Sprint | Goal | Output |
|---|---|---|
| CI-0 | Lock planning and re-scope dependencies | PRD + pre-mortem + sprint plan + paused A4 note |
| CI-1 | Ship code indexing in context graph | Schema v2 + code indexer + builder integration + docs update |

**Critical path**: CI-0 must close before CI-1 starts. CI-1 completion is the prerequisite for re-scoping audit drift Sprint A4 into graph-query orchestration.
