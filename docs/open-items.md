# Open Items

> Open questions, known issues, and pending decisions.
> Resolved items move to `specs/decisions/`.

## Open Questions

<!-- Questions that need answers before proceeding. -->

## Known Issues

### Shell compression layer not yet implemented (tracking item)

Specification and ADR are landed:

- `specs/oz-shell-compression-specification.md`
- `specs/decisions/0005-oz-shell-compression-architecture.md`

Implementation has not started yet. Risks to manage in rollout:

1. preserving full failure signal while aggressively reducing tokens
2. maintaining deterministic compact output across tool versions
3. keeping wrapper overhead within the v1 SLO targets

### oz audit — performance baseline (Sprint A6, 2026-04-20)

`go test ./internal/audit -bench=BenchmarkAuditAll` runs the full check bundle on a small
scaffolded workspace (orphans, coverage, staleness, drift). On Apple M1-class hardware this
has measured on the order of **~0.7 ms/op** — well under the PRD’s **< 1s** target for the
real oz repo after a fresh `oz context build`. AT-05 (slow audit) is not triggered.

`--include-tests` / `--include-docs` are wired on the parent `oz audit` command and passed
through `audit.Options` to drift (test symbols merged from `*_test.go` via `codeindex` only
when `--include-tests` is set).

### oz audit drift — accepted noise (Sprint A5 self-validation, 2026-04-20)

Running `oz audit drift` against this repo after a fresh `oz context build` produces 0 errors and 222 DRIFT003 warnings.

**DRIFT002 errors: 0.** One false positive (`OPENROUTER_API_KEY` — an env var name) was eliminated by narrowing the spec scanner regex to exclude underscores from identifier patterns (AT-01 mitigation). The canonical narrowing is documented in `internal/audit/drift/specscan/scan.go` (the `goIdentRe` comment).

**DRIFT003 warnings: 222.** All exported symbols in the codebase (`cmd`, `internal/*`) are technically "unmentioned" in specs because the specs describe behaviour and architecture, not every public API symbol. This is expected for a codebase-in-development. DRIFT003 is `warn` severity and does not trigger CI failure under the default `--exit-on=error` policy. Accepted as known noise.

| Code | Disposition |
|---|---|
| DRIFT002 (0 errors) | Clean. AT-01 resolved: narrowing applied, no false positives. |
| DRIFT003 (222 warns) | Accepted noise. Codebase is growing faster than spec coverage. Revisit at A6. |

### oz audit staleness — accepted noise (Sprint A3 self-validation, 2026-04-20)

Running `oz audit --only=orphans,coverage,staleness` after a fresh `oz context build` produces 1 warning and 1 info finding and 0 errors.

| Code | Finding | Disposition |
|---|---|---|
| COV002 | `code/oz/` is not owned by any agent | Carried over from A2. Accepted. |
| STALE004 | semantic.json not found | Expected. No enrichment has been run. Info-only. |

AT-02 (determinism): resolved. Determinism test (A3-05) passes 100 consecutive runs with byte-identical findings.

### oz audit orphans + coverage — accepted noise (Sprint A2 self-validation, 2026-04-20)

Running `oz audit --only=orphans,coverage` against this repo produces 3 warnings and 0 errors.
All are accepted noise per pre-mortem AT-03 (threshold: > 3 warnings triggers tuning).

| Code | Finding | Disposition |
|---|---|---|
| COV002 | `code/oz/` is not owned by any agent | Expected. The oz-coding agent has broad responsibilities but no explicit `code/oz/**` scope path. The workspace follows a convention-light ownership model (AE-02). No tuning needed. |
| ORPH002 | `notes/planning/oz-context-v1-prd.md` has no inbound references | Historical PRD, superseded by the shipped feature. Accepted noise. |
| ORPH002 | `notes/planning/oz-context-v1-sprints.md` has no inbound references | Historical sprint plan, not linked from live docs. Accepted noise. |

