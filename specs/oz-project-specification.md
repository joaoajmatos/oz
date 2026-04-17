# oz — Project Specification

> This document is the canonical context for any LLM or contributor working on oz.
> Read it fully before writing any code.

---

## What is oz?

oz is an open source **workspace convention and toolset** for LLM-first development.

It solves a specific problem: LLMs (Claude, Codex, Cursor, etc.) have no persistent
understanding of a codebase. Every session starts from zero. oz gives any LLM a
structured, predictable workspace it can immediately understand — without custom
integrations or provider-specific configuration.

oz is **not** an agent orchestrator. It is a convention + toolset that any LLM can
follow by reading markdown files. The convention is trust-based and provider-agnostic.

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

---

## Workspace Structure

```
<workspace>/
├── AGENTS.md                    # Entry point for all LLMs. Routes to correct agent.
├── OZ.md                        # Workspace manifest. oz standard version, registered agents.
├── README.md                    # Human-facing project readme.
├── agents/
│   └── <name>/
│       └── AGENT.md             # Role definition + read-chain for this agent.
├── specs/
│   └── decisions/
│       └── _template.md         # ADR template. All decisions live here.
├── docs/
│   ├── architecture.md          # High-level architecture.
│   └── open-items.md            # Open questions, known issues, pending decisions.
├── context/
│   └── <topic>/
│       └── summary.md           # Shared context snapshots, organized by topic.
│                                # Any agent can read any topic. Not agent-specific.
├── skills/                      # Reusable LLM skill definitions.
│   └── <name>/
├── rules/
│   └── coding-guidelines.md     # Hard constraints for all code in this workspace.
├── notes/                       # Raw thinking. Lowest trust. Crystallized via oz crystallize.
├── code/                        # Project code. Can be inline or git submodules.
│   └── README.md
├── tools/                       # oz tools + custom workspace tooling.
└── scripts/
    └── setup.sh
```

### Key conventions

- `AGENTS.md` is the single entry point for any LLM. It always exists at root.
- `OZ.md` is the workspace manifest. It declares the oz standard version and registered agents.
- `context/` is shared across all agents — organized by topic, not by agent.
- `code/` holds actual project code. May contain git submodules.
- `notes/` is the only low-trust layer. Everything else should be considered authoritative.

---

## The oz Toolset

oz ships as a **single Go binary** with subcommands:

```
oz init       # scaffold a new oz-compliant workspace interactively
oz validate   # lint a workspace against the oz convention (coming soon)
oz audit      # detect drift between specs and code (coming soon)
oz context    # build and query the workspace knowledge graph (coming soon)
oz crystallize # promote notes into canonical workspace truth (coming soon)
```

### oz init (priority — build this first)

Interactively scaffolds a new oz workspace. Asks:
- Project name
- Description
- Code directory mode: `inline` or `submodule`
- Agents: use defaults (coding, maintainer, onboarding) or define custom

Generates the full directory structure and all required files from templates.

### oz validate (build second)

Lints a workspace against the oz convention. Reports:
- Missing required files/directories
- Missing recommended files/directories
- Agent definitions missing required sections
- OZ.md standard version

Exit code 0 = valid. Exit code 1 = invalid. Suitable for CI.

### oz audit (build third)

Detects drift between specs and code. On-demand only (not automatic).
Uses tree-sitter to parse code structure and compares against specs.

### oz context (build later)

The knowledge graph engine. Indexes the workspace:
- Markdown files respecting the source of truth hierarchy
- Code via tree-sitter (AST-level, not just text)
- Builds a knowledge graph with hierarchy encoded as node metadata
- Serves relevant subgraphs to LLMs for a given task

This is the most complex tool. Build init and validate first.

### oz crystallize (build later)

Promotes content from `notes/` up the hierarchy into the correct canonical
location based on convention. Notes become specs, docs, or context entries.

---

## Repository Structure

oz is a Go monorepo. Single binary, multiple subcommands.

