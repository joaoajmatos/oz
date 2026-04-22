---
crystallize: spec
crystallize-title: "Pre-Mortem: oz crystallize"
---

# Pre-Mortem: oz crystallize

**Date**: 2026-04-22
**Status**: Draft

---

## Risk Summary
- **Tigers**: 7 (2 launch-blocking, 3 fast-follow, 2 track)
- **Paper Tigers**: 5
- **Elephants**: 2 (E1 resolved by design decision)

---

## Launch-Blocking Tigers

| # | Risk | Likelihood | Impact | Mitigation | Owner | Deadline |
|---|------|-----------|--------|-----------|-------|----------|
| T1 | **LLM classifier accuracy below threshold in practice** — Few-shot context from the workspace's own artifacts helps, but the prompt may not generalise to very short, very informal, or highly mixed notes. If users get misclassified promotions they lose trust and stop using the command. | Medium | Critical | Before shipping, validate against ≥30 notes from ≥3 workspaces other than oz's own. If accuracy < 85%, improve the prompt (add more examples, tighten the type descriptions) before iterating on the heuristic fallback. Target: ≥85% external, ≥90% oz's own. | oz-coding | Before V1 release |
| T2 | **--accept-all promotes misclassified content silently** — `--accept-all` in CI means the classifier's errors become canonical without any human review gate. A note about a decision that gets classified as `guide` will be written to `docs/guides/` with no warning. | Medium | Critical | `--accept-all` only auto-accepts *high-confidence* results (gap ≥ 4, score ≥ 6). Medium-confidence files must be skipped unless `--force` is also passed. Print a mandatory summary of skipped and low-confidence files at the top of every `--accept-all` run. | oz-coding | Before V1 release |

---

## Fast-Follow Tigers

| # | Risk | Likelihood | Impact | Planned Response | Owner |
|---|------|-----------|--------|-----------------|-------|
| T3 | **Review fatigue kills adoption for large backlogs** — A user with 40 notes who sees 40 individual diffs to review will quit after 5. Per-file interactive review works well for 7 files; it becomes a blocker at scale. | High | High | Add a batch summary view before entering per-file review: show all classifications at once, let the user select which ones to review vs. skip. Consider `--interactive=summary` mode (default for >15 files) that groups by type and allows bulk-accept before drilling into individual diffs. | oz-coding | Sprint 2 |
| T4 | **Silent skips erode trust** — Files the LLM returns `unknown` or `low` confidence for are silently skipped with a single warning line. Users don't know why. | Medium | High | For each skipped file, print the LLM's `reason` field and returned confidence. Add a `--verbose` flag. | oz-coding | Sprint 2 |
| T5 | **API key not configured in CI / offline environments** — If `OPENROUTER_API_KEY` is absent and the user hasn't passed `--no-enrich`, the command should not hard-fail. | Medium | High | Detect missing key at startup. If absent: print a clear warning, auto-fall back to heuristic classifier, proceed. Never hard-fail on a missing key. | oz-coding | Sprint 1 |

---

## Track Tigers

**TT1 — open-items.md append grows unbounded**

If a workspace accumulates many open-item type notes, `docs/open-items.md` becomes large and undifferentiated.
Trigger: `docs/open-items.md` exceeds 200 lines after a single crystallize run.
Mitigation when triggered: implement directory mode (`docs/open-items/<slug>.md`). Flag as V2 now so the V1 API design doesn't preclude it.

**TT2 — Crystallize log diverges from reality on interrupted runs**

If a process is killed between the atomic write succeeding and the log append, the file exists but isn't logged.
Trigger: any user reports a missing log entry.
Mitigation: reconcile log on startup by scanning for promoted files not in the log.

---

## Paper Tigers

**P1 — Atomic write failures**
The temp-file + rename pattern is bulletproof on local filesystems. The only realistic failure is a full disk, which surfaces immediately as an OS error. Not a real risk.

**P2 — ADR number collisions in team workflows**
Vanishingly rare for a CLI tool used by small teams. Collision would be caught immediately by git. Not worth adding a locking mechanism for V1.

**P3 — --no-enrich flag or LLM failure leaves users with heuristic**
The heuristic fallback is a deliberate, tested code path. If the LLM call fails or the key is absent, the command degrades gracefully rather than erroring. Users get slightly worse classification, not a broken command.

**P4 — Frontmatter tag (crystallize:) won't get adopted**
The tag is an escape hatch for power users and CI authors, not the primary path. The heuristic classifier is primary. The tag costs nothing to implement.

---

## Elephants in the Room

**E1 — RESOLVED: LLM is the primary classifier**

The original design used a keyword-based heuristic as primary with LLM as opt-in (`--enrich`). This was flagged as a risk because LLMs are genuinely better at understanding semantic intent — a note saying "I think we should go with BM25 for now, here's why" would never match "we decided" but an LLM reads it correctly as an ADR.

**Decision**: OpenRouter is now the primary classifier, using workspace-aware few-shot context (existing ADRs, specs, and guides as examples). The heuristic becomes the fallback for offline / no-key environments. The `--enrich` flag is replaced by `--no-enrich`. Record this as an ADR in Sprint 1.

**E2 — notes/ may never contain content worth promoting in most workspaces**

The value of `oz crystallize` depends on teams actually accumulating meaningful content in `notes/`. In many real workflows, notes are throwaway — spike outputs, meeting scratchpads, temporary brainstorming. If the median workspace has zero promotable notes after 3 months of use, `oz crystallize` is a command nobody runs.

The oz `notes/` tier exists because it's clean architecture, not because there's evidence teams use it this way. Crystallize may be solving a problem the current user base doesn't have yet.

*Conversation starter*: "Before building this, can we look at 5 real oz workspaces and count how many notes we'd actually want to promote? If the answer is zero or one, what does that mean for prioritisation?"

**E3 — Crystallize may normalise writing worse first drafts**

By making promotion easy, there's a risk of normalising a workflow where everything goes through `notes/` first — even content that should have been a spec or ADR from the start. The command could inadvertently teach users to defer quality because "crystallize will clean it up."

*Conversation starter*: "Are we making it easier to do the right thing, or making it easier to defer doing the right thing?"

---

## Go / No-Go Checklist

- [ ] T1 mitigated: classifier accuracy validated on ≥30 notes from multiple workspaces
- [ ] T2 mitigated: `--accept-all` only auto-accepts high-confidence results; medium-confidence skipped without `--force`
- [ ] T3 planned: batch summary view for large backlogs on the fast-follow roadmap
- [ ] T4 planned: verbose skip explanations on the fast-follow roadmap
- [ ] TT1 logged as V2 open item: `docs/open-items/` directory mode
- [ ] Zero partial-state writes verified by kill-mid-write test
- [ ] Rollback path documented in `--help` output and receipt
- [ ] E1 decision recorded as ADR: LLM primary, heuristic fallback, cache for determinism
