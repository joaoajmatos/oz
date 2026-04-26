# AGENTS.md — LLM Entry Point

> You are entering an oz workspace. Read this file first.

## Project

**oz**: Open workspace convention and toolset for LLM-first development.

oz gives any LLM a structured, predictable workspace it can immediately understand —
with clean integrations for Claude Code, Cursor, and any other editor or model.

## How to navigate this workspace

1. Craft a one-line task statement and run `oz context query "<task>"` first.
2. Use the routing packet to choose the best agent (prefer `best_agent`, then confirm with the top candidates).
3. Open the chosen **Definition** path (`AGENT.md`) and follow its read-chain before starting edits.
4. Use `oz shell read` for targeted follow-up reads after routing; avoid broad manual file reading before query.

Default behavior contract (mandatory):
- In an oz workspace, **always start with `oz context query`** before manual codebase exploration.
- If the query depends on stale or missing graph artifacts, run `oz context build` and then re-run the same query.
- Only skip query-first for explicit user requests that are pure shell one-liners (for example, `date`) or for tasks that are fully outside the workspace.
- If you skip query-first, state the reason in one sentence before doing so.

Query crafting quick guide:
- Keep it one sentence with action + artifact + scope.
- Include concrete paths, command names, or output files when known.
- Prefer intent words like "implement", "debug", "refactor", "audit", "validate".

Examples:
- `oz context query "implement shell observe subcommand in code/oz/cmd with cobra tests"`
- `oz context query "debug oz context build failure when codeindex scans go files under code/oz/internal"`
- `oz context query "audit AGENTS.md and scaffold templates for routing-packet-first onboarding"`

For command and file-read workflows, **must** use `oz` wrappers (`oz shell run`,
`oz shell read`, `oz shell pipe`). Direct shell/file tooling is disallowed unless the
`oz` wrapper path is unavailable for the task. This keeps hook behavior and shell
compression consistent across agents and editors.

**File writes are out of scope for oz shell.** `oz shell run` is an output-compression
wrapper, not a shell. Use the host editor's native `Write`/`Edit` tools to create or
modify files — these are already compliant with oz policy and are not intercepted. Never
use shell heredocs or Python subprocesses to write files; the stdin pipe breaks inside
the wrapper and the shell environment errors are not recoverable.

Enforcement boundary: Both Claude Code and Cursor hooks enforce this policy.
Claude Code: shell rewrites via `PreToolUse(Bash)` → `oz-shell-rewrite-claude.sh`;
in-workspace native reads denied via `PreToolUse(Read)` → `oz-read-policy-claude.sh`.
Cursor: shell rewrites via `beforeShellExecution` → `oz-shell-rewrite-cursor.sh`;
out-of-workspace reads denied via `beforeReadFile` → `oz-read-policy-cursor.sh`.

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
| **oz-notes** | Working with `notes/`: classifying, deciding what to promote, and placing content in the right workspace tier (`specs/`, `docs/`, `specs/decisions/`) — not writing Go code and not making convention decisions unilaterally | `agents/oz-notes/AGENT.md` |

---

## Source of Truth Hierarchy

When information conflicts, trust this order (highest to lowest):

1. `specs/` — architectural decisions and specifications (highest trust)
2. `docs/` — architecture docs, open items
3. `context/` — oz-generated graph artifacts (query via `oz context query` or MCP)
4. `notes/` — raw thinking (lowest trust; use the `oz-notes` agent to promote content)

**Code is the source of truth for behaviour.** When code and spec diverge, the code
wins — the spec is flagged and updated to reflect reality. Drift is detected by `oz audit`.
