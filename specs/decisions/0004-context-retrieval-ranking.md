# ADR-0004: Context Retrieval Ranking

Date: 2026-04-23
Status: Accepted
Accepted: 2026-04-24

## Context

`oz context query <task>` produces a routing packet with two conceptually
different jobs collapsed into one pipeline:

1. **Routing** — pick the agent that owns the task.
2. **Retrieval** — pick the context the agent (human or LLM) should actually
   read before acting.

The shipped pipeline does (1) well: BM25F scores agent documents, softmax
produces a confidence, and thresholds decide "no clear owner" vs. routed. The
contract for this is `specs/routing-packet.md` and the implementation lives in
`code/oz/internal/query/scorer.go`.

The shipped pipeline does (2) poorly. `BuildContextBlocks`
(`code/oz/internal/query/contextbuilder.go`) selects blocks with these rules:

- All `spec_section` and `decision` nodes are included unconditionally.
- `doc` and `context_snapshot` nodes are included only if the winning agent
  has a `reads` edge to them.
- `note` nodes are excluded unless `include_notes` is set.
- Blocks are sorted by `(trust, file, section)` — no relevance signal.

On a representative query ("how is drift detection implemented?") this
produced 48 blocks, of which ~40 were spec sections unrelated to drift (API
design, enrich pipeline, scoring, routing packet, etc.). The winning agent
was correct; the context delivered was a firehose.

Three additional gaps show up on the same query:

- **Code entry points are not surfaced.** `code_symbol` nodes exist in the
  graph but the routing packet has no field for them. An LLM asking "how is
  drift detection implemented" gets no pointer to `LoadSymbols` or the
  surrounding drift detector entry points.
- **`implementing_packages` is over-inclusive.** `loadImplementingPackages`
  in `code/oz/internal/query/engine.go` does a binary token-overlap check
  against concept name+description. Every concept that shares any token with
  the query contributes its packages. There is no ranking and no cap.
- **Notes are blanket-excluded.** The `notes/` tier contains design
  rationale that is often the most direct answer to "why" questions, but the
  current logic is all-or-nothing gated on `include_notes`.

The normative spec currently says:

> The BM25F corpus is agent-only; code nodes do not participate in routing.
> `context_blocks` always include all spec sections and decisions.

Both statements are correct for **routing** and both are wrong for
**retrieval**. The spec conflates the two.

## Decision

Separate retrieval from routing. Introduce a per-block relevance scorer that
reuses the BM25 math already in `scorer.go` but operates on a different
corpus and with different field weights.

### Pipeline shape

`runRouting` in `code/oz/internal/query/engine.go` is renamed conceptually to
"route + retrieve" and runs in this order:

1. Load graph, tokenize query, score agents, softmax, pick winner.
   (Unchanged.)
2. Build a **retrieval corpus** of candidate blocks from the graph.
3. Score each block against the query with BM25 + trust boost + agent
   affinity boost.
4. Rank, apply a min-relevance threshold, truncate to a max block count,
   return.

### Retrieval corpus

The corpus includes, by node type:

- `spec_section` — section heading + body tokens (body loaded on demand from
  the file).
- `decision` — title + body tokens.
- `doc` — section heading + body tokens.
- `context_snapshot` — file + body tokens.
- `note` — file + body tokens (always eligible; tier downweight handles
  trust).
- `code_package` — name + `doc_comment` tokens.
- `code_symbol` — name + containing package + `symbol_kind` tokens.

Every block carries its trust tier (specs, docs, context, notes) and, for
code nodes, a synthetic tier `code` that maps to `"medium"` trust in the
packet.

### Scoring

Per block, the retrieval score is:

```
score(block) =
    BM25(query_terms, block_fields)
  * trust_boost(block.tier)
  * agent_affinity(block, winning_agent)
```

- The block BM25 term uses the same `normTF` / IDF form as
  `ComputeBM25F`, refactored so the helpers (`normTF`, `avgFieldLengths`,
  `computeDF`) accept a generic field-doc shape instead of `AgentDoc`.
  Fields and their weights are configured in `[retrieval]` (see scoring
  config update).
