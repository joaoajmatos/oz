# Sprint Plan: oz Semantic Code Understanding

**Date**: 2026-04-23  
**Duration**: 2 weeks  
**PRD**: `notes/PRD-semantic-code-understanding.md`  
**Pre-mortem**: `notes/premortem-semantic-code-understanding.md`

---

## Sprint Goal

By end of sprint, `oz context enrich` produces package-level concept edges stored in `semantic.json`, `oz audit` surfaces real spec coverage gaps (COV005) instead of DRIFT003 noise, and unreviewed edges are never visible to query results.

---

## Capacity

| | |
|--|--|
| Team | 1 developer (solo) |
| Working days | 8 (2-week sprint, accounting for overhead) |
| Productive hours/day | ~5h |
| Total available | ~40h |
| Buffer (20%) | 8h |
| **Committed capacity** | **~32h / 16 story points** |

*1 story point ≈ 2 focused hours. Buffer absorbs review iterations, prompt tuning, and unexpected Go toolchain friction.*

---

## Stories

| # | Story | Points | Depends on | Notes |
|---|-------|--------|-----------|-------|
| S1 | **Design decision: review gate** — Document and commit to one of: (a) `confidence > 0.85` auto-promotes, low-confidence requires review; or (b) all edges need human review but `oz context review` is fast enough. Write the decision as an ADR in `specs/decisions/`. | 1 | — | **Must complete before S5. This is the pre-mortem E1 elephant.** |
| S2 | **Graph schema: `code_package` node + `implements` edge** — Add new node kind `code_package` and edge kind `implements` to `graph.go`. Update schema version. Write unit tests for round-trip serialization. | 2 | — | Low risk. Pure schema extension. |
| S3 | **goindexer: emit package-level nodes** — Extend `goindexer.go` to walk packages and emit one `code_package` node per package with: import path, list of exported symbol IDs, and optional first doc comment block. | 2 | S2 | Packages already implicitly exist in the graph via `contains` edges — this makes them first-class nodes. |
| S4 | **Enrichment prompt: docs + code packages sections** — Extend `prompt.go` with two new input sections: (1) **Docs** — include `docs/` content to seed more implementation-adjacent concepts before code mapping; (2) **Code Packages** — package path, doc comment, exported symbol names. Add source-of-truth note to prompt: when doc-derived concepts conflict with spec-derived ones, spec wins. Ask LLM to produce per-package intent sentences and map to existing concept IDs. | 2 | S2 | Keep prompt additive — do not restructure existing agent/spec sections. Test prompt output manually before wiring into pipeline. Docs section goes first to densify concept graph before code section runs. |
| S5 | **Enrichment pipeline: process `implements` edges** — Extend `enrich.go` to: (a) collect `code_package` nodes from graph, (b) include them in LLM request, (c) parse LLM response for package→concept mappings, (d) write `implements` edges to `semantic.json` with `reviewed: false` and confidence score. Apply the review gate decision from S1. | 3 | S1, S3, S4 | Highest complexity story. Spike: run one real enrichment call on oz's own packages before finalising the implementation. |
| S6 | **Pre-flight: concept graph density check** — Before running code enrichment, check that `semantic.json` contains ≥15 concept nodes. If not, print a clear error: "Concept graph too sparse — run `oz context enrich` on specs first, or use `--force` to skip this check." | 1 | S5 | Addresses launch-blocking risk T1 from pre-mortem. |
| S7 | **Query: hide unreviewed `implements` edges** — Update query engine / scoring so that `implements` edges with `reviewed: false` are excluded from traversal. They remain in `semantic.json` and are visible to `oz audit` — just not to `oz context query`. | 2 | S2 | Addresses launch-blocking risk T2. Add a `--include-unreviewed` flag for debugging. |
| S8 | **Audit: COV005** — Add check: any concept node with no inbound `implements` edges → `warn`. Message: "Concept '{name}' has no implementing code. Add it to a package's concept mapping or remove the concept." | 2 | S2, S7 | Ship this first among audit changes. This is the high-value direction: spec describes something unbuilt. |
| S9 | **Audit: DRIFT003 demotion** — Change DRIFT003 severity from `warn` to `info`. Update `audit-catalogue.md` and any docs that reference it. Add release note. | 1 | — | Low risk. No logic changes, just severity constant + docs. Can be done any time. |
| S10 | **End-to-end validation on oz's own codebase** — Run the full pipeline on oz itself: `oz context build` → `oz context enrich` → `oz audit`. Verify: (a) each package in `code/oz/internal/` has at least one `implements` edge, (b) COV005 fires for at least one concept, (c) DRIFT003 is info not warn, (d) `oz context query "how is drift detection implemented?"` returns packages. Document findings. | 2 | S5, S6, S7, S8, S9 | This is the sprint's acceptance test. If this doesn't work, the sprint doesn't ship. |

