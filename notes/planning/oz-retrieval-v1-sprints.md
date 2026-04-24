# oz retrieval V1 — Sprint Plan

**PRD**: notes/planning/oz-retrieval-v1-prd.md
**Pre-mortem**: notes/planning/oz-retrieval-v1-premortem.md
**ADR**: specs/decisions/0004-context-retrieval-ranking.md
**Spec**: specs/routing-packet.md
**Test framework**: docs/test-framework.md
**Format**: 1-week sprints, solo developer

Pre-mortem go/no-go checks before Sprint 2:
- [ ] T-01: `04_retrieval` golden suite fixtures drafted (Sprint 1 output)
- [ ] T-02: Tuning grid + metric defined

---

## Sprint 0 — Planning (this sprint)

**Goal**: Lock the design and the planning docs so implementation has no open design questions.

Completed:
- [x] ADR-0004 written and committed (`specs/decisions/0004-context-retrieval-ranking.md`)
- [x] `specs/routing-packet.md` revised with new fields, selection semantics, and `[retrieval]` config sections
- [x] PRD written (`notes/planning/oz-retrieval-v1-prd.md`)
- [x] Pre-mortem written (`notes/planning/oz-retrieval-v1-premortem.md`)
- [x] Sprint plan written (this file)

**Definition of done**: All planning docs committed, `oz validate` passes, `oz audit` shows no new drift or orphan findings attributable to these docs.

---

## Sprint 1 — BM25 core refactor + retrieval golden suite

**Goal**: The BM25 helpers become generic and reusable. The `04_retrieval` golden suite exists and locks what V1 means by "retrieval works." Zero user-visible behaviour change in routing or retrieval.

Resolves pre-mortem T-01.

### Stories

| # | Story | AC |
|---|---|---|
| S1-01 | Extract generic BM25 core from `scorer.go` | New internal interface `FieldDoc` with `Fields() map[string][]string` (or equivalent). `normTF`, `avgFieldLengths`, `computeDF` accept `[]FieldDoc`. `ComputeBM25F` reimplemented in terms of the generic core. Unit tests for the generic helpers pass. |
| S1-02 | Routing behaviour unchanged | `TestRoutingAccuracy` passes all golden suites (`01_minimal`, `02_medium`, `03_large`) at their current floors. Agent confidence byte-identical on two test queries. |
| S1-03 | Draft `04_retrieval` fixture | New directory `code/oz/internal/query/testdata/golden/04_retrieval/`. `workspace.yaml` mirrors a realistic oz-like workspace with: 3 agents, ≥6 spec sections, ≥3 ADRs, ≥4 docs, ≥2 notes, ≥3 code packages with ≥8 code symbols. Uses the `testws` fixture loader. |
| S1-04 | Draft `04_retrieval` queries | `queries.yaml` with ≥8 cases covering: code-implementation, spec-intent, docs-only, notes-rationale, ambiguous, low-relevance (should produce near-empty retrieval), cross-tier (answer in both specs and notes, specs should win), code-symbol-only (answer only in `code_symbol` names). |
| S1-05 | Retrieval assertion matchers | `ExpectBlockInTopK(file, section, k)`, `ExpectCodeEntryPoint(symbol, k)`, `ExpectPackageInTopK(pkg, k)`, `ExpectRelevanceDescending()`. Added to `internal/testws/assertions.go`. |
| S1-06 | Test harness stub | `TestRetrievalAccuracy` added but skipped (`t.Skip("awaiting Sprint 2 implementation")`) — confirms suite loads and matchers compile. |

### Sprint risks
- Premature abstraction: the generic `FieldDoc` interface may not fit the retrieval corpus shape discovered in Sprint 2. Mitigation: keep the interface minimal (fields → token slices), resist adding methods speculatively.

### Definition of done
- `go test ./internal/query/...` passes with new generic core
- Routing unchanged (R-05 partial)
- `04_retrieval` fixtures committed, suite skipped but compiles

---

## Sprint 2 — Ranked `context_blocks`

**Goal**: R-01 ships. Blocks carry a `relevance` field, sort by `(relevance DESC, trust, file, section)`, respect `max_blocks` and `min_relevance`. Agent-affinity boost replaces the hard `reads`-edge gate for docs/snapshots.

Resolves pre-mortem T-02 (tuning step) and T-03 (latency check) and T-04 (affinity verification).

