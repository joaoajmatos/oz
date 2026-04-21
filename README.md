# oz

**oz** is an open workspace convention and a single **Go CLI** for LLM-first development: predictable layout, explicit agents, specs-first truth, and machine-checkable health — without tying you to one editor or model vendor.

- **Convention**: [`AGENTS.md`](./AGENTS.md), [`OZ.md`](./OZ.md), `agents/`, `specs/`, `docs/`, `context/`, `skills/`, `rules/`, `notes/`, optional `code/`.
- **Tooling**: `oz init`, `oz validate`, `oz repair`, `oz context` (build / query / enrich / review / **serve** for MCP), `oz audit`, and more as the project grows.

Canonical behaviour and vocabulary live in **[`specs/oz-project-specification.md`](./specs/oz-project-specification.md)**. This repository is both the **reference implementation** (`code/oz`) and a **dogfooding workspace** that follows the same layout.

---

## Install the CLI

From a clone of this repo:

```bash
cd code/oz
make build          # writes ./bin/oz
# or
go install .
```

Ensure the resulting binary is on your `PATH` as `oz` (or invoke it by full path).

---

## Create a new workspace

```bash
mkdir my-workspace && cd my-workspace
oz init
```

Interactive prompts set the project name, description, code layout (**inline** vs **submodule**), and agents. That generates `AGENTS.md`, `OZ.md`, a sensible **README.md**, `agents/*/AGENT.md`, `skills/workspace-management/`, stubs under `docs/`, and (in **inline** mode) `code/README.md`.

**`oz repair`** recreates any missing default files from the same templates **without overwriting** files you already edited.

---

## Work in this repository

Development here is **intent-first**: you describe the outcome you want (“check that this workspace still matches the convention”, “refresh the graph before an audit”, “see if specs and code have drifted”, “run the usual checks before we merge”), and the **LLM** working in the repo follows **`AGENTS.md`**, the per-agent read-chain, and the **`skills/oz/`** playbook to decide **when** to invoke **`oz`** and with which flags — the same way it chooses `go test`, file edits, or anything else.

| Intent | What “done” looks like |
|--------|-------------------------|
| Convention health | Required paths, manifests, and agent files satisfy **`oz validate`** |
| Graph matches the tree | **`context/graph.json`** rebuilt after meaningful edits (`oz context build`) |
| Workspace health | Findings from **`oz audit`** (staleness, drift, orphans, …) |
| Go implementation | **`go test ./...`** (and **`go vet ./...`**) clean under **`code/oz`** |

**Routing is for models, not a daily human workflow.** `oz context query` and MCP tools like **`query_graph`** / **`agent_for_task`** exist so the **assistant** can decide which agent definition to lean on, which read-chain files to open next, and which **context blocks** to load — scoped work from the graph instead of reading the whole tree. People reach for that mainly when debugging routing or wiring MCP.

You rarely need to memorize subcommands: point collaborators or models at **`AGENTS.md`** and **`skills/oz/`**. Shell commands are there for CI and for humans debugging the same paths the agents use. **[`docs/architecture.md`](./docs/architecture.md)** explains how the graph, query engine, and audit layers connect.

---

## Editor integrations

oz integrates cleanly with Claude Code and Cursor. Run the appropriate command inside any oz workspace:

```bash
oz add claude   # writes CLAUDE.md + Claude Code hooks (.claude/settings.json)
oz add cursor   # writes Cursor hooks (.cursor/hooks.json)
```

Both commands install the three shared hook scripts under `.oz/hooks/`:

| Hook | What it does |
|------|-------------|
| `oz-session-init.sh` | Injects agent routing context at session start |
| `oz-after-edit.sh` | Runs `oz validate` + `oz context build` after `.md`/`.go` edits |
| `oz-pre-commit.sh` | Blocks `git commit` if `oz validate` or `oz audit staleness` fails |

`oz init` also offers to configure hooks as part of the workspace creation flow.

---

## MCP (Model Context Protocol)

`oz context serve` is a stdio MCP server so Claude Code, Cursor, and other clients can call **`query_graph`**, **`get_node`**, **`get_neighbors`**, and **`agent_for_task`** against the current workspace.

Add to project **`.mcp.json`** or global MCP config:

```json
{
  "mcpServers": {
    "oz": {
      "command": "oz",
      "args": ["context", "serve"]
    }
  }
}
```

After agents, specs, docs, or indexed code change, the graph should be rebuilt so audits and routing stay truthful — that usually means an **`oz context build`** (or equivalent via MCP) once the model knows the tree moved.

---

## Documentation map

| Doc | Purpose |
|-----|---------|
| [`AGENTS.md`](./AGENTS.md) | LLM entry — roles and agent definitions |
| [`specs/oz-project-specification.md`](./specs/oz-project-specification.md) | Normative workspace standard |
| [`docs/architecture.md`](./docs/architecture.md) | System design for this implementation |
| [`docs/open-items.md`](./docs/open-items.md) | Gaps and follow-ups |

---

## Contributing

Issues and PRs are welcome. Before merging, expect the usual outcomes: **tests and vet clean for `code/oz`**, and the **workspace validates** from the repo root — whether a human runs those commands or an agent does on your behalf. Follow the read-chain for the role that matches your change (see **`AGENTS.md`**).