- `trust_boost` is a multiplicative factor per tier. Defaults:
  `specs = 1.3`, `docs = 1.0`, `context = 1.0`, `code = 0.9`, `notes = 0.6`.
  A tier boost < 1.0 effectively downweights that tier without excluding
  it, which is the behavior we want for notes.
- `agent_affinity` is `1.2` when the block is connected to the winning
  agent via a `reads`, `owns`, or (through the semantic overlay) an
  `agent_owns_concept` → `implements` chain; `1.0` otherwise. This boost
  replaces the current hard gate on `reads` edges for docs and snapshots.

### Selection

After scoring:

1. Drop blocks with `score < retrieval.min_relevance`.
2. Sort by `(score DESC, trust_rank ASC, file ASC, section ASC)`.
3. Truncate to `retrieval.max_blocks`.
4. Ensure at least one block per declared `scope` path remains if any such
   block cleared the threshold — this preserves the current behavior of
   "the winning agent's scope gets represented" without unconditionally
   shipping all specs.

### Code entry points

Add a new `code_entry_points` field to the routing packet. It is populated
from the top-K `code_symbol` blocks (after scoring) that belong to the
winning agent's scope or to a `code_package` connected via a reviewed
`implements` edge to a concept that scored above threshold. Each entry is
`{file, symbol, kind, line, package}`. K defaults to `retrieval.max_code_entry_points = 5`.

### `implementing_packages` tightening

`loadImplementingPackages` is rewritten to:

1. Score each concept against the query using the same retrieval BM25 over
   `(concept.name, concept.description)` with `[retrieval.concepts]`
   weights.
2. Keep concepts with `score >= retrieval.concept_min_relevance`.
3. Walk reviewed `implements` edges from those concepts to packages.
4. Sort packages by the max concept score that reaches them, truncate to
   `retrieval.max_implementing_packages` (default `5`).

### Notes

Notes are dropped from the blanket `excluded` mechanism. They flow through
the same ranked pipeline and are surfaced when relevant; the `notes`
trust_boost of `0.6` keeps them out of top slots when higher-trust material
answers the query. `include_notes=false` now means "hard-exclude notes from
the corpus" rather than the current "include all" / "exclude all" toggle,
and defaults to `true` to match the tiered model.

### What does not change

- Agent routing (BM25F over agent docs, softmax, thresholds) is unchanged.
- The `no_clear_owner` path is unchanged.
- Graph schema is unchanged; no new node or edge types.
- Semantic overlay review gate (ADR-0003) is unchanged.

## Consequences

**Positive**

- Context packets become signal-dense. The firehose shrinks from ~48
  unranked blocks to a ranked top-K.
- Code entry points are first-class; "how is X implemented" returns
  symbols, not just specs.
- `implementing_packages` becomes a curated pointer list instead of a
  shotgun.
- Notes are accessible when they are the best answer, without polluting
  high-signal queries.
- The retrieval scorer shares BM25 helpers with the router, so the math
  stays in one place.

**Negative**

- Two scorers now share internals; refactoring `scorer.go` to expose a
  generic BM25 core is required before retrieval can be built.
- Body tokens for `spec_section` / `doc` / `note` are not in `graph.json`
  today. Retrieval must either load file contents on demand (I/O per
  query) or bake tokenized bodies into the graph (schema grows). The
  default choice below is on-demand with an in-process cache keyed on
  graph `content_hash`.
- The packet contract grows: new `relevance` field on context blocks,
  new `code_entry_points` field, new `[retrieval]` config section.
  Consumers that pin exact-field shape must be updated.
- Trust boost multipliers for tiers other than the tuned `notes` boost, and field
  weights, were chosen as reasonable starting points; further workspace-specific
  tuning may use `oz context scoring` and golden suites.

## V1 tuning (Sprint 2 grid, re-confirmed Sprint 4)

A fixed grid over `retrieval.min_relevance` ∈ `{0.03, 0.05, 0.08}`,
`retrieval.trust_boost.notes` ∈ `{0.5, 0.6, 0.7}`, and `retrieval.agent_affinity` ∈
`{1.1, 1.2, 1.3}` is implemented as `TestRetrievalTuningGridS2` in
`code/oz/internal/query/tuning_s2_test.go`. Each candidate is rejected if
`02_medium` routing accuracy drops below the default-config baseline; the remainder
is ranked by top-K `expect_blocks_in_topk` hit rate on `04_retrieval`, with a
tie-break toward the pre-ADR default triplet.

