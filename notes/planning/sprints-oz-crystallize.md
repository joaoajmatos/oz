---
crystallize: spec
crystallize-title: "Sprint Plan: oz crystallize"
---

# Sprint Plan: oz crystallize

**Team**: João Matos (solo)
**Sprint length**: 1 week
**Velocity basis**: Solo Go developer, ~25 effective hours/week after 15% buffer
**Point scale**: 1 pt ≈ 1.5–2 hours focused work
**Capacity per sprint**: ~15 pts (25h / ~1.7h avg)
**Sprints planned**: 4 (4 weeks to V1 ship)

**Pre-mortem risks incorporated**:
- T1 (launch-blocker): LLM classifier accuracy validated on external corpus in Sprint 4 — findings fed back to prompt tuning before ship
- T2 (launch-blocker): Confidence thresholds for `--accept-all` in Sprint 2, before the flag wires up in Sprint 3
- T3 (fast-follow): Batch summary view pulled into Sprint 3 (V1, not post-launch)
- T4 (fast-follow): Verbose skip explanations (using LLM `reason` field) pulled into Sprint 3 (V1, not post-launch)
- T5 (fast-follow): Graceful API key fallback in Sprint 1 (auto-fall back to heuristic if key absent)
- E1 (elephant): **Resolved** — LLM is primary classifier, heuristic is fallback. ADR recorded in Sprint 1.

---

## Sprint 1 — Classifier Foundation (LLM + Heuristic Fallback)

**Goal**: LLM classifier using OpenRouter is working end-to-end with workspace-aware context. Heuristic fallback exists. Cache package is implemented. Graceful API-key fallback works. ADR recorded.

**Duration**: Week 1
**Capacity**: 15 pts
**Committed**: 14 pts

| # | Story | Pts | Notes |
|---|-------|-----|-------|
| S1.1 | **ADR: LLM primary, heuristic fallback** — Write and commit `specs/decisions/NNNN-crystallize-classifier-approach.md`. Record: LLM is primary via OpenRouter, heuristic is the fallback, cache resolves non-determinism. | 1 | Blocks all implementation. Do this first. |
| S1.2 | **Package skeleton + exported types** — `internal/crystallize/classifier/` with `Classify(path string) (Classification, error)`, types `ArtifactType`, `Classification` (`Type`, `Confidence`, `Title`, `Reason`). | 1 | Sets the API surface. Both LLM and heuristic return the same `Classification` type. |
| S1.3 | **LLM classifier** — Build the workspace-aware prompt: source-of-truth hierarchy, type→target-path table, few-shot examples (first 30 lines of one real ADR, spec, guide from the workspace). Call `internal/openrouter/` client. Parse JSON response into `Classification`. | 3 | Core of Sprint 1. Prompt quality determines accuracy. |
| S1.4 | **Cache package** — `internal/crystallize/cache/`: read/write `.oz/crystallize-cache.json`. Key: `sha256(path + content + model)`. On cache hit: return cached `Classification` without LLM call. Invalidated automatically when content changes. | 2 | Resolves LLM non-determinism for `--dry-run` in CI. |
| S1.5 | **Graceful API-key fallback** (T5) — At startup, detect if `OPENROUTER_API_KEY` is absent. If so: print warning, set `useHeuristic = true`, proceed. Also: if LLM call errors, fall back to heuristic silently (log to stderr with `--verbose`). | 1 | T5 mitigation. Never hard-fail on missing key. |
| S1.6 | **Heuristic fallback** — `internal/crystallize/classifier/heuristic/`: keyword + structure signal scoring for all 5 types. Scoring: strong×3, structure×2, supporting×1, anti−2. Thresholds: score < 4 → unknown, gap < 2 → ambiguous. | 3 | Smaller than originally planned — a fallback, not the primary path. |
| S1.7 | **Frontmatter tag override** — Parse `crystallize:` and `crystallize-title:` from YAML frontmatter. Tag overrides both LLM and heuristic. | 1 | Simple; enables oz's own notes/ to self-label. |
| S1.8 | **Unit tests** — LLM classifier tested with a mock OpenRouter client (golden JSON responses). Cache: hit, miss, invalidation on content change. Heuristic: table-driven, 3 positive + 1 negative + 1 ambiguous per type. | 2 | |