```
oz/                              # the oz repo is itself an oz workspace
├── AGENTS.md
├── OZ.md
├── agents/
│   ├── oz-coding/
│   │   └── AGENT.md             # builds oz tools
│   ├── oz-maintainer/
│   │   └── AGENT.md             # keeps convention consistent
│   └── oz-spec/
│       └── AGENT.md             # evolves the oz standard specification
├── specs/
│   └── decisions/               # ADRs for oz design decisions
├── docs/
├── context/
├── notes/
├── code/
│   └── oz/                      # the actual Go code lives here
│       ├── main.go
│       ├── cmd/
│       │   ├── init.go          # oz init subcommand
│       │   ├── validate.go      # oz validate subcommand
│       │   ├── audit.go         # oz audit subcommand
│       │   ├── context.go       # oz context subcommand
│       │   └── crystallize.go   # oz crystallize subcommand
│       ├── internal/
│       │   ├── convention/      # oz workspace convention as Go types
│       │   │   └── convention.go
│       │   ├── workspace/       # workspace detection and traversal
│       │   │   └── workspace.go
│       │   ├── scaffold/        # scaffolding logic for oz init
│       │   │   ├── scaffolder.go
│       │   │   └── templates.go
│       │   └── validate/        # validation logic for oz validate
│       │       └── validator.go
│       └── go.mod
└── tools/
```

---

## Go Implementation Notes

### Convention (internal/convention)

The convention package defines the oz workspace standard as Go types.
This is the source of truth for what a valid oz workspace looks like.

```go
// Hierarchy defines source of truth order, highest trust first
var Hierarchy = []string{"specs", "docs", "context", "notes"}

// Directories defines required/recommended/optional directories
var Directories = map[string]string{
    "agents":          "required",
    "specs/decisions": "required",
    "docs":            "required",
    "context":         "required",
    "skills":          "required",
    "rules":           "required",
    "notes":           "required",
    "code":            "recommended",
    "tools":           "optional",
    "scripts":         "optional",
}

// RootFiles defines required files at workspace root
var RootFiles = map[string]string{
    "AGENTS.md": "required",
    "OZ.md":     "required",
    "README.md":  "recommended",
}
```

### Workspace (internal/workspace)

Detects and represents an oz workspace. Provides:
- `New(path)` — load workspace from path
- `Valid()` — check if workspace has required root files
- `Agents()` — list registered agents
- `Hierarchy()` — return layers with existence status

### Templates

All generated markdown files follow consistent templates.
Key templates: `OZ.md`, `AGENTS.md`, `AGENT.md`, `coding-guidelines.md`,
`architecture.md`, `open-items.md`, `decisions/_template.md`, `code/README.md`.

Templates should be embedded in the binary using Go's `embed` package.

### CLI

Use `github.com/spf13/cobra` for the CLI. Single `oz` binary, subcommands per tool.

```
oz init [path]      # defaults to current directory
oz validate [path]  # defaults to current directory
```

---

## Design Principles

1. **Convention over configuration.** oz workspaces are predictable. Any LLM or
   human can understand the structure without reading docs.

2. **Provider agnostic.** oz works with Claude, Codex, Cursor, or any LLM.
   No provider-specific integrations. Markdown is the interface.

3. **Trust-based.** The read-chain and hierarchy are not enforced by code.
   They are conventions that well-behaved LLMs follow. This is intentional.

4. **Code wins, spec follows.** Code is the source of truth for behaviour.
   When code and spec diverge, the spec is flagged and updated to match.
   Drift is surfaced by `oz audit`.

5. **Single binary.** oz ships as one Go binary. No runtime dependencies.
   Install with one command.

6. **oz eats its own dog food.** The oz repository is itself an oz workspace.

---

## Current Status

- Ruby prototype exists (oz-init, oz-core) — being ported to Go
- Decision: Go chosen over Ruby for single binary distribution
- oz-init is the first priority
- oz-validate is second
- oz-context (knowledge graph) is the most complex, built last

## What to build next

Start here:

```
code/oz/go.mod
code/oz/main.go
code/oz/internal/convention/convention.go
code/oz/internal/workspace/workspace.go
code/oz/internal/scaffold/templates.go
code/oz/internal/scaffold/scaffolder.go
code/oz/cmd/init.go
```

Get `oz init` working end-to-end first. Then oz-validate.