### Stories

| # | Story | AC |
|---|---|---|
| S2-01 | New `contextretrieval` package | `internal/query/contextretrieval/` with `Score(query []string, blocks []Block, cfg RetrievalConfig, winningAgent string) []ScoredBlock`. Uses Sprint-1 generic BM25 core. |
| S2-02 | Replace `BuildContextBlocks` | `contextbuilder.go`'s selection logic delegates to `contextretrieval.Score` + threshold + cap. Old unconditional-spec-inclusion behaviour removed. |
| S2-03 | Body-token loader + cache | On-demand loader reads `spec_section`/`doc`/`note` bodies from disk, tokenizes, returns `[]string`. Cache keyed on graph `content_hash`. Cache cleared on rebuild. |
| S2-04 | Scoring config wiring | `[retrieval]`, `[retrieval.bm25]`, `[retrieval.fields]` loaded in `ScoringConfig`. Defaults match PRD §6 and ADR-4. `oz context scoring describe` lists new keys. |
| S2-05 | Emit `relevance` on blocks | Context block JSON shape includes `relevance` (number). Omitted when retrieval skipped (no-clear-owner path). |
| S2-06 | Per-scope survivor constraint | If any block under a winning-agent scope path cleared the threshold, at least one survives truncation. Unit test with a scope-covering block that otherwise wouldn't make top-K. |
| S2-07 | Tuning pass | Grid over `(min_relevance ∈ {0.03, 0.05, 0.08}, trust_boost_notes ∈ {0.5, 0.6, 0.7}, agent_affinity ∈ {1.1, 1.2, 1.3})`. Pick config maximising top-K precision on `04_retrieval` without regressing `02_medium` accuracy. Record in ADR-4 consequences. |
| S2-08 | Latency benchmark | `BenchmarkQuery` against the oz workspace. p95 ≤ 2× pre-Sprint-1 baseline with cache warm. If not met, implement on-disk body sidecar. |
| S2-09 | Affinity verification | On `03_large`, log top-10 blocks for a query where the winning agent has a wide read-chain. Confirm query-relevant blocks beat read-chain blocks without strong query match. If not, reduce boost or gate on `BM25 > 0`. |
| S2-10 | Un-skip `TestRetrievalAccuracy` for R-01 assertions | Cases asserting only `context_blocks` top-K membership pass. Cases asserting `code_entry_points` / `implementing_packages` still skipped (Sprints 3–4). |

### Sprint risks
- The tuning grid over-fits to `04_retrieval`. Mitigation: require no regression on `02_medium` routing; tuning picks parameters that are Pareto-OK, not suite-optimal.
- Body loading I/O dominates cold-cache queries. If latency misses, escalate to S2-08 sidecar.

### Definition of done
- R-01 accepted (ranked blocks, `relevance` emitted, `max_blocks` / `min_relevance` defaults locked)
- R-04 partially accepted (notes flow through pipeline with `trust_boost_notes`; `include_notes=false` path still in Sprint 4)
- R-05 fully accepted (routing unchanged)
- R-06 partially accepted (config loaded; editor surface in Sprint 4)
- Latency p95 ≤ 2× baseline

---

## Sprint 3 — Code entry points + tightened `implementing_packages`

**Goal**: R-02 and R-03 ship. "How is X implemented?" returns symbols and a curated package list.

### Stories

| # | Story | AC |
|---|---|---|
| S3-01 | `code_entry_points` packet field | New field in `query.Result` and in MCP serialisation. Objects: `{file, symbol, kind, line, package, relevance}`. Absent when empty. |
| S3-02 | Code-symbol scoring | `code_symbol` blocks added to retrieval corpus with fields `{name, package, kind}` and `[retrieval.fields]` weights (`weight_title` on `name`, `weight_path` on `package`, `weight_kind` on `kind`). |
| S3-03 | Eligibility rule | Symbol eligible if: (a) its `file` is under a winning-agent scope path, OR (b) its containing `code_package` has a reviewed `implements` edge to a concept with `score ≥ retrieval.concept_min_relevance`. |
| S3-04 | Cap and rank | Eligible symbols sorted by `relevance`, truncated to `retrieval.max_code_entry_points` (default 5). |
| S3-05 | Rewrite `loadImplementingPackages` | Replace token-overlap binary match with: BM25 over concept `(name, description)` using `[retrieval.concepts]` weights. Filter by `concept_min_relevance`. Walk reviewed `implements` edges. Sort packages by max reaching concept score. Cap at `max_implementing_packages` (default 5). |
| S3-06 | Un-skip `04_retrieval` code-level cases | Cases asserting `code_entry_points` and `implementing_packages` top-K pass. Specifically: drift-detection query returns `drift.Run`/`drift.LoadSymbols`/`specscan.Scan` (or equivalents in fixture) in top-5; packages include only `audit/drift` (+ optionally `audit` parent), not unrelated packages. |
| S3-07 | Regression guard | Existing routing golden suites unaffected; `TestRoutingAccuracy` still green. |

