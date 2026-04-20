# ADR-0001: Audit V1 drift symbols from the structural graph (goindexer)

Date: 2026-04-20
Status: Accepted

## Context

Early sprint plans called for a standalone Go AST “extractor” inside `internal/audit/drift`
that re-parsed `code/` on every `oz audit drift` run. In parallel, CI-1 added code indexing
to `oz context build`, emitting `code_file` / `code_symbol` nodes and `contains` edges into
`context/graph.json` using `go/parser` via `internal/codeindex/goindexer`.

Running two independent indexers risks drift between graph and audit, doubles parse work,
and complicates testdata / `_test.go` policy (already centralized in `codeindex.WalkCode`).

## Decision

V1 `oz audit drift` treats **`context/graph.json` as the authoritative symbol set** for
production Go files. `internal/audit/drift.LoadSymbols` maps `code_symbol` nodes to
`drift.Symbol`. Optional `--include-tests` merges additional symbols by walking `code/` for
`*_test.go` only (using `codeindex.WalkCode` with `WalkOpts{IncludeTestGo: true}`), because
the default graph build intentionally skips test files.

Spec references are extracted by `internal/audit/drift/specscan` from markdown under
`specs/` (and `docs/` when `--include-docs` is set).

## Consequences

**Positive**

- Single parsing pipeline for symbols; audit stays fast and aligned with routing context.
- `testdata/` and `_test.go` exclusion for the graph remain one policy in `codeindex`.
- Deterministic symbol order: sorted `(Pkg, Name)` from `LoadSymbols`.

**Negative**

- `oz audit drift` requires a fresh `oz context build` when code changes (already required
  for a correct graph; staleness check `STALE001` surfaces a stale graph).

## Alternatives considered

1. **Standalone audit extractor (original A4 design)** — Rejected: duplicate work and
   divergence risk versus the graph built for query/routing.
2. **Tree-sitter for V1** — Rejected in PRD: extra native dependency and binary cost; Go
   stdlib `go/parser` suffices while `code/` is Go-only.
3. **Always index `_test.go` into graph.json** — Rejected for V1: keeps graph smaller and
   keeps test-only exports out of default drift noise; opt-in via `--include-tests` instead.
