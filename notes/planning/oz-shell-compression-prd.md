# oz Shell Compression Layer — PRD

**Author**: oz-spec  
**Date**: 2026-04-25  
**Status**: Draft  
**Spec**: [specs/oz-shell-compression-specification.md](../../specs/oz-shell-compression-specification.md)  
**ADR**: [specs/decisions/0005-oz-shell-compression-architecture.md](../../specs/decisions/0005-oz-shell-compression-architecture.md)  
**Sprint plan**: [oz-shell-compression-sprints.md](oz-shell-compression-sprints.md)

---

## 1. Summary

This PRD defines a new `oz` shell compression layer that reduces token-heavy shell output before it reaches LLM context.  
The feature ships in pure Go and supports both explicit wrapper usage (`oz shell run -- <cmd>`) and optional transparent interception via hook rewrite.

---

## 2. Contacts

| Name/Role | Comment |
|---|---|
| oz-coding (implementation owner) | Builds `cmd` and `internal` packages, tests, and performance tuning |
| oz-spec (spec owner) | Maintains normative contracts in `specs/` |
| oz-maintainer (repo health) | Ensures docs/validation/audit alignment and release hygiene |
| End users (LLM-assisted developers) | Primary beneficiary: lower token cost and less noise in shell workflows |

---

## 3. Background

`oz context query` already reduces token load for workspace retrieval by selecting focused context.  
However, real coding sessions still spend many tokens on shell output (`git`, `rg`, test runs, lints, logs).

Why now:

- Shell output is one of the largest uncontrolled token sinks in day-to-day agent execution.
- The project has a clear architecture slot for an execution-time compaction layer.
- We already have deterministic testing patterns and can enforce correctness guarantees (especially exit-code preservation).

---

## 4. Objective

Build a deterministic shell-output compression layer for `oz` that lowers token usage while preserving correctness and failure visibility.

### Why it matters

- **For users:** lower LLM costs, faster iterations, less context noise.
- **For oz:** stronger end-to-end value (context selection + execution-output optimization).
- **For reliability:** keep CI-safe semantics via strict exit-code propagation.

### Key Results (SMART)

1. **KR1 — Token reduction (MVP commands):**  
   For `git status`, `git diff`, `rg`, and `go test`, median token reduction is **>= 60%** on fixture suites.

2. **KR2 — Correctness preservation:**  
   **100%** of integration tests confirm wrapped command exit codes are preserved.

3. **KR3 — Determinism:**  
   For fixed fixtures, compact output snapshots are byte-stable across two consecutive runs in CI.

4. **KR4 — Performance overhead:**  
   Wrapper overhead p95 for short commands is **<= 30ms** on the benchmark runner.

5. **KR5 — Adoption readiness:**  
   Both explicit and transparent modes are documented and validated in at least one supported agent integration path.

6. **KR6 — Analytics availability:**  
   `oz shell gain` is available in v1 and returns aggregate token/perf savings from local tracking data.

---

## 5. Market Segment(s)

Primary segment:

- developers and teams using LLM coding agents with frequent shell interactions.

Jobs to be done:

- “Help me execute and inspect commands without drowning my context window.”
- “Keep all failure-critical information but remove repetitive noise.”
- “Let me trust wrapper output in CI-sensitive workflows.”

Constraints:

- single Go binary preference in `oz`.
- no network dependency for core operation.
- deterministic, testable behavior required.

---

## 6. Value Proposition(s)

### Core value

- **High signal per token** for shell workflows.
- **Safe by default** through strict exit-code and fail-safe fallback behavior.
- **Low friction adoption** with explicit mode first and optional transparent mode.

### Better than alternatives

- Unlike manual prompt instructions, behavior is deterministic and test-enforced.
- Unlike ad hoc truncation, filters preserve known critical diagnostics per command family.
- Unlike external sidecars, pure Go implementation aligns with `oz` build/distribution model.

---

## 7. Solution

### 7.1 UX / Flows

#### Flow A — explicit usage

1. User/agent runs `oz shell run -- go test ./...`.
2. `oz` executes command, captures raw output.
3. Matching filter compacts output.
4. Compact result is emitted, plus tee pointer on failures.
5. Exit code from wrapped command is preserved.

#### Flow B — transparent usage (optional)

1. Hook integration is enabled.
2. Incoming shell commands are rewritten to `oz shell run -- ...`.
3. Same execution/filtering pipeline as explicit mode.
4. Hook failures fall open (original command still runs).

### 7.2 Key Features (V1)

1. **`oz shell run -- <cmd...>` command**
   - flags: `--mode compact|raw`, `--json`, `--tee`, verbosity levels
2. **Deterministic filter engine**
   - MVP specialized filters: `git status`, `git diff`, `rg`, `go test`
   - generic safe profile for unknown commands
3. **Strict correctness guarantees**
   - exit-code propagation
   - filter-failure fallback to raw output
4. **Local observability**
   - token estimate before/after
   - duration and matched filter metadata
5. **Optional transparent interception**
   - opt-in hook rewrite with command exclusion controls
6. **`oz shell gain` analytics command**
   - summarize tracked command usage, token reduction, and execution timing
   - support human-readable output and JSON output

### 7.3 Technology

- Pure Go implementation under `code/oz`.
- Suggested package shape:
  - `cmd/shell.go`
  - `internal/shell/exec`
  - `internal/shell/filter`
  - `internal/shell/track`
  - `internal/shell/tee`
  - `internal/shell/hook` (integration adapters)
- Backing store for tracking: SQLite (see clarified decision SHL-01 below).

### 7.4 Assumptions

- `chars / 4` token estimator is sufficient for comparative savings metrics.
- Command-family heuristics remain robust enough across common tool versions.
- Most value is achieved from the first 4 command families.
- Suggest-mode-first for transparent interception reduces rollout risk.

---

## 8. Release

### V1 scope

- spec and ADR finalized
- explicit command mode
- MVP command families
- JSON envelope
- local tracking and tee support
- `oz shell gain` analytics command
- optional transparent mode (feature flag / opt-in)

### Fast-follow

- expand command-family coverage
- richer per-agent transparent integrations
- additional analytics subcommands beyond `oz shell gain`

### Out of scope for V1

- remote telemetry
- all-command perfect filtering
- full parity with broader external tooling ecosystems

---

## 9. Clarified Decisions and Open Items

This section resolves what we can decide before implementation and labels what remains post-implementation.

### Resolved now

1. **SHL-01 tracking backend (resolved): SQLite for v1**
   - rationale: local, queryable, robust for retention and aggregate metrics.

2. **SHL-02 transparent default behavior (resolved): suggest-mode first**
   - default posture: explicit mode remains primary; transparent interception is opt-in and starts in suggest mode.
   - auto-rewrite can be enabled explicitly once validated in target integration.

3. **MVP command families (resolved):**
   - `git status`, `git diff`, `rg`, `go test`

4. **Core safety contract (resolved):**
   - strict exit-code preservation
   - filter-failure fallback to raw output
   - fail-open hook behavior

### Deferred until implementation evidence

1. exact p95 overhead across supported environments
2. command-version edge compatibility for parser heuristics
3. default `tee` retention thresholds under large-output workloads
4. final `oz shell gain` sub-flag set for advanced reporting modes

These deferred items should remain in `docs/open-items.md` until measured.