### Sprint risks
- Reviewed-`implements` gate (ADR-0003) may exclude all symbols on workspaces where enrichment hasn't been run. Mitigation: the eligibility rule's (a) clause (agent scope) ensures code symbols surface even without a semantic overlay.

### Definition of done
- R-02 accepted
- R-03 accepted
- Success metric "code entry points surfaced" + "package precision" green on golden suite

---

## Sprint 4 — Notes, config surface, docs, ship

**Goal**: R-04 and R-06 fully shipped. Scoring surface is editable through `oz context scoring`. Docs reflect the new pipeline.

### Stories

| # | Story | AC |
|---|---|---|
| S4-01 | `include_notes` semantics | Default flips to `true`. `false` hard-excludes `note` nodes from corpus and adds `"notes/"` to `excluded`. Golden-suite notes-rationale case passes with default config. |
| S4-02 | `oz context scoring` knows `[retrieval]` keys | `list`, `describe`, `get`, `set`, `show`, `validate` all handle the new tables. Invalid values (e.g. `max_blocks = -1`) rejected by `set` and `validate`. |
| S4-03 | `--raw` debug envelope includes retrieval math | Per-block BM25, trust boost, affinity, final relevance. Snapshot test locks the shape. |
| S4-04 | `reason: "no_relevant_context"` | When routing succeeds but zero blocks clear `min_relevance`, packet includes `reason` and omits `context_blocks`. Golden-suite low-relevance case asserts this. |
| S4-05 | Update `docs/architecture.md` | Diagram and prose updated to show retrieval pipeline distinct from routing. ADR-0004 cross-linked. |
| S4-06 | Update `specs/oz-project-specification.md` | Retrieval section added or updated to reflect shipped contract. Pre-existing drift items in project spec untouched. |
| S4-07 | Final tuning re-run | Re-run Sprint-2 grid with all surfaces live (notes, code entry points, packages). Lock final defaults. Update ADR-4 consequences. |
| S4-08 | `oz audit` clean on all new docs | No new DRIFT001/DRIFT002 findings attributable to this work. |

**Done 2026-04-24 — S4-07 / S4-08 close-out:** Sprint-2 grid re-run via `TestRetrievalTuningGridS2` (all 27 candidates pass `02_medium` gate; block top-K on `04_retrieval` tied at 1.0; **locked default triplet** `min_relevance=0.05`, `trust_boost.notes=0.6`, `agent_affinity=1.2` asserted in-test). [ADR-0004](../../specs/decisions/0004-context-retrieval-ranking.md) set to **Accepted** with V1 tuning, **micro-benchmark** warm/cold numbers, audit notes, and how to re-check the PRD latency gate. `oz validate` ok; full-repo `oz audit` still reports catalogued DRIFT/orphans unrelated to retrieval V1 (see ADR).

### Sprint risks
- Doc drift: `docs/architecture.md` may reference current behaviour elsewhere. Grep for `context_blocks` across docs and fix any stale mentions in S4-05.

### Definition of done
- All R-01 through R-09 accepted
- Success metrics in PRD §3 all green on the golden suite
- `oz validate` passes
- `oz audit` shows zero new findings from V1 work
- ADR-0004 status flipped from Proposed to Accepted

---

## Exit criteria for V1

- Golden suite `04_retrieval` green across all cases
- All routing golden suites still green at their floors
- Latency p95 ≤ 2× baseline
- ADR-0004 Accepted
- `specs/routing-packet.md` reflects shipped behaviour
- `docs/architecture.md` reflects shipped pipeline

Not covered by V1 (tracked separately): parser-mangling-skills bug, review-state inconsistency in `semantic.json`, rules files as graph nodes. See PRD §9.