**Sprint 4 re-run (2026-04-24):** all 27 candidates passed the routing gate; block
top-K score was 1.0 for every cell on the then-current `04_retrieval` cases;
the selected winner is **`min_relevance=0.05`**, **`trust_boost.notes=0.6`**, **`agent_affinity=1.2`**
— matching `DefaultScoringConfig` in `internal/query/config.go` and
`[retrieval]` in generated/published `context/scoring.toml` guidance. The test
now asserts the grid winner matches this locked triplet (fail if the golden
suites or tie-break would pick something else without an intentional default change).

Full `04_retrieval` behaviour (including `code_entry_points`, `implementing_packages`,
`no_relevant_context`, notes) is enforced by `TestRetrievalAccuracy` and routing
harnesses at their configured floors.

**Latency (micro-benchmark, 2026-04-24):** Command: `go test -run=^$ -bench=BenchmarkQuery -benchmem -benchtime=2s -count=5 ./internal/query` from `code/oz` on **darwin/arm64 (Apple M1)**.

| Benchmark | ns/op (5 runs, observed range) | B/op (approx.) | allocs/op |
|-----------|---------------------------------|----------------|-----------|
| `BenchmarkQueryWarmCache` | 582k–600k | 198k | 1350 |
| `BenchmarkQueryColdCache` | 773k–827k | 247k | 1600 |

The harness builds a **small** synthetic workspace in memory (`benchmark_test.go`); it is not
the full oz monorepo graph. **Warm** keeps the retrieval body-token cache hot across
iterations; **cold** calls `query.ResetRetrievalBodyCacheForBenchmark()` each iteration.
On this run, **cold / warm ≈ 1.33×** wall time, which is a sanity check that on-demand body
reads dominate cold behaviour.

**PRD gate (p95 ≤ 2× pre–Sprint-1 baseline on the real workspace):** not automatically
enforced. Re-run the same `go test -bench=BenchmarkQuery…` at a baseline tag vs HEAD, or
time `oz context query` end-to-end on a fixed query set over `context/graph.json` for the
dogfood repo. The numbers above are a **regression canary** for the query engine hot path
with V1 retrieval enabled, not a production SLO.

**S4-08 (audit):** `oz validate` passes on the oz repo. `oz audit` may still
report pre-existing `DRIFT001`/`DRIFT002` and orphan/coverage items from catalogued
docs and spec wording; those are not introduced by retrieval V1 and are tracked
outside this ADR.

## Alternatives considered

**Keep trust-only sort, just cap at top-K**

Rejected. A cap without relevance scoring means "include the first N spec
sections alphabetically," which is worse than today's firehose because it
silently hides material rather than drowning in it.

**Pure embedding search over all nodes**

Rejected for V1. Requires an embedding model dependency, a vector index,
and a rebuild step. BM25 over the existing graph is sufficient for
token-heavy technical queries and stays zero-dependency. Embeddings may
come later as an additive signal.

**Inline file contents in `context_blocks`**

Rejected. The current contract deliberately does not inline bodies, and
inlining would explode packet size while duplicating what the caller can
read from disk. Relevance ranking solves the real problem (which blocks
to read) without changing what a block contains.

**Separate "retrieve" CLI command instead of expanding `query`**

Rejected. Callers already expect one call that returns "route + context";
splitting into two commands pushes orchestration onto every consumer and
breaks the MCP `query_graph` contract.

**Emit code symbols as additional `context_blocks` entries instead of a
new field**

Rejected. `context_blocks` are whole-file-or-section references with a
trust tier; code symbols are file+line+symbol tuples. Overloading one
shape for both hides the difference from consumers and makes sort order
ambiguous (rank by what? symbol line numbers?). A dedicated
`code_entry_points` field keeps each shape clean.

## Implementation note (tooling, routing)

Agent routing and BM25F parameters are already tunable via `context/scoring.toml` and the
`oz context scoring` command group (`list`, `describe`, `get`, `set`, `show`, `validate`).
When a `[retrieval]` section and retrieval-specific keys are added (per this ADR), extend the
same TOML contract and CLI metadata in `internal/query` so users do not fall back to
hand-editing for those knobs.
