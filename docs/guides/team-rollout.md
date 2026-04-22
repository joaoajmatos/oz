# Team Rollout

Roll out `oz` across a team with minimal disruption and clear ownership.

## Goal

Adopt the convention as a shared operating model across local development, PR flow, and CI, so contributors at different levels work with the same rules and routing model.

## Preconditions

- Baseline single-workspace adoption is already stable.
- Team agrees on initial owners for `specs/`, `docs/`, and `agents/`.

## Steps

1. Define ownership boundaries in agent responsibilities and scopes.
2. Add CI checks for convention health.
3. Ensure contributors run context rebuilds after meaningful convention/code edits.
4. Decide your repo strategy explicitly:
   - single-repo workspace (all code in this repository), or
   - meta-repo workspace (separate code repositories mounted as git submodules in `code/`).
5. Standardize pull-request review checks around:
   - `oz validate`
   - `oz audit` (or selected checks)
   - language-level test commands
6. Create a lightweight cadence to review drift and stale context.

## Verify

- CI fails when convention-critical files drift from expectations.
- New contributors can start from `AGENTS.md` without additional tribal knowledge.
- Teams can explain where decisions belong (`specs/` vs `docs/` vs `notes/`).
- Teams can explain how submodule repos are kept independent while shared workspace conventions stay centralized at the root.

## Common pitfalls

- Assigning strict ownership without updating routing hints.
- Overloading CI with non-essential checks before baseline cleanup.
- Keeping strategic decisions in `notes/` instead of promoting them to canonical layers.
