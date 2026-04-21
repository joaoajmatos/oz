# Open Items

> Open questions, known issues, and pending decisions.
> Resolved items move to `specs/decisions/`.

## Open Questions

<!-- Questions that need answers before proceeding. -->

## Known Issues

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
| ORPH002 | `docs/planning/oz-context-v1-prd.md` has no inbound references | Historical PRD, superseded by the shipped feature. Accepted noise. |
| ORPH002 | `docs/planning/oz-context-v1-sprints.md` has no inbound references | Historical sprint plan, not linked from live docs. Accepted noise. |

AT-03 is considered **not triggered** (3 warnings ≤ threshold of 3).

## Pending Decisions

### CI-1 CT-01 threshold decision — no mitigation needed (2026-04-20)

After shipping code indexing (schema v2), running `oz context build` on this repository produced:

- `code_file`: 55
- `code_symbol`: 222
- Total code nodes: 277

CT-01 mitigation threshold is `> 500` code nodes. Current total is below threshold, so no partitioning or additional mitigation is required in CI-1.

### AGENT.md required sections updated — spec needs to reflect this

The oz workspace convention now requires `Rules` and `Skills` sections in every `AGENT.md`, in addition to the existing Role, Read-chain, Responsibilities, Out of scope, and Context topics sections.

- **Rules**: lists which rule files govern this agent's behavior (separate from read-chain, which is context-only)
- **Skills**: lists which skills this agent is authorized to invoke

`oz validate` should be updated to check for these sections.
`specs/oz-project-specification.md` should be updated to include Rules and Skills in the canonical AGENT.md template.
Owner: oz-spec (spec update) + oz-coding (validate enforcement)
