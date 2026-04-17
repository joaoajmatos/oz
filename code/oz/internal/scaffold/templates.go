package scaffold

// ozMDTmpl is the workspace manifest template.
const ozMDTmpl = `# OZ.md — Workspace Manifest

oz standard: {{.OZVersion}}
project: {{.Name}}
description: {{.Description}}

## Registered Agents
{{range .Agents}}
- **{{.Name}}**: ` + "`agents/{{.Name}}/AGENT.md`" + `
{{- end}}

## Source of Truth Hierarchy

1. ` + "`specs/`" + ` — highest trust. Architectural decisions and specifications.
2. ` + "`docs/`" + ` — architecture docs, open items.
3. ` + "`context/`" + ` — shared agent context snapshots.
4. ` + "`notes/`" + ` — lowest trust. Raw thinking, crystallize via ` + "`oz crystallize`" + `.
`

// agentsMDTmpl is the workspace entry point template for LLMs.
const agentsMDTmpl = `# AGENTS.md — LLM Entry Point

> You are entering an oz workspace. Read this file first.

## Project

**{{.Name}}**: {{.Description}}

## How to navigate this workspace

1. Find your role below.
2. Read your ` + "`AGENT.md`" + ` file.
3. Follow the read-chain in your agent definition before starting any task.

## Agents
{{range .Agents}}
### {{.Name}}

Agent definition: ` + "`agents/{{.Name}}/AGENT.md`" + `
{{end}}
## Source of Truth Hierarchy

When information conflicts, trust this order (highest to lowest):

1. ` + "`specs/`" + ` — architectural decisions and specifications
2. ` + "`docs/`" + ` — architecture docs, open items
3. ` + "`context/`" + ` — shared agent context snapshots
4. ` + "`notes/`" + ` — raw thinking (lowest trust)

Code is the source of truth for behaviour. When code and spec diverge, the code
wins — but the spec is flagged and updated to reflect reality.
`

// agentMDTmpl is the per-agent role definition template.
const agentMDTmpl = `# {{.Name}} Agent
{{if .Description}}
> {{.Description}}
{{end}}
## Role

<!-- Describe what this agent is responsible for. -->

## Responsibilities

-

## Out of scope

-

## Read-chain

Read these files in order before starting any task:

1. ` + "`AGENTS.md`" + ` — workspace entry point
2. ` + "`OZ.md`" + ` — workspace manifest
{{- if eq .Type "coding"}}
3. ` + "`rules/coding-guidelines.md`" + ` — hard constraints
4. ` + "`docs/architecture.md`" + ` — system architecture
5. ` + "`docs/open-items.md`" + ` — open questions and known issues
{{- else}}
3. ` + "`docs/architecture.md`" + ` — system architecture
4. ` + "`docs/open-items.md`" + ` — open questions and known issues
{{- end}}

<!-- Add spec files relevant to this agent's domain. -->

## Context topics

<!-- List ` + "`context/<topic>`" + ` entries this agent should read. -->
`

// codingGuidelinesTmpl is the hard-constraints rules file template.
const codingGuidelinesTmpl = `# Coding Guidelines

> Hard constraints for all code in this workspace.
> These apply to all agents and all contributors.

## Principles

1. **Code wins, spec follows.** Code is the source of truth for behaviour. When code and spec diverge, update the spec to match — don't revert the code.
2. **Convention over configuration.** Follow oz workspace conventions.
3. **Single responsibility.** Each module does one thing well.

## Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

- State assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them — don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## Simplicity

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it — don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

Every changed line should trace directly to the request.

## Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
` + "```" + `
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
` + "```" + `

## Rules

<!-- Add project-specific coding rules here. -->

## File naming

<!-- Add file naming conventions. -->

## Testing

<!-- Add testing requirements. -->
`

// architectureTmpl is the high-level architecture doc template.
const architectureTmpl = `# Architecture

> High-level architecture of {{.Name}}.

## Overview

{{.Description}}

## Components

<!-- Describe the major components. -->

## Data flow

<!-- Describe how data flows through the system. -->

## Key decisions

See ` + "`specs/decisions/`" + ` for architectural decision records (ADRs).
`

// openItemsTmpl is the open questions and known issues template.
const openItemsTmpl = `# Open Items

> Open questions, known issues, and pending decisions.
> Resolved items move to ` + "`specs/decisions/`" + `.

## Open Questions

<!-- Questions that need answers before proceeding. -->

## Known Issues

<!-- Known bugs or problems, with workarounds if available. -->

## Pending Decisions

<!-- Decisions that need to be made. -->
`

// decisionTemplateTmpl is the ADR template.
const decisionTemplateTmpl = `# ADR-000: [Title]

Date: YYYY-MM-DD
Status: Proposed

## Context

<!-- What is the issue motivating this decision? -->

## Decision

<!-- What was decided? -->

## Consequences

<!-- What are the positive and negative consequences? -->

## Alternatives considered

<!-- What other options were evaluated and why were they rejected? -->
`

// codeREADMETmpl is the code directory readme template.
const codeREADMETmpl = `# Code

This directory contains the source code for {{.Name}}.

## Structure

<!-- Describe the code structure. -->

## Getting started

<!-- How to build and run the code. -->
`

// ozDirGitignoreTmpl is the .gitignore for the .oz runtime directory.
// Keeps .oz/ tracked in git but ignores generated runtime artifacts.
const ozDirGitignoreTmpl = `# oz runtime artifacts — generated by oz context and oz audit
cache/
graph/
`

// claudeMDTmpl is the Claude Code native entry point template.
// It uses @file imports so Claude Code auto-loads the oz read-chain.
const claudeMDTmpl = `# CLAUDE.md — Claude Code Entry Point

This is an [oz workspace](https://github.com/oz-tools/oz).

**{{.Name}}**: {{.Description}}

## oz workspace

The entry point for all LLMs is ` + "`AGENTS.md`" + `. Read it to find your role and
follow the read-chain defined in your agent definition before starting any task.

@AGENTS.md
`

// readmeTmpl is the human-facing project readme template.
const readmeTmpl = `# {{.Name}}

{{.Description}}

## Getting started

<!-- Add setup instructions here. -->

## Development

This workspace follows the [oz convention](https://github.com/oz-tools/oz).
See ` + "`AGENTS.md`" + ` for workspace structure and agent definitions.
`
