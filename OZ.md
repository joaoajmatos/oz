# OZ.md — Workspace Manifest

oz standard: v0.1
project: oz
description: Open workspace convention and toolset for LLM-first development

---

## Registered Agents

| Agent | Use when | Definition |
|---|---|---|
| **oz-coding** | Primary work is Go in `code/oz/`, the `oz` CLI, its tests, or embedded files under `internal/scaffold/` | `agents/oz-coding/AGENT.md` |
| **oz-maintainer** | Convention work: **new or updated agents, skills, or rules** (via `skills/workspace-management/`), keeping `AGENTS.md` / `OZ.md` / manifests accurate, `oz validate` / `oz audit`, layout — not shipping Go in `code/oz/` and not rewriting normative sections of `specs/oz-project-specification.md` | `agents/oz-maintainer/AGENT.md` |
| **oz-spec** | Primary work is normative convention text: `specs/oz-project-specification.md`, `specs/decisions/` (ADRs), or spec alignment — not implementing Go in `code/oz/` | `agents/oz-spec/AGENT.md` |

---

## Source of Truth Hierarchy

1. `specs/` — highest trust. Architectural decisions and specifications.
2. `docs/` — architecture docs, open items.
3. `context/` — shared agent context snapshots.
4. `notes/` — lowest trust. Raw thinking, crystallize via `oz crystallize`.

---

## Workspace Structure

```
oz/
├── AGENTS.md                    # Entry point for all LLMs
├── OZ.md                        # This file — workspace manifest
├── README.md                    # Human-facing project readme
├── agents/
│   ├── oz-coding/AGENT.md       # Go implementation agent
│   ├── oz-maintainer/AGENT.md   # Convention and repo health agent
│   └── oz-spec/AGENT.md         # Specification evolution agent
├── specs/
│   ├── oz-project-specification.md  # Canonical oz standard (primary artifact)
│   └── decisions/               # ADRs for all significant design decisions
├── docs/
│   ├── architecture.md          # High-level architecture
│   └── open-items.md            # Open questions, known issues, pending decisions
├── context/                     # Shared agent context snapshots (organized by topic)
├── skills/
│   └── <name>/
│       ├── SKILL.md             # Entry point: when to invoke and steps to follow
│       ├── references/          # Sub-instructions and routing for branching skills
│       └── assets/              # Templates, examples, and support files
├── rules/
│   └── coding-guidelines.md     # Hard constraints for all code
├── notes/                       # Raw thinking — lowest trust
├── code/
│   └── oz/                      # Go source: the oz binary
└── tools/
```
