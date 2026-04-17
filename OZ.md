# OZ.md — Workspace Manifest

oz standard: v0.1
project: oz
description: Open workspace convention and toolset for LLM-first development

---

## Registered Agents

| Agent | Role | Definition |
|---|---|---|
| **oz-coding** | Builds the oz toolset in Go | `agents/oz-coding/AGENT.md` |
| **oz-maintainer** | Keeps workspace convention consistent and repo healthy | `agents/oz-maintainer/AGENT.md` |
| **oz-spec** | Evolves the oz standard specification | `agents/oz-spec/AGENT.md` |

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
├── rules/
│   └── coding-guidelines.md     # Hard constraints for all code
├── notes/                       # Raw thinking — lowest trust
├── code/
│   └── oz/                      # Go source: the oz binary
└── tools/
```
