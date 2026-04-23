# Pre-Mortem: oz Semantic Code Understanding

**Date**: 2026-04-23  
**Status**: Draft  
**PRD**: `notes/PRD-semantic-code-understanding.md`

---

### Risk Summary

- **Tigers**: 5 (2 launch-blocking, 2 fast-follow, 1 track)
- **Paper Tigers**: 3
- **Elephants**: 2

---

## Launch-Blocking Tigers

| # | Risk | Likelihood | Impact | Mitigation | Owner | Deadline |
|---|------|-----------|--------|-----------|-------|----------|
| T1 | **Concept graph is too sparse to map to** — enrichment produces concepts only from spec/agent text. If oz's concept graph has 5–10 high-level concepts, most packages will fail to map to anything meaningful. The LLM either forces bad mappings or returns empty edges, making the whole feature useless. | High | Fatal | Before extending enrichment, audit the *existing* concept graph quality. If fewer than ~15 concepts exist, run a spec-first enrichment pass to flesh them out. Add a pre-flight check: refuse to run code enrichment if concept graph has < N nodes. | João | Before sprint start |
| T2 | **LLM maps packages to wrong concepts with high confidence** — the model confidently assigns `code/oz/internal/query` to `concept:drift-detection` because both involve "checking things." Downstream queries and audit checks silently return wrong results. Users trust them. | Medium | Fatal | Add a mandatory spot-check step in `oz context review` that shows each `implements` edge with the package's actual exported symbols. Make `reviewed: false` edges invisible to query by default — only auditable. Never let unreviewed edges affect query results. | João | Before any query integration |

---

## Fast-Follow Tigers

| # | Risk | Likelihood | Impact | Planned Response | Owner |
|---|------|-----------|--------|-----------------|-------|
| T3 | **COV005/COV006 are as noisy as DRIFT003** — "package has no concept" fires for every internal helper, every `testutil`, every `cmd/` entry point. Users immediately learn to ignore audit output again. | High | High | Scope COV006 to non-trivial packages only: exclude `*_test`, `testutil`, `cmd/`, `internal/scaffold`. Add a config opt-in rather than opt-out. Ship COV005 (concept with no code) first — it's the more valuable direction. | João | Sprint 1 post-launch |
| T4 | **Enrichment latency kills the workflow** — if enriching a 20-package codebase takes 30+ seconds and costs real money per run, users run it once and never again. The semantic layer goes stale immediately. | Medium | High | Implement incremental enrichment: only re-enrich packages whose file hashes changed since last run. Cap total LLM calls per enrichment run. Log cost estimate before executing. | João | Sprint 1 post-launch |

---

## Track Tigers

| # | Risk | Trigger | Monitor |
|---|------|---------|---------|
| T5 | **Package granularity is wrong for some codebases** — very large packages (e.g. a single `internal/engine` with 40 files) produce one vague concept edge that means nothing. Very small packages (one-file helpers) produce concept edges that are obvious. | Triggered if any package has >15 exported symbols and its concept edge has confidence <0.7 | Track concept edge confidence distribution after first real enrichment run; consider file-level fallback for large packages |

---

## Paper Tigers

**PT1: "The LLM won't understand Go packages without reading source"**  
Feels scary but isn't. Package doc comments, exported symbol names, and the concept graph as context are enough for a capable LLM to make reasonable mappings. This is a low-complexity semantic task — packages are named things like `audit/drift` and `codeindex/goindexer`. The names carry intent. *Would become a real Tiger if packages were cryptically named or the codebase had no doc comments at all.*

**PT2: "Adding code to the enrichment prompt will blow the context window"**  
For a codebase the size of oz (~15 packages), a package name + exported symbol list is ~100 tokens per package. That's 1,500 tokens total — trivially small. Even at 50 packages it's fine. *Would become a Tiger if oz ever runs on monorepos with 500+ packages — needs a chunking strategy at that scale, but that's V3 territory.*

**PT3: "Changing DRIFT003 to info will break existing workflows"**  
Nobody has workflows built on DRIFT003. It was always noise. Demoting it to info is safe. If someone genuinely needs it, the flag `--include-symbols` can restore it. This is not a migration risk.

---

## Elephants in the Room

**E1: The human review step will never happen in practice**  
The design says `reviewed: false` edges are the default and a human approves them. But in a solo/small-team project, "I'll review it later" means never. If unreviewed edges are invisible to queries, the feature delivers no value until reviewed. If they're visible without review, the quality gate is meaningless.

*Suggested conversation*: "Do we actually want a review gate, or do we want confidence thresholds? Maybe `confidence > 0.85` auto-promotes to reviewed, and only low-confidence edges require human review. That's honest about how this will actually be used."

**E2: We're building a feature that makes oz dependent on LLM quality we don't control**  
Symbol collection is deterministic. Semantic enrichment is not. If OpenRouter changes models, if the model regresses, if prompts drift — the concept graph degrades silently. Nobody will notice until audit checks start producing garbage. The structural graph (graph.json) was trustworthy because it was deterministic. semantic.json is already non-deterministic; this feature doubles down on that.

*Suggested conversation*: "What's our strategy for detecting semantic layer rot? Should `oz audit` include a check that samples concept edges and validates them, or do we accept that enrichment output is best-effort and document it clearly as such?"

---

## Go/No-Go Checklist

- [ ] Concept graph has sufficient density (≥15 concepts) before code enrichment runs
- [ ] Unreviewed `implements` edges do NOT affect query results — only audit visibility
- [ ] `oz context review` shows enough context per edge for a human to approve/reject in <10s
- [ ] COV006 scoped to exclude test/helper packages before shipping
- [ ] Incremental enrichment plan exists (even if not built in V1)
- [ ] Cost/latency of one enrichment run measured on oz's own codebase before shipping
- [ ] Rollback: removing semantic.json restores full prior behavior with no graph breakage
- [ ] E1 (review gate design) decision made explicitly before implementation starts
