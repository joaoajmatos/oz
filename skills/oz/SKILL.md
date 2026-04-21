---
name: oz
description: >-
  Runs the oz CLI in an oz workspace (validate, context build/query/serve and MCP, audit, optional enrich/review) for dogfooding, routing, drift checks, and scoped context from the graph.
triggers:
  - oz validate
  - oz context build
  - oz context query
  - oz context serve
  - oz audit
  - oz audit drift
  - oz audit staleness
  - MCP oz
  - graph.json
  - routing packet
  - dogfood oz
  - workspace health oz
---

# Skill: oz

> Runs the shipped `oz` binary for validation, structural graph, BM25F routing, audit, optional semantic overlay, and MCP — instead of guessing workspace shape or drift.

## When to invoke

Invoke this skill when you need to:

- Run **`oz validate`** after layout, `agents/`, `OZ.md`, `AGENTS.md`, or template changes.
- Regenerate **`context/graph.json`** before query, audit, or MCP (`oz context build`).
- Route a task or fetch scoped reads (**`oz context query`**, or MCP `query_graph` / `agent_for_task`).
- Check **staleness**, **orphans**, **drift**, or a **full audit** (`oz audit …`).
- Wire or reason about **`oz context serve`** (stdio MCP).
- Run **`oz context enrich`** / **`oz context review`** (OpenRouter + overlay).

Do not use this skill for tasks that belong only in read-chain files — use it when the **CLI or MCP** is the right interface.

---

## Steps

1. **Choose how to invoke the binary.**
   - With `oz` on **PATH**: `oz <subcommand>` from any directory under the workspace root (oz walks up to `AGENTS.md` + `OZ.md`).
   - In **this repo** without install: from repository root, `go run -C code/oz . <subcommand> …`.

2. **Refresh the structural graph when markdown under convention paths or indexed Go under `code/` changed.**
   Run `oz context build`. Details: [references/context-and-mcp.md](references/context-and-mcp.md).

3. **Validate workspace shape.**
   Run `oz validate`. See [references/audit-and-validate.md](references/audit-and-validate.md).

4. **Check generated artifacts (optional but recommended before handoff).**
   Run `oz audit staleness`, then `oz audit` or targeted `oz audit drift` / `oz audit orphans`. Subcommands, flags, and severity handling: [references/audit-and-validate.md](references/audit-and-validate.md).

5. **Route or narrow context (optional).**
   Run `oz context query "<one-line task>"`. Flags, JSON fields, **`no_clear_owner`**, and post-query reading order: [references/context-and-mcp.md](references/context-and-mcp.md).

6. **Semantic overlay (optional).**
   **`oz context enrich`** / **`oz context review`** — prerequisites and flags: [references/context-and-mcp.md](references/context-and-mcp.md).

7. **MCP (optional).**
   **`oz context serve`** — tools list and README wire-up: [references/context-and-mcp.md](references/context-and-mcp.md).
