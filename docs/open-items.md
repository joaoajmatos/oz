# Open Items

> Open questions, known issues, and pending decisions.
> Resolved items move to `specs/decisions/`.

## Open Questions

<!-- Questions that need answers before proceeding. -->

## Known Issues

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
| ORPH002 | `docs/oz-context-v1-prd.md` has no inbound references | Historical PRD, superseded by the shipped feature. Accepted noise. |
| ORPH002 | `docs/sprints.md` has no inbound references | Historical sprint plan, not linked from live docs. Accepted noise. |

AT-03 is considered **not triggered** (3 warnings ≤ threshold of 3).

## Pending Decisions

### AGENT.md required sections updated — spec needs to reflect this

The oz workspace convention now requires `Rules` and `Skills` sections in every `AGENT.md`, in addition to the existing Role, Read-chain, Responsibilities, Out of scope, and Context topics sections.

- **Rules**: lists which rule files govern this agent's behavior (separate from read-chain, which is context-only)
- **Skills**: lists which skills this agent is authorized to invoke

`oz validate` should be updated to check for these sections.
`specs/oz-project-specification.md` should be updated to include Rules and Skills in the canonical AGENT.md template.
Owner: oz-spec (spec update) + oz-coding (validate enforcement)
