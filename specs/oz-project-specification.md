# oz — Project Specification

> This document is the canonical context for any LLM or contributor working on oz.
> Read it fully before writing any code.

---

## What is oz?

oz is an open source **workspace convention and toolset** for LLM-first development.

It solves a specific problem: LLMs (Claude, Codex, Cursor, etc.) have no persistent
understanding of a codebase. Every session starts from zero. oz gives any LLM a
structured, predictable workspace it can immediately understand — with clean
integrations for Claude Code, Cursor, and any other editor or model.

oz is **not** an agent orchestrator. It is a convention + toolset that any LLM can
follow by reading markdown files. The convention integrates cleanly with providers
rather than abstracting them away.

The core idea: open the workspace, read AGENTS.md, and the LLM knows exactly what
to do.

---

## Core Concepts

### The Workspace Convention

An oz workspace is a directory with a specific structure. Any LLM entering an oz
workspace reads `AGENTS.md` first, gets routed to the correct agent definition,
and executes a read-chain that loads the right rules, context, and specs before
starting work.

### The Read-Chain

Each agent definition specifies a read-chain — an ordered list of files to load
before starting any task. This is the LLM's boot sequence. It controls what the
LLM knows about itself, its constraints, and the codebase before it acts.

```mermaid
sequenceDiagram
  participant LLM
  participant Agents as AGENTS_md
  participant Agent as agents_name_AGENT_md
  LLM->>Agents: read_entry_point
  Agents->>LLM: route_to_agent_definition
  LLM->>Agent: execute_read_chain_in_order
```

### Source of Truth Hierarchy

When information conflicts, oz defines a strict trust order (highest to lowest):

1. `specs/` — architectural decisions and specifications (highest trust)
2. `docs/` — architecture docs, open items
3. `context/` — shared agent context snapshots
4. `notes/` — raw thinking, uncrystallized ideas (lowest trust)

Code is the source of truth for behaviour. When code and spec diverge, the code
wins — but the spec is flagged and updated to reflect reality.
Drift is detected by `oz audit`.

### Agents

Agents are role definitions — markdown files that tell an LLM how to behave,
what it's responsible for, what's out of scope, and what read-chain to follow.
Agents are not code. They are conventions.

Agents share context via `context/` at the workspace root, organized by topic.
Any agent can read any context topic.

#### AGENTS.md agent routing

`AGENTS.md` is the LLM entry point. Under `## Agents`, the workspace MUST include a single
markdown **routing table** (not free-form subsections) with exactly these columns, in order:

| Column | Purpose |
|---|---|
| **Agent** | Bold handle for the agent (e.g. `**oz-coding**`). Convention: matches the directory name under `agents/<name>/`. |
| **Use when** | One scannable line of **routing hints**: concrete situations (paths, commands, artefact types) where this agent is the right choice — not a vague job title. Prefer disambiguation (“X, not Y”) when two agents are easy to confuse. Avoid pipe characters (markdown column separators) inside the cell. |
| **Definition** | Backtick path to `agents/<name>/AGENT.md`. |

Optional packages and other tooling MAY append rows to this table when they register an agent,
using the same column shape so merges stay predictable.

`OZ.md` carries a **Registered Agents** table with the same three columns for the workspace manifest;
keep **Use when** text aligned between the two files when both exist.

#### AGENT.md required sections

Every `agents/<name>/AGENT.md` must contain these sections in order:

| Section | Purpose |
|---|---|
| `## Role` | One-paragraph description of what this agent does and its operating constraints |
| `## Read-chain` | Ordered list of files to load before starting any task (context only — not rules) |
| `## Rules` | Rule files that govern this agent's behavior — hard constraints, not just reading material |
| `## Skills` | Skills this agent is authorized to invoke (`skills/<name>/`) |
| `## Responsibilities` | Bulleted list of what this agent owns and produces |
| `## Out of scope` | Explicit disownment — what this agent must not do |
| `## Context topics` | Which `context/` topics this agent reads when relevant |

**Read-chain vs. Rules**: The read-chain loads context and understanding. Rules are behavioral constraints the agent must follow without exception. Keep them separate so the distinction is unambiguous — both to the LLM and to `oz validate`.

### Skills

Skills are reusable task playbooks that agents invoke to complete well-defined procedural tasks.
Each skill lives in `skills/<name>/` and has a fixed internal structure:

