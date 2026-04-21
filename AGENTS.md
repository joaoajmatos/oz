# AGENTS.md — LLM Entry Point

> You are entering an oz workspace. Read this file first.

## Project

**oz**: Open workspace convention and toolset for LLM-first development.

oz gives any LLM a structured, predictable workspace it can immediately understand —
with clean integrations for Claude Code, Cursor, and any other editor or model.

## How to navigate this workspace

1. In **Agents** below, pick the row whose **Use when** column best matches your situation (not every keyword has to match).
2. Open the **Definition** path (`AGENT.md` for that agent).
3. Follow the read-chain defined there before starting any task. Do not skip steps.

Each agent authorizes `skills/oz/` for the full `oz` CLI and MCP workflow (`validate`,
`context` build/query/serve, `audit`, optional enrich/review) so this workspace dogfoods the
shipped toolset.

If no agent matches your task, read `specs/oz-project-specification.md` for full context,
then proceed with the source of truth hierarchy below.

---

## Agents

**Use when** is a routing hint: it should say *which situations* choose that agent, not repeat the job title from `AGENT.md`. One line per cell; do not use `|` inside the cell (it breaks the markdown table).

| Agent | Use when | Definition |
|---|---|---|
| **oz-coding** | Primary work is Go in `code/oz/`, the `oz` CLI, its tests, or embedded files under `internal/scaffold/` | `agents/oz-coding/AGENT.md` |
| **oz-maintainer** | Convention work: **new or updated agents, skills, or rules** (via `skills/workspace-management/`), keeping `AGENTS.md` / `OZ.md` / manifests accurate, `oz validate` / `oz audit`, layout — not shipping Go in `code/oz/` and not rewriting normative sections of `specs/oz-project-specification.md` | `agents/oz-maintainer/AGENT.md` |
| **oz-spec** | Primary work is normative convention text: `specs/oz-project-specification.md`, `specs/decisions/` (ADRs), or spec alignment — not implementing Go in `code/oz/` | `agents/oz-spec/AGENT.md` |

---

## Source of Truth Hierarchy

When information conflicts, trust this order (highest to lowest):

1. `specs/` — architectural decisions and specifications (highest trust)
2. `docs/` — architecture docs, open items
3. `context/` — shared agent context snapshots
4. `notes/` — raw thinking (lowest trust, crystallize via `oz crystallize`)

**Code is the source of truth for behaviour.** When code and spec diverge, the code
wins — the spec is flagged and updated to reflect reality. Drift is detected by `oz audit`.
