# Pre-Mortem: oz context V1

**Date**: 2026-04-19
**Status**: All items resolved — V1 shipped
**PRD**: docs/oz-context-v1-prd.md

---

## Risk Summary

- **Tigers**: 6 (1 launch-blocking, 3 fast-follow, 2 track)
- **Paper Tigers**: 3
- **Elephants**: 3

---

## Launch-Blocking Tigers

| # | Risk | Likelihood | Impact | Mitigation | Owner | Deadline | Status |
|---|------|-----------|--------|-----------|-------|----------|--------|
| T-01 | **Query scoring algorithm is undefined.** The PRD says "tokenises the query against scope declarations and concept nodes, scores candidates by scope match + concept proximity" — but specifies no algorithm. TF-IDF? BM25? Keyword intersection? The 95% routing accuracy target depends entirely on this decision. It is a P0 requirement with zero implementation guidance. | High | Critical | Before a single line of query logic is written, define and document the scoring algorithm explicitly: token overlap metric, tie-breaking rules, how concept node proximity is weighted vs. direct scope keyword match, minimum score threshold for a valid result. Add this as a new section in the PRD before Phase 1 implementation begins. | oz-spec | Before Phase 1 sprint kick-off | ✅ **Resolved (Sprint 0)**: BM25F with Porter stemming and temperature-scaled softmax specified in PRD §6 and implemented in `internal/query/`. IDF floor of 1.0 added for small corpora. |

---

## Fast-Follow Tigers

