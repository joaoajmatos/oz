# Pre-Mortem: oz context V1

**Date**: 2026-04-19
**Status**: Draft
**PRD**: docs/oz-context-v1-prd.md

---

## Risk Summary

- **Tigers**: 6 (1 launch-blocking, 3 fast-follow, 2 track)
- **Paper Tigers**: 3
- **Elephants**: 3

---

## Launch-Blocking Tigers

| # | Risk | Likelihood | Impact | Mitigation | Owner | Deadline |
|---|------|-----------|--------|-----------|-------|----------|
| T-01 | **Query scoring algorithm is undefined.** The PRD says "tokenises the query against scope declarations and concept nodes, scores candidates by scope match + concept proximity" — but specifies no algorithm. TF-IDF? BM25? Keyword intersection? The 95% routing accuracy target depends entirely on this decision. It is a P0 requirement with zero implementation guidance. | High | Critical | Before a single line of query logic is written, define and document the scoring algorithm explicitly: token overlap metric, tie-breaking rules, how concept node proximity is weighted vs. direct scope keyword match, minimum score threshold for a valid result. Add this as a new section in the PRD before Phase 1 implementation begins. | oz-spec | Before Phase 1 sprint kick-off |

---

## Fast-Follow Tigers

| # | Risk | Likelihood | Impact | Planned Response | Owner |
|---|------|-----------|--------|-----------------|-------|
| T-02 | **Leiden community detection degenerates on small graphs.** Leiden finds communities by edge density and works well on graphs with hundreds of nodes. The oz workspace structural graph will have ~20–40 nodes at most (3 agents, ~10 spec sections, ~5 decisions, a handful of docs). On a small sparse graph, Leiden may produce degenerate output: one giant cluster, or every node its own cluster. The claim that "graph topology is the similarity signal" may simply not hold at oz workspace scale. | Medium | High | Before committing to Leiden, prototype it on oz's own workspace and measure community quality. If communities degenerate, fall back to simpler clustering (connected components + trust-tier grouping) which is deterministic and scale-invariant. Don't carry an algorithm that requires a minimum graph density oz will never reach. | oz-coding |
| T-03 | **`semantic.json` is not meaningfully human-reviewable.** The PRD's acceptance mechanism is "human reviews and commits." But raw JSON with potentially hundreds of concept nodes and typed edges is not legible in a `git diff`. In practice, developers will rubber-stamp it with `git add context/semantic.json`. If the curated overlay is the trust foundation of enriched routing, and it is not actually curated, the semantic layer is built on a false premise — and every routing result that depends on it is unverified. | High | Medium | Before Phase 2 ships, add `oz context review` — a purpose-built diff view that presents only new/changed nodes and edges in a readable format (table or markdown, not JSON), flags low-confidence INFERRED edges for explicit confirmation, and writes a `reviewed: true` marker back to the file. Committing without running review should produce a validation warning from `oz validate`. | oz-coding |
| T-04 | **MCP protocol conformance is non-trivial.** The PRD says "`oz context serve` wraps this in an MCP stdio server" as if it is a minor implementation detail. The MCP stdio protocol has specific framing requirements, capability negotiation, error response shapes, and tool schema conventions. A subtly off-spec implementation will silently fail to work with any MCP client. No conformance test or reference client is mentioned anywhere in the PRD. | Medium | High | Add an explicit acceptance criterion to C-04: the MCP server must be validated against at least one real MCP client (Claude Code's MCP runtime) as part of the Phase 2 definition of done. Write an integration test that spawns `oz context serve` as a subprocess and exercises all four tool calls over stdio. | oz-coding |

---

## Track Tigers

*Monitor post-Phase 1 — trigger conditions noted.*

**T-05 — Semantic overlay staleness**
If the workspace evolves (new agents added, spec sections rewritten) but no one reruns `oz context enrich`, the semantic overlay silently diverges from the structural graph. An LLM using stale concepts gets confidently wrong routing with no signal that something is wrong. The PRD has no freshness check, no content hash comparison, no staleness warning.

*Trigger*: If the oz workspace undergoes any structural change (new agent, spec rewrite) without a corresponding `oz context enrich` run, activate. Mitigation: on `oz context query`, compare a hash of `graph.json` against a hash stored inside `semantic.json` at generation time. If they differ, print a staleness warning. Add to Phase 2 scope.

**T-06 — Default model for enrich is deferred**
The PRD says the default model for `oz context enrich` is "TBD at implementation time." The choice of model affects semantic extraction quality, which directly affects the 95% routing accuracy target, and cost per enrichment run. Deferring this means the implementer picks it ad-hoc.

*Trigger*: When Phase 2 implementation begins. Mitigation: decide before Phase 2 kick-off — pick a specific OpenRouter model as the default and document it in the PRD. Recommend: a model with strong structured output support (JSON mode) and reasonable cost per 1K tokens.

---

## Paper Tigers

**P-01 — OpenRouter availability**
`oz context enrich` is the only network-dependent command in the binary. If OpenRouter is unavailable, only enrichment fails — structural graph, query, and MCP server are entirely unaffected. And enrichment is human-initiated, not part of any runtime path. This risk is low and well-contained by the architecture.

**P-02 — API cost of enrichment**
oz workspaces are small by design — the spec explicitly says "convention over configuration" and workspaces stay lean. A full enrich pass on oz's own workspace is unlikely to exceed a few cents on any capable model. Not a meaningful cost concern.

**P-03 — Markdown section anchors are fragile**
The concern is that duplicate section headers in a markdown file would break anchor resolution. In practice, oz workspace files are structured convention documents with named sections — unlikely to have collisions. If a collision is found, the parser detects it deterministically and can warn at build time. This is a well-bounded edge case, not a systemic risk.

---

## Elephants in the Room

**E-01 — The scoring algorithm is the thing nobody named**
The PRD has detailed JSON schemas, acceptance criteria for every edge case, a precise routing packet structure — but says nothing concrete about the algorithm that actually computes which agent owns a task. "Tokenises and scores" describes what the algorithm does at a high level but is not an algorithm. This is the most important design decision in the entire system, and it is currently a blank space dressed up as a solved problem. The team should name this explicitly and design it before any Phase 1 code is written.

*Conversation starter*: "Before we write any query logic — what is the exact scoring function? What inputs does it take, what does it return, and how do we test it in isolation?"

**E-02 — oz's own workspace is too small to validate routing accuracy**
The success metric is "95% accuracy on 20 representative queries against the oz workspace itself." But oz has only 3 agents. Any crude keyword match that works at all will probably hit 95% accuracy when there are only 3 possible answers. The real routing problem emerges at 10+ agents with overlapping scope declarations, competing spec ownership, and sparse concept graphs. Validating exclusively on oz produces false confidence that will not survive the first real adopter's workspace.

*Conversation starter*: "What's our plan for validating routing on a workspace we don't control? Do we need to create a synthetic multi-agent test workspace before we call Phase 1 done?"

**E-03 — "Commit as acceptance" is a process claim with no tooling support**
The PRD states that committing `semantic.json` is the acceptance mechanism for the semantic overlay. This is entirely process-dependent — there is no diff tooling, no pre-commit validation, no schema check, no reviewer guidance. The system trusts that developers will review JSON node-by-edge before committing. This is an optimistic assumption that will fail silently in practice. The semantic overlay will accumulate unreviewed LLM extractions and become a source of routing errors that are hard to trace back to their origin.

*Conversation starter*: "If a developer runs `oz context enrich` and immediately commits the output without reading it — which they will — what breaks, and how do we detect it?"

---

## Go / No-Go Checklist

### Before Phase 1 implementation begins
- [ ] T-01: Query scoring algorithm designed and documented in the PRD
- [ ] E-02: Plan for multi-agent validation workspace documented (synthetic test workspace or equivalent)

### Phase 1 definition of done
- [ ] Leiden prototype benchmarked on oz workspace; fallback defined if communities degenerate (T-02)
- [ ] `graph.json` schema published and frozen
- [ ] `oz audit` integration test passing against `graph.json`
- [ ] Staleness hash added to `graph.json` schema (for T-05 mitigation in Phase 2)

### Before Phase 2 implementation begins
- [ ] Default OpenRouter model selected and documented (T-06)
- [ ] `oz context review` workflow designed (T-03)
- [ ] MCP conformance test strategy documented (T-04)

### Phase 2 definition of done
- [ ] MCP server validated against Claude Code MCP runtime (T-04)
- [ ] `oz context review` command ships alongside `oz context enrich` (T-03)
- [ ] Staleness warning implemented and tested (T-05)
- [ ] `oz validate` warns if `semantic.json` exists but has unreviewed nodes (E-03)

---

*Address T-01 and E-02 before any sprint planning. Everything else is manageable within the phased delivery plan.*
