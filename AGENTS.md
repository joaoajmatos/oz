# AGENTS.md — LLM Entry Point

> You are entering an oz workspace. Read this file first.

## Project

**oz**: Open workspace convention and toolset for LLM-first development.

oz gives any LLM a structured, predictable workspace it can immediately understand —
without custom integrations or provider-specific configuration. The convention is
trust-based and provider-agnostic.

## How to navigate this workspace

1. Find your role in the **Agents** section below.
2. Open your `AGENT.md` file.
3. Follow the read-chain defined there before starting any task. Do not skip steps.

If no agent matches your task, read `specs/oz-project-specification.md` for full context,
then proceed with the source of truth hierarchy below.

---

## Agents

### oz-coding

**Builds the oz toolset in Go.**
Go implementation work: subcommands, internal packages, templates, tests.

Agent definition: `agents/oz-coding/AGENT.md`

### oz-maintainer

**Keeps workspace convention consistent and the repo healthy.**
Audits workspace structure, keeps agent definitions current, flags drift.

Agent definition: `agents/oz-maintainer/AGENT.md`

### oz-spec

**Evolves the oz standard specification.**
Owns `specs/oz-project-specification.md` and `specs/decisions/`. Crystallizes notes into spec language.

Agent definition: `agents/oz-spec/AGENT.md`

---

## Source of Truth Hierarchy

When information conflicts, trust this order (highest to lowest):

1. `specs/` — architectural decisions and specifications (highest trust)
2. `docs/` — architecture docs, open items
3. `context/` — shared agent context snapshots
4. `notes/` — raw thinking (lowest trust, crystallize via `oz crystallize`)

**Code is the source of truth for behaviour.** When code and spec diverge, the code
wins — the spec is flagged and updated to reflect reality. Drift is detected by `oz audit`.