**Total committed: 18 points / ~36h**  
**Buffer used: 2 points remaining for prompt iteration, LLM response parsing edge cases**

---

## Critical Path

```
S1 (design) ──┐
S2 (schema) ──┼──► S3 (indexer) ──► S5 (pipeline) ──► S6 (preflight) ──► S10 (e2e)
              └──► S4 (prompt)  ──►                                        ↑
S2 ──────────────► S7 (query)   ──────────────────────────────────────────┤
S2 ──────────────► S8 (COV005)  ──────────────────────────────────────────┤
S9 (independent) ─────────────────────────────────────────────────────────┘
```

**Day 1–2**: S1, S2, S9 in parallel (decisions + schema foundations)  
**Day 3–5**: S3, S4, S7, S8 in parallel (all depend on S2, not each other)  
**Day 6–8**: S5 (pipeline, needs S1+S3+S4), then S6, then S10

---

## Sequencing by Day

| Day | Work |
|-----|------|
| 1 | S1: Write review gate ADR. S9: Demote DRIFT003. |
| 2 | S2: Graph schema changes + tests. |
| 3 | S3: goindexer package nodes. |
| 4 | S4: Enrichment prompt extension. Manual prompt test. |
| 5 | S7: Hide unreviewed edges from query. S8: COV005 audit check. |
| 6 | S5: Wire enrichment pipeline (start). |
| 7 | S5: Wire enrichment pipeline (finish). Real LLM call smoke test. |
| 8 | S6: Pre-flight check. S10: End-to-end validation + fix-up. |

---

## Risks

| Risk | Mitigation |
|------|-----------|
| LLM response format is unpredictable — package→concept mapping may not parse cleanly | Spike on S4 output format before writing S5 parser. Define strict JSON schema for LLM response. |
| oz's concept graph currently has <15 concepts — S6 pre-flight blocks the whole feature | Including docs in S4 partially mitigates this. Check concept count before sprint starts. If still sparse after docs enrichment, add explicit spec enrichment pass as Day 0 task (30 min). |
| S5 takes longer than 3 points — prompt tuning and LLM response parsing often snowball | Timebox prompt iteration to 2 cycles. Ship with "best effort" confidence scores rather than perfect mappings. |
| S10 reveals COV006 (package with no concept) is noisy before it's built | COV006 is explicitly out of scope for this sprint (fast-follow). If it appears, document as backlog item. |

## Pre-Sprint Observations (from first enrichment run, 2026-04-23)

**Concept count is 10, not 15**  
The S6 pre-flight threshold was set at ≥15 but the current graph has 10 concepts. Lower the threshold to ≥10 for V1. The gap is partly caused by the skipped edges bug below — concepts exist but are less connected than they should be. Docs enrichment (S4) should increase density further.

**Skipped edges reveal a node ID format bug**  
9 of 10 skipped edges failed because the LLM referenced nodes by file path (e.g. `specs/oz-project-specification.md`) instead of the graph's actual node IDs. Specs are indexed as section-level nodes, not file-level — so the file path doesn't exist as a node ID. This will affect `implements` edges too if not fixed. **S4 must explicitly include the list of valid concept node IDs in the prompt** so the LLM can reference them correctly rather than guessing.

One additional skipped edge had unknown type `agent_owns_concept` — the LLM invented an edge type that doesn't exist in the schema. The prompt should list valid edge kinds explicitly.

---

## Out of Scope (fast-follow)

- **COV006**: Package with no `implements` edges (too noisy without package exclusion list — deferred)
- **Incremental enrichment**: Only re-enrich changed packages (deferred to sprint 2)
- **`oz context review` UX improvements**: Showing concept edges with context for approval (deferred)
- **Multi-language support**: Go only in V1

---

## Definition of Done

- [ ] All 10 stories merged to main
- [ ] `oz context build && oz context enrich && oz audit` runs cleanly on oz's own codebase
- [ ] At least one `implements` edge exists per internal package
- [ ] COV005 fires for at least one concept
- [ ] DRIFT003 is `info` severity
- [ ] No unreviewed `implements` edges appear in `oz context query` results
- [ ] E1 decision recorded as ADR in `specs/decisions/`
- [ ] `notes/` artifacts (PRD, pre-mortem, sprint plan) promoted to appropriate tier or closed out