| Path | Purpose |
|---|---|
| `SKILL.md` | Entry point. Describes when to invoke the skill and the steps to follow. |
| `references/` | Sub-instructions and routing for skills with multiple execution paths. Each file covers one path. |
| `assets/` | Templates, examples, and support files the skill uses during execution. |

`references/` and `assets/` are optional for trivial skills, but required for skills with branching logic or templated outputs.

#### SKILL.md required frontmatter

Every `skills/<name>/SKILL.md` must begin with a YAML frontmatter block:

```yaml
---
name: <skill-name>
description: <One sentence describing what the skill does and when it is useful.>
triggers:
  - <keyword or phrase that should invoke this skill>
---
```

| Field | Purpose |
|---|---|
| `name` | The skill name. Must match the directory name (kebab-case). |
| `description` | One sentence summary for discovery by `oz context`. |
| `triggers` | Keywords/phrases used by `oz validate` and `oz context` for skill surfacing. |

#### SKILL.md required sections

Every `skills/<name>/SKILL.md` must contain these sections (after the frontmatter):

| Section | Purpose |
|---|---|
| `## When to invoke` | The conditions or signals that should trigger this skill |
| `## Steps` | Ordered, actionable instructions for executing the skill |

Additional sections (e.g. `## References`, `## Notes`) are allowed but not required.

---

## Workspace Structure

```mermaid
flowchart TB
  root["Workspace root"]
  root --> entry["AGENTS.md entry for LLMs"]
  root --> oz["OZ.md manifest"]
  root --> readme["README.md"]
  root --> agentsDir["agents/name/AGENT.md"]
  root --> specsDir["specs/decisions ADRs"]
  root --> docsDir["docs architecture open-items"]
  root --> ctxDir["context/topic/summary.md shared"]
  root --> skillsDir["skills/name SKILL references assets"]
  root --> rulesDir["rules/coding-guidelines.md"]
  root --> notesDir["notes/"]
  root --> codeDir["code/ README optional submodules"]
  root --> toolsDir["tools/"]
  root --> scriptsDir["scripts/"]
```

### Key conventions

- `AGENTS.md` is the single entry point for any LLM. It always exists at root and MUST route
  agents using the `## Agents` markdown table defined above.
- `OZ.md` is the workspace manifest. It declares the oz standard version and registered agents.
- `context/` is shared across all agents — organized by topic, not by agent.
- `code/` holds actual project code. May contain git submodules.
- `notes/` is the only low-trust layer. Everything else should be considered authoritative.
  Optional convention: `notes/planning/` for product-management artifacts (PRDs, pre-mortems, sprint plans) that inform work but are not normative until promoted to `specs/` or `docs/`.

---

## The oz Toolset

oz ships as a **single Go binary** with subcommands:

```mermaid
flowchart TB
  binary["oz binary"]
  binary --> initCmd["oz init scaffold workspace"]
  binary --> validateCmd["oz validate convention lint"]
  binary --> auditCmd["oz audit structure and drift"]
  binary --> contextCmd["oz context graph and query"]
  binary --> crystallizeCmd["oz crystallize planned"]
```

### oz init (priority — build this first)

Interactively scaffolds a new oz workspace. Asks:

- Project name
- Description
- Code directory mode: `inline` or `submodule`
- Agents: use default (`maintainer` only) or define custom agents

Generates the full directory structure and all required files from templates.

### oz validate (build second)

Lints a workspace against the oz convention. Reports:

- Missing required files/directories
- Missing recommended files/directories
- Agent definitions missing required sections (Role, Read-chain, Rules, Skills, Responsibilities, Out of scope, Context topics)
- OZ.md standard version
- `context/semantic.json` present but containing unreviewed nodes (warning, not error)

Exit code 0 = valid. Exit code 1 = invalid. Suitable for CI.

### oz audit (V1 complete)

On-demand workspace health checks (not automatic). Loads `context/graph.json` and walks
the workspace to report:

