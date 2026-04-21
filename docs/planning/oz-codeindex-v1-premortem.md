# Pre-Mortem: oz code indexing V1

**Author**: oz-spec
**Date**: 2026-04-20
**Status**: Draft
**PRD**: docs/planning/oz-codeindex-v1-prd.md
**Sprint plan**: docs/planning/oz-codeindex-v1-sprints.md

---

## Risk Summary

- **Tigers**: 4 (2 launch-blocking, 2 fast-follow)
- **Paper Tigers**: 2
- **Elephants**: 2

---

## Launch-Blocking Tigers

| # | Risk | Likelihood | Impact | Mitigation | Owner | Deadline |
|---|---|---|---|---|---|---|
| CT-01 | **Graph node explosion makes `graph.json` too large or slow.** The oz repo has ~80 Go files; file+symbol indexing can grow graph size from ~85 nodes to ~1,000+ nodes. This can inflate MCP payloads and increase build/query costs. | Medium | High | Measure node counts immediately after first CI-1 integration run. If total code nodes > 500, apply one or more filters: skip unreferenced internals, add `--no-code-index`, or allow opt-in subdirectory indexing via ignore controls. Record the threshold decision before CI-1 closes. | oz-coding | End of CI-1 |
| CT-02 | **Module-path detection fails outside a Go module root.** Walking up for `go.mod` may fail for edge directories, producing invalid `Package` values and unstable `code_symbol` IDs. | Low | Medium | Bound parent traversal to 10 levels. If no `go.mod` is found, fall back to workspace-relative directory as `Package`, emit warning, continue indexing. Add unit test for a file tree without `go.mod`. | oz-coding | End of CI-1 |

---

## Fast-Follow Tigers

| # | Risk | Likelihood | Impact | Mitigation | Owner | Deadline |
|---|---|---|---|---|---|---|
| CT-03 | **Schema version bump breaks hidden `"1"` assumptions.** Any consumer hard-coding schema string checks can reject V2 graphs. | Low | Low | Search for schema-version string literals and route all checks through `graph.SchemaVersion` constant. Add/refresh tests that assert schema value only via constant paths. | oz-coding | CI-1 code freeze |
| CT-04 | **Query routing quality regresses if code nodes enter BM25F corpus.** If `BuildAgentDocs` accidentally includes code nodes, routing confidence can degrade. | Low | High | Add query-engine test assertion that only `NodeTypeAgent` contributes to scoring corpus. Verify with a graph fixture containing code nodes. | oz-coding | End of CI-1 |

---

## Paper Tigers

**CP-01 — "Wait for multi-language support before indexing code."**
Not a real risk. V1 interface is explicitly extensible. Waiting delays core value (single-source graph) and forces temporary parallel extraction work.

**CP-02 — "`go/parser` cannot cover real Go syntax."**
Not a risk. `go/parser` is the canonical parser for valid Go and is sufficient for exported declaration extraction in V1.

---

## Elephants in the Room

**CE-01 — File-level vs symbol-level indexing**
Symbol-level indexing is required for the key drift question ("is this exported symbol mentioned in specs/docs?"). File-only indexing cannot power `DRIFT003`. Tradeoff is graph growth; CT-01 defines the operational cap and mitigation.

**CE-02 — Graph purpose expands beyond convention files**
The graph began as a convention index (agents/specs/docs/notes/context). V1 code indexing intentionally turns it into a hybrid convention + code index so LLM workflows and drift checks share one structural substrate. This architectural expansion should be documented in `docs/architecture.md`.

---

## Go / No-Go Checklist

### Before CI-1 begins
- [ ] PRD, pre-mortem, and sprint plan are reviewed and frozen.
- [ ] CI-1 acceptance criteria are agreed with owners.

### CI-1 definition of done
- [ ] CT-01 resolved: node count measured and threshold decision documented.
- [ ] `go test ./... -race` passes.
- [ ] `oz context build` on this repo produces `code_file` and `code_symbol` nodes.
- [ ] `oz audit staleness` is clean (`STALE001` absent) after fresh build.
- [ ] `docs/architecture.md` updated to include code indexing pipeline and schema additions.

---

*This pre-mortem is updated as risks close and moves to shipped state when all launch-blocking tigers are resolved.*