AT-03 is considered **not triggered** (3 warnings ≤ threshold of 3).

## Pending Decisions

### SHL-01 — `oz shell run` storage backend for tracking (v1)

`specs/oz-shell-compression-specification.md` requires local token/perf tracking with bounded
retention, but leaves backend choice open (`sqlite` vs structured file store).

**Current recommendation:** SQLite (queryable and robust), matching expected analytics needs.
**Status:** pending implementation choice during Phase 1.

### SHL-02 — Transparent interception default mode

`oz shell run` v1 includes optional transparent interception, but default behavior is undecided:

- suggest-only first
- auto-rewrite when hook is present

**Status:** pending before hook integrations ship.

## CCA-0 Token Budget Spike (CCA-0-06) — RESOLVED

Measured against this repo's `context/graph.json` (651 nodes, 522 edges) with
`scoring.toml` defaults (`max_blocks = 12`).

### Prompt component breakdown

| Component | Chars | Est. tokens |
|-----------|-------|-------------|
| Instruction template | 203 | ~50 |
| Retrieval slice (12 blocks) | 1,000 | ~250 |
| Allowlist section (72 IDs) | 4,042 | ~1,010 |
| Existing reviewed concepts | 332 | ~83 |
| Schema + rules tail | 743 | ~185 |
| **Static subtotal** | **6,320** | **~1,580** |
| + seed text (user, ~200 words) | — | ~270 |
| + response (1 concept) | — | ~150 |
| **Grand total** | — | **~2,000** |

### Allowlist breakdown (current repo)

| Type | Count |
|------|-------|
| `agent:` | 4 |
| `spec_section:` + `decision:` | 34 |
| `code_package:` | 34 |
| **Total** | **72** |

### Decisions for CCA-1

1. **No trimming needed now.** ~2k tokens is comfortably under the 8k target. The
   full allowlist (all `agent:`, `spec_section:`, `decision:`, `code_package:` IDs) ships
   in V1 — same set as `enrich` uses today.

2. **Default `--retrieval-k = 5`.** Half of `max_blocks` (12). Keeps the retrieval
   slice at ~100 tokens for a focused concept proposal, while `--retrieval-k 12` stays
   available for queries that benefit from broader context.

3. **Growth trigger for trimming:** `code_package:` IDs are the primary growth vector
   (~1 per Go package). At ~150 packages the allowlist hits ~3k tokens (still fine).
   Add allowlist trimming to concept-relevant packages only when the grand total
   approaches 6k tokens (roughly 3× current repo size). Track in the CCA-0-06 item.

4. **Retrieval blocks as text labels only** (file + section + trust tier, no body
   content). Including body content would add ~500–2000 tokens per block — defer to
   a flag if users request richer grounding.

## Known Limitations: oz context concept add

### CCA-L-01 — Retrieval packet partially used in proposal prompt

`RetrievalForProposal` returns `context_blocks`, `relevant_concepts`, `implementing_packages`,
and `code_entry_points`. The proposal prompt currently uses only `context_blocks` (top-k
file+section+trust labels). `relevant_concepts` and `implementing_packages` are not yet fed
into the prompt.

**Deferral rationale:** context blocks alone provide sufficient grounding for V1. Extending
`ProposeOptions` + `BuildProposalPrompt` to include concept/package context is tracked here.
Add when users report that concept proposals miss package or concept relationships.

### CCA-L-02 — `--from` anchors do not steer retrieval

`--from <file>` paths appear in the prompt as anchors but do not change the retrieval query
or boost candidates in `RetrievalForProposal`. Retrieval is driven by `--name` + `--seed` only.

**Deferral rationale:** grounding via retrieved blocks already covers the common case. Boosting
specific files in the retrieval candidate set is a retrieval-tuning concern; track for a
follow-up when the `--from` signal proves useful enough to warrant the added complexity.