- **Orphans** (`ORPH001`–`ORPH003`): convention files with weak or missing inbound references.
- **Coverage** (`COV001`–`COV004`): agent scope paths vs disk and ownership overlaps.
- **Staleness** (`STALE001`–`STALE004`): graph / semantic overlay freshness.
- **Drift** (`DRIFT001`–`DRIFT003`): spec markdown references to missing `code/` paths,
  unknown exported identifiers (against graph `code_symbol` nodes), and exported symbols
  never mentioned in scanned markdown. Optional `--include-docs` and `--include-tests`.

Human output groups by severity; `--json` emits `schema_version: "1"` with sorted
`findings[]` and per-severity `counts`. Subcommands mirror each check; `oz audit graph-summary`
prints the historical node/edge summary. See `docs/architecture.md` and ADR 0001 for symbol
indexing policy (`go/parser` via `context build`, not tree-sitter in V1).

### oz context (V1 complete)

The knowledge graph engine. Shipped as V1 with the following subcommands:

**`oz context build`** — walks the workspace, parses all oz-convention files, extracts
cross-references, and writes a deterministic `context/graph.json`. Byte-identical output on
repeated runs with no changes (SHA-256 content hash embedded).

**`oz context query <text>`** — loads `graph.json`, scores agents using multi-field BM25F
with Porter stemming and temperature-scaled softmax, and returns a JSON routing packet:
agent, confidence, scope paths, context blocks (sorted by trust tier), and relevant concepts
from the semantic overlay when present. Supports `--raw` for debug output and `--include-notes`.

**`oz context enrich`** — sends the structural graph to an LLM via OpenRouter and writes
`context/semantic.json` with extracted concept nodes and typed relationships. Requires
`OPENROUTER_API_KEY`. Default model: `anthropic/claude-haiku-4`. Supports `--model`.

**`oz context review`** — presents unreviewed concepts and edges from `semantic.json` in a
human-readable table, then prompts accept or reject for each item. Supports `--accept-all`
for CI pipelines.

**`oz context serve`** — starts an MCP stdio server (protocol version `2024-11-05`) exposing
four tools: `query_graph`, `get_node`, `get_neighbors`, `agent_for_task`. Wire into Claude
Code or Cursor with `{"mcpServers":{"oz":{"command":"oz","args":["context","serve"]}}}`.

Staleness detection: on `oz context query` and `oz context serve` startup, `graph_hash` in
`semantic.json` is compared against the current `graph.json` hash. A warning is printed if
they diverge.

Performance: `oz context build` completes in < 500ms on a 50-file workspace (benchmark:
`go test -bench=BenchmarkBuild_50Files ./internal/context/`). Query output averages ≤ 10%
of full workspace token size on the 10-agent test fixture.

### oz crystallize (planned)

Promotes content from `notes/` up the hierarchy into the correct canonical location based
on convention. Notes become specs, docs, or context entries. Not yet implemented.

---

## Design Principles

1. **Convention over configuration.** oz workspaces are predictable. Any LLM or
   human can understand the structure without reading docs.

2. **Clean provider integrations.** oz works with Claude, Codex, Cursor, or any LLM.
   Markdown is the common interface; first-class integrations (hooks, CLAUDE.md, MCP)
   let each provider participate fully.

3. **Convention-enforced.** The read-chain and hierarchy are backed by hooks and
   `oz validate`. Conventions are machine-checkable, not just hoped-for.

4. **Code wins, spec follows.** Code is the source of truth for behaviour.
   When code and spec diverge, the spec is flagged and updated to match.
   Drift is surfaced by `oz audit`.

5. **Single binary.** oz ships as one Go binary. No runtime dependencies.
   Install with one command.

6. **oz eats its own dog food.** The oz repository is itself an oz workspace.

---

## Current Status (V1 complete — April 2026)

- **oz init**: complete — scaffolds a full oz-compliant workspace from embedded templates.
- **oz validate**: complete — enforces all 7 required AGENT.md sections, checks required files/directories, warns on unreviewed semantic nodes.
- **oz audit**: V1 complete — orphans, coverage, staleness, drift, JSON report, deterministic ordering, `graph-summary` stub preserved. Multi-language / tree-sitter drift is deferred.
- **oz context**: V1 complete — `build`, `query`, `enrich`, `review`, and `serve` all ship. MCP server validated. BM25F scoring with Porter stemming and softmax routing.
- **oz crystallize**: planned — not yet implemented.

See `docs/architecture.md` for the full system design and `context/implementation/summary.md` for the V1 implementation snapshot.
