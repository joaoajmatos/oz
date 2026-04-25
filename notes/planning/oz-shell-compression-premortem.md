# oz Shell Compression Layer — Pre-mortem

**Author**: oz-spec  
**Date**: 2026-04-25  
**Status**: Draft  
**PRD**: [oz-shell-compression-prd.md](oz-shell-compression-prd.md)  
**Sprint plan**: [oz-shell-compression-sprints.md](oz-shell-compression-sprints.md)

---

## Failure scenario (14 days after launch)

The feature launches, but adoption is low and trust drops quickly.  
Teams report that compact output occasionally hides important debugging details, transparent mode behavior feels unpredictable, and performance gains are inconsistent.  
As a result, many users disable the layer and return to raw shell outputs.

---

## Tigers (Real Risks)

### T1 — Failure signal loss in compact mode

If filtering hides actionable error context, users cannot safely debug.

- **Why real:** this is the highest-risk failure mode for any compression layer.
- **Urgency:** Launch-Blocking

### T2 — Exit-code propagation regression

If wrapped commands return wrong status, CI and automation become unreliable.

- **Why real:** one incorrect propagation invalidates trust for engineering workflows.
- **Urgency:** Launch-Blocking

### T3 — Non-deterministic output snapshots

If output changes unpredictably for same input, users and tests lose confidence.

- **Why real:** deterministic contracts are core to `oz` behavior elsewhere.
- **Urgency:** Launch-Blocking

### T4 — Hook rewrite causes surprising behavior

Transparent interception may rewrite commands users expected to run raw.

- **Why real:** interception creates perception risk even if technically correct.
- **Urgency:** Fast-Follow

### T5 — Performance overhead exceeds tolerance

If wrapper adds visible latency, users avoid it.

- **Why real:** shell interactions are frequent and latency compounds.
- **Urgency:** Fast-Follow

### T6 — Filter drift across tool versions

Parser assumptions may break with output format changes.

- **Why real:** output formats evolve in `git`, test runners, and search tools.
- **Urgency:** Track

### T7 — `oz shell gain` reports misleading numbers

If aggregation or retention logic is wrong, users lose trust in the analytics and in rollout decisions.

- **Why real:** v1 now includes `oz shell gain`, so metric correctness is part of the product promise.
- **Urgency:** Launch-Blocking

---

## Paper Tigers (Overblown Concerns)

### P1 — “Any compression is unsafe by definition”

Not true if safety invariants are enforced (exit code, failure-line preservation, raw fallback).

### P2 — “Must support 100 commands before launch”

Not required. V1 can deliver strong value with 4 high-frequency command families.

### P3 — “Need cloud telemetry for useful metrics”

Not required for v1; local tracking is sufficient to validate adoption and savings.

---

## Elephants (Unspoken Worries)

### E1 — Team disagreement on transparent default

Even with opt-in, there may be hidden disagreement on suggest-first vs auto-rewrite-first.

**Investigation:** lock default in docs and acceptance tests before implementation starts.

### E2 — Scope creep into source-code compression

There is a subtle risk of extending shell compression into lossy code-view paths too early.

**Investigation:** enforce boundary: shell-output compression only for v1.

### E3 — Maintenance ownership for filter growth

Long-term support for many command families can become fragmented.

**Investigation:** define ownership and contribution checklist before adding non-MVP families.

---

## Action Plans for Launch-Blocking Tigers

| Risk | Mitigation | Owner | Due |
|---|---|---|---|
| T1 failure signal loss | Add invariant tests: failing fixtures must preserve root error lines and file/line identifiers; enforce fallback-to-raw on filter parse failures | oz-coding | Before first implementation PR merge |
| T2 exit-code regression | Add dedicated integration suite validating exact exit propagation for success and failure commands; block merge on mismatch | oz-coding | Before first implementation PR merge |
| T3 non-deterministic output | Add golden snapshot determinism test (same input twice -> byte-identical compact output); stabilize ordering rules in filters | oz-coding | Before first implementation PR merge |
| T7 gain metric correctness | Add fixture-backed aggregation tests for totals, retention-window behavior, and JSON output schema; block merge on mismatches | oz-coding | Before first implementation PR merge |

---

## Fast-follow actions (for non-blocking Tigers)

| Risk | Action | Owner | Target |
|---|---|---|---|
| T4 hook surprise | Ship suggest-mode default first; add explicit opt-in for auto-rewrite and command exclusion docs | oz-maintainer + oz-coding | Within 30 days post-v1 |
| T5 overhead risk | Add benchmark command and CI budget checks for p95 overhead on short commands | oz-coding | Within 30 days post-v1 |

---

## Track items

| Risk | Signal to monitor | Trigger |
|---|---|---|
| T6 filter drift | Increased fallback-to-raw rate for a command family | >5% fallback rate over recent invocations |
| E3 maintenance load | PR cycle time and defect rate per new filter module | Sustained slowdown or repeated regressions |

---

## Premortem outcome

Launch recommendation: **proceed**, with T1/T2/T3 mitigations treated as hard implementation gates.  
The project should not declare v1 complete until those gates pass in CI and are documented in release notes.