| # | Risk | Likelihood | Impact | Planned Response | Owner | Status |
|---|------|-----------|--------|-----------------|-------|--------|
| T-02 | **Leiden community detection degenerates on small graphs.** Leiden finds communities by edge density and works well on graphs with hundreds of nodes. The oz workspace structural graph will have ~20–40 nodes at most (3 agents, ~10 spec sections, ~5 decisions, a handful of docs). On a small sparse graph, Leiden may produce degenerate output: one giant cluster, or every node its own cluster. The claim that "graph topology is the similarity signal" may simply not hold at oz workspace scale. | Medium | High | Before committing to Leiden, prototype it on oz's own workspace and measure community quality. If communities degenerate, fall back to simpler clustering (connected components + trust-tier grouping) which is deterministic and scale-invariant. Don't carry an algorithm that requires a minimum graph density oz will never reach. | oz-coding | ✅ **Resolved**: Leiden not adopted. Multi-field BM25F with an IDF floor (min 1.0) is deterministic and scale-invariant at any corpus size. Tested from 3-agent (01_minimal) to 25-agent (03_large) fixtures. Accuracy targets met. |
| T-03 | **`semantic.json` is not meaningfully human-reviewable.** The PRD's acceptance mechanism is "human reviews and commits." But raw JSON with potentially hundreds of concept nodes and typed edges is not legible in a `git diff`. In practice, developers will rubber-stamp it with `git add context/semantic.json`. If the curated overlay is the trust foundation of enriched routing, and it is not actually curated, the semantic layer is built on a false premise — and every routing result that depends on it is unverified. | High | Medium | Before Phase 2 ships, add `oz context review` — a purpose-built diff view that presents only new/changed nodes and edges in a readable format (table or markdown, not raw JSON), flags low-confidence INFERRED edges for explicit confirmation, and writes a `reviewed: true` marker back to the file. Committing without running review should produce a validation warning from `oz validate`. | oz-coding | ✅ **Resolved (Sprint 6)**: `oz context review` ships with tabular diff view, interactive accept/reject, and `--accept-all` flag. `oz validate` warns on unreviewed nodes. `reviewed: true` is preserved across re-enrichment runs. |
| T-04 | **MCP protocol conformance is non-trivial.** The PRD says "`oz context serve` wraps this in an MCP stdio server" as if it is a minor implementation detail. The MCP stdio protocol has specific framing requirements, capability negotiation, error response shapes, and tool schema conventions. A subtly off-spec implementation will silently fail to work with any MCP client. No conformance test or reference client is mentioned anywhere in the PRD. | Medium | High | Add an explicit acceptance criterion to C-04: the MCP server must be validated against at least one real MCP client (Claude Code's MCP runtime) as part of the Phase 2 definition of done. Write an integration test that spawns `oz context serve` as a subprocess and exercises all four tool calls over stdio. | oz-coding | ✅ **Resolved (Sprint 6)**: MCP server implements JSON-RPC 2.0 over stdio with protocol version `2024-11-05`. Four tools implemented: `query_graph`, `get_node`, `get_neighbors`, `agent_for_task`. Integration test in `internal/mcp/server_test.go` exercises all tool calls. |

---

## Track Tigers

*Monitor post-Phase 1 — trigger conditions noted.*

**T-05 — Semantic overlay staleness** — ✅ **Resolved (Sprint 6)**
Mitigation implemented: on `oz context query` and `oz context serve` startup, the `graph_hash` embedded in `semantic.json` is compared against SHA-256 of the current `graph.json`. If they differ:
```
warning: semantic overlay may be stale — run 'oz context enrich' to update
```
Tested in `internal/semantic/` package.

**T-06 — Default model for enrich is deferred** — ✅ **Resolved (Sprint 5)**
Default model selected: `anthropic/claude-haiku-4`. Strong structured output (JSON mode), reasonable cost per 1K tokens. Documented in `cmd/context.go` flag default and in `internal/enrich/`. Model overridable at runtime via `--model` flag.

---

## Paper Tigers

**P-01 — OpenRouter availability** — ✅ **Not a real risk.** Confirmed: only `oz context enrich` uses the network. All other commands (build, query, review, serve) are fully offline. Enrichment is human-initiated.

**P-02 — API cost of enrichment** — ✅ **Not a real risk.** oz workspaces are small by design. Full enrich on the oz workspace (58 nodes, 28 edges) costs a fraction of a cent on claude-haiku-4.

**P-03 — Markdown section anchors are fragile** — ✅ **Not a real risk.** oz convention files use structured headings without collisions. The indexer handles duplicate headings deterministically (appends suffix). No issues found on the oz workspace itself.

---

## Elephants in the Room

**E-01 — The scoring algorithm is the thing nobody named** — ✅ **Resolved.**
Named and designed before any query code was written. BM25F with multi-field weights, Porter stemming, IDF floor, and softmax routing. Specified in PRD §6. Default weights committed to `context/scoring.toml`. Validated across five golden suites (01_minimal through 05_semantic).

**E-02 — oz's own workspace is too small to validate routing accuracy** — ✅ **Resolved.**
The validation strategy goes far beyond oz's 3-agent workspace. Five golden suites cover:
- `01_minimal`: 3 agents, 20 queries, 100% accuracy target
- `02_medium`: 10 agents, 50 queries, 95% accuracy target
- `03_large`: 25 agents with overlapping scopes, 125 queries, 90% accuracy target
- `04_adversarial`: 20 deliberately ambiguous queries, 65% accuracy target (graceful degradation)
- `05_semantic`: `02_medium` + semantic overlay, 50 queries, 93% accuracy target

All suites pass. `TestRoutingAccuracy` is the regression gate.

**E-03 — "Commit as acceptance" is a process claim with no tooling support** — ✅ **Resolved.**
`oz context review` ships alongside `oz context enrich`. It is not optional tooling — the workflow is:
1. `oz context enrich` writes `semantic.json` with all items `reviewed: false`
2. `oz context review` (or `--accept-all`) sets `reviewed: true` per item
3. `oz validate` warns if any `reviewed: false` items remain in `semantic.json`

The process claim is now backed by tooling enforcement.

---

## Go / No-Go Checklist

### Before Phase 1 implementation begins
- [x] T-01: Query scoring algorithm designed and documented in the PRD
- [x] E-02: Plan for multi-agent validation workspace documented (synthetic test workspace or equivalent)

### Phase 1 definition of done
- [x] Leiden prototype benchmarked on oz workspace; fallback defined if communities degenerate (T-02)
- [x] `graph.json` schema published and frozen
- [x] `oz audit` integration test passing against `graph.json`
- [x] Staleness hash added to `graph.json` schema (for T-05 mitigation in Phase 2)

### Before Phase 2 implementation begins
- [x] Default OpenRouter model selected and documented (T-06)
- [x] `oz context review` workflow designed (T-03)
- [x] MCP conformance test strategy documented (T-04)

### Phase 2 definition of done
- [x] MCP server validated against Claude Code MCP runtime (T-04)
- [x] `oz context review` command ships alongside `oz context enrich` (T-03)
- [x] Staleness warning implemented and tested (T-05)
- [x] `oz validate` warns if `semantic.json` exists but has unreviewed nodes (E-03)

---

*All Tigers and Elephants resolved. oz context V1 shipped April 2026.*