**Buffer**: 1 pt

**Sprint 1 risks**:
- Prompt engineering may need several iterations before LLM returns well-formed JSON consistently → build with strict JSON schema and retry on parse error (max 2 retries)
- ADR decision (S1.1) should be quick — the decision is already made; this is just recording it

**Definition of done**:
- `go test ./internal/crystallize/...` passes
- `Classify()` returns correct type for oz's own `notes/` files (manual spot check)
- ADR committed to `specs/decisions/`
- Cache read/write/invalidation tests all pass

---

## Sprint 2 — Atomic Promotion + Log

**Goal**: Files can be promoted atomically to all 5 target types. ADR numbering is automatic. The crystallize log is append-only and correct. Kill-mid-write test passes. Confidence thresholds for `--accept-all` are implemented (T2 blocker resolved).

**Duration**: Week 2
**Capacity**: 15 pts
**Committed**: 13 pts

| # | Story | Pts | Notes |
|---|-------|-----|-------|
| S2.1 | **Confidence threshold model** — Define high/medium/low confidence from Sprint 1's `Classification`. High = score ≥ 6 AND gap ≥ 4. Medium = score ≥ 4 AND gap ≥ 2. Low = everything else. Export `IsAutoAcceptable(c Classification) bool`. | 2 | T2 launch-blocker mitigation. Must be done before S3.5 wires `--accept-all`. |
| S2.2 | **Atomic write** — `internal/crystallize/promote/` package. Write to `<target>.tmp`, then `os.Rename` on success. Return error (never partial write) on any failure. | 2 | Core correctness guarantee. |
| S2.3 | **ADR auto-numbering** — Scan `specs/decisions/` for highest `NNNN-*.md`, increment by 1. Pad to 4 digits. Deterministic on repeated calls (same input → same number). | 1 | |
| S2.4 | **ADR template injection** — Prepend frontmatter (`status: accepted`, `date: <today>`) and H1 title (`# ADR-NNNN: <title>`) to promoted ADR content. | 1 | Template is hardcoded (no config). |
| S2.5 | **open-item append mode** — Append promoted content as a new `## <title>` section to `docs/open-items.md`. If file doesn't exist, create it with a standard header. | 1 | |
| S2.6 | **Collision guard** — If target path already exists: compute diff, return `ErrCollision` with the diff. Caller decides whether to prompt or fail. Never silent overwrite. | 1 | |
| S2.7 | **internal/crystallize/log/** — Append-only log at `context/crystallize.log`. Format: `<RFC3339>\t<source>\t→\t<target>\t[<type>]`. Create file if absent. | 1 | |
| S2.8 | **Kill-mid-write atomicity test** — Start a promotion, send SIGKILL after `.tmp` is written but before rename, verify `.tmp` is cleaned up and target does not exist. | 2 | Tests the core correctness guarantee. Use `exec.Command` subprocess approach. |
| S2.9 | **Unit tests for promote package** — Table-driven: all 5 types, collision guard, log append, ADR numbering with gaps in existing sequence. | 2 | |

**Buffer**: 2 pts

**Sprint 2 risks**:
- `os.Rename` is not atomic across filesystems (e.g., if tmp and target are on different mounts) → document this limitation; acceptable for local workspace use
- Kill-mid-write test is tricky to write reliably → timebox 3 hours; if flaky, replace with a mock that injects failure after write but before rename

**Definition of done**:
- `go test ./internal/crystallize/...` passes
- Kill-mid-write test passes reliably (run 5 times)
- `IsAutoAcceptable` exported and tested

---

## Sprint 3 — Review TUI, CLI Wiring, Audit Delta

**Goal**: The full `oz crystallize` command is usable end-to-end. Interactive review works. All flags are wired. T3 (batch summary) and T4 (verbose skips) are in V1. `--dry-run` and `--accept-all` (with confidence gate) both work correctly.

**Duration**: Week 3
**Capacity**: 15 pts
**Committed**: 14 pts

| # | Story | Pts | Notes |
|---|-------|-----|-------|
| S3.1 | **Per-file diff review TUI** — `internal/crystallize/review/` package. Show unified diff of source note vs. proposed artifact. Actions: (a)ccept, (e)dit, (s)kip, (q)uit. Quit = clean exit, no partial state. Use `charmbracelet/huh` (existing dep). | 3 | Mirrors `oz context review` UX. |
| S3.2 | **Batch summary mode** (T3) — If file count > 15, show a table of all classifications first (file, type, confidence). User selects which to review individually vs. bulk-accept/skip by type. Default for large backlogs. | 2 | T3 fast-follow pulled into V1 per pre-mortem. |
| S3.3 | **Verbose skip explanations** (T4) — For each skipped file (unknown or low-confidence), print: top 2 candidate types with scores, reason for skip. Gated on `--verbose` flag. | 1 | T4 fast-follow pulled into V1 per pre-mortem. |
| S3.4 | **cmd/crystallize.go** — Cobra command. Orchestrates: walk notes/, classify, filter by --topic, review, promote, log, audit delta. Wire all packages. | 2 | |
| S3.5 | **--dry-run flag** — Skip all writes. Print classification table and proposed target paths. Show diffs without prompting for action. Exit 0. | 1 | |
| S3.6 | **--topic flag** — Filter notes to those whose content contains the topic string (case-insensitive). Applied before review loop. | 1 | |
| S3.7 | **--accept-all flag** — Auto-accept `IsAutoAcceptable` results. Skip review for medium/low confidence. Print mandatory summary of skipped files with their confidence levels. Require `--force` to override confidence gate. | 1 | T2 mitigation wired here. |
| S3.8 | **--no-enrich / --no-cache flags** — Wire `--no-enrich` (use heuristic only) and `--no-cache` (force fresh LLM call). Both flags are no-ops on the promotion path; they only affect classifier routing. | 1 | Straightforward flag wiring. |
| S3.9 | **Audit delta report** — After all promotions, run orphans + coverage + staleness checks. Print before/after counts: `Before: 4 errors → After: 1 error (−3)`. Reuse existing `audit.RunAll`. | 1 | |
| S3.10 | **Promotion receipt + undo hint** — After completion, print a receipt table (source → target, type) and an undo hint: `To undo: git checkout -- notes/ && git rm <targets>`. | 1 | |

**Buffer**: 1 pt

**Sprint 3 risks**:
- Diff TUI is the most open-ended story — charmbracelet/huh may not have a clean diff view primitive → fallback: print raw unified diff via stdlib, use huh only for the action prompt
- --enrich integration depends on semantic.json schema stability from oz context enrich → read schema before implementing; if unstable, stub out and skip

**Definition of done**:
- `oz crystallize --dry-run` runs on oz's own workspace without errors
- `oz crystallize --accept-all` auto-accepts only high-confidence results
- All flags produce correct output
- `go test ./cmd/... ./internal/crystallize/...` passes

---

## Sprint 4 — Validation, Dogfood, Ship

**Goal**: Classifier accuracy validated on external corpus (T1 resolved). oz's own notes/ backlog crystallized. All DoD criteria met. V1 shipped.

**Duration**: Week 4
**Capacity**: 15 pts
**Committed**: 12 pts

| # | Story | Pts | Notes |
|---|-------|-----|-------|
| S4.1 | **External corpus collection** (T1) — Collect ≥30 notes from ≥3 workspaces other than oz's own. Label ground truth manually. Record in `notes/planning/classifier-accuracy-external.md`. | 2 | This is the T1 launch-blocker validation. Do this first. |
| S4.2 | **Accuracy measurement + prompt tuning** — Run LLM classifier against external corpus. If accuracy < 85%, improve the prompt (add more few-shot examples, tighten type descriptions, adjust JSON schema). Re-run until ≥85% or document the gap in the ADR. | 3 | Prompt iteration is faster than signal table rework. Budget 3 pts for tuning cycles. |
| S4.3 | **Integration tests** — Full end-to-end tests using `testws` fixture package. Test: dry-run on fixture workspace, --accept-all on high-confidence fixture, collision guard, atomicity on fixture write. | 2 | |
| S4.4 | **Dogfood: crystallize oz's own notes/** — Run `oz crystallize` on the oz repo's own `notes/planning/`. Promote the PRD, pre-mortem, and sprint plan to their canonical locations. Fix any issues found. | 1 | Self-dogfooding. The command should be able to promote the very files that specified it. |
| S4.5 | **oz-coding agent read-chain update** — Add `internal/crystallize/`, `cmd/crystallize.go` to `agents/oz-coding/AGENT.md` read-chain. | 1 | Required for oz validate to pass (coverage check). |
| S4.6 | **oz validate exit 0** — Run `oz validate` and `oz audit`. Confirm 0 validation errors. Confirm audit error count is ≤ pre-crystallize count (should be lower). | 1 | DoD gate. |
| S4.7 | **Help text and --help polish** — Ensure `oz crystallize --help` describes all flags, examples for common workflows, and the note about confidence gates on `--accept-all`. | 1 | |
| S4.8 | **Final DoD checklist** — Walk through all items in the PRD definition of done and pre-mortem Go/No-Go checklist. Record pass/fail. Ship if all pass. | 1 | |

**Buffer**: 3 pts (larger buffer because accuracy fixes in S4.2 may spill)

**Sprint 4 risks**:
- External corpus accuracy < 85% requires significant signal table rework → if this happens, extend Sprint 4 by 1 week rather than ship below threshold (T1 is launch-blocking)
- Dogfood run surfaces edge cases in review TUI or atomic write → 3pt buffer absorbs up to 2 unexpected fixes

**Definition of done** (full V1 DoD):
- [ ] `go test ./...` passes, zero failures
- [ ] `oz crystallize --dry-run` correctly classifies ≥85% of external corpus (30+ notes, 3+ workspaces)
- [ ] `oz crystallize --dry-run` correctly classifies ≥90% of oz's own notes/
- [ ] Kill-mid-write atomicity test passes 5/5
- [ ] `--accept-all` only auto-accepts high-confidence results; verified by test
- [ ] `oz validate` exits 0 on oz repo
- [ ] `oz audit` error count on oz repo is lower after dogfood run
- [ ] crystallize added to oz-coding agent read-chain
- [ ] ADR from S1.1 committed (heuristic vs. LLM decision recorded)
- [ ] Pre-mortem Go/No-Go checklist: all items checked

---

## Summary

| Sprint | Theme | Deliverable | Key Risk Addressed |
|--------|-------|-------------|-------------------|
| Sprint 1 | Classifier | LLM classifier + heuristic fallback + cache + ADR | E1 (resolved), T5 (key fallback) |
| Sprint 2 | Promotion | Atomic writes, ADR numbering, log, confidence thresholds | T2 (--accept-all safety) |
| Sprint 3 | UX + CLI | Review TUI, all flags, batch mode, audit delta | T3, T4 (review UX) |
| Sprint 4 | Validate + Ship | External corpus test + prompt tuning, dogfood, DoD gate | T1 (LLM accuracy) |

**Total**: 4 weeks. No scope cut needed if Sprint 4 accuracy is ≥85% on first measurement.
**Contingency**: If Sprint 4 accuracy requires signal rework, extend by 1 week. Do not ship below T1 threshold.
