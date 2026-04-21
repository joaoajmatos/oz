<pre align="center">
    ███████    ███████████
  ███▒▒▒▒▒███ ▒█▒▒▒▒▒▒███ 
 ███     ▒▒███▒     ███▒  
▒███      ▒███     ███    
▒███      ▒███    ███     
▒▒███     ███   ████     █
 ▒▒▒███████▒   ███████████
   ▒▒▒▒▒▒▒    ▒▒▒▒▒▒▒▒▒▒▒ 
</pre>

<p align="center"><strong>oz</strong> — open workspace convention and Go CLI for LLM-first development</p>

<p align="center">
  <a href="./AGENTS.md">AGENTS.md</a> ·
  <a href="./specs/oz-project-specification.md">Specification</a> ·
  <a href="./docs/architecture.md">Architecture</a>
</p>

---

## What is oz?

**oz** gives any LLM a predictable workspace: explicit **agents**, **specs-first** truth, a machine-checkable layout, and a small **Go binary** (`oz`) for validate, audit, and context graph workflows — without locking you to one editor or model vendor.

Running `oz` with **no subcommand** prints the banner above with terminal colours (see [`code/oz/cmd/banner.go`](./code/oz/cmd/banner.go)).

| Layer | What you get |
|--------|----------------|
| **Convention** | [`AGENTS.md`](./AGENTS.md), [`OZ.md`](./OZ.md), `agents/`, `specs/`, `docs/`, `context/`, `skills/`, `rules/`, `notes/`, optional `code/` |
| **CLI** | `oz init`, `oz validate`, `oz repair`, `oz context` (build / query / enrich / review / **serve** for MCP), `oz audit`, and more as the project grows |

Canonical vocabulary and behaviour live in **[`specs/oz-project-specification.md`](./specs/oz-project-specification.md)**. This repository is the **reference implementation** (`code/oz`) and a **dogfooding workspace** that follows the same layout.

---

## Install

From a clone of this repository:

```bash
cd code/oz
make build          # writes ./bin/oz
# or
go install .
```

Put the binary on your `PATH` as `oz`, or call it by full path.

---

## Quick start: new workspace

```bash
mkdir my-workspace && cd my-workspace
oz init
```

Interactive prompts set the project name, description, code layout (**inline** vs **submodule**), and agents. That scaffolds `AGENTS.md`, `OZ.md`, a starter **README.md**, `agents/*/AGENT.md`, `skills/workspace-management/`, stubs under `docs/`, and (in **inline** mode) `code/README.md`.

**`oz repair`** recreates any missing default files from the same templates **without overwriting** files you already changed.

---

## Working in an oz workspace

Development is **intent-first**: you describe the outcome, and the **LLM** follows **`AGENTS.md`**, each agent’s read-chain, and **`skills/oz/`** to decide when to run **`oz`** — the same way it chooses `go test` or edits.

| Intent | What “done” looks like |
|--------|-------------------------|
| Convention health | **`oz validate`** passes |
| Graph matches the tree | **`context/graph.json`** rebuilt after meaningful edits (`oz context build`) |
| Workspace health | **`oz audit`** (staleness, drift, orphans, …) |
| Go implementation | **`go test ./...`** and **`go vet ./...`** clean under **`code/oz`** |

**Routing is for models, not a daily human workflow.** `oz context query` and MCP tools such as **`query_graph`** / **`agent_for_task`** help the assistant pick an agent, open the right read-chain files, and load **context blocks** from the graph instead of reading the whole tree. People use that mainly when debugging routing or MCP wiring. For the full picture, see **[`docs/architecture.md`](./docs/architecture.md)**.

---

## Editor integrations

```bash
oz add claude   # CLAUDE.md + Claude Code hooks (.claude/settings.json)
oz add cursor   # Cursor hooks (.cursor/hooks.json)
```

Both install shared hook scripts under `.oz/hooks/`:

| Hook | Role |
|------|------|
| `oz-session-init.sh` | Injects agent routing context at session start |
| `oz-after-edit.sh` | Runs `oz validate` + `oz context build` after `.md` / `.go` edits |
| `oz-pre-commit.sh` | Blocks `git commit` if `oz validate` or `oz audit staleness` fails |

`oz init` can set these up as part of workspace creation.

---

## MCP (Model Context Protocol)

`oz context serve` is a stdio MCP server so clients can call **`query_graph`**, **`get_node`**, **`get_neighbors`**, and **`agent_for_task`** against the current workspace.

Example **`.mcp.json`** snippet:

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

After agents, specs, docs, or indexed code change, rebuild the graph (`oz context build` or the equivalent via MCP) so audits and routing stay accurate.

---

## Documentation map

| Doc | Purpose |
|-----|---------|
| [`AGENTS.md`](./AGENTS.md) | LLM entry — roles and agent definitions |
| [`specs/oz-project-specification.md`](./specs/oz-project-specification.md) | Normative workspace standard |
| [`docs/architecture.md`](./docs/architecture.md) | Implementation: graph, query, audit |
| [`docs/open-items.md`](./docs/open-items.md) | Gaps and follow-ups |

---

## Contributing

Issues and PRs are welcome. Before merging: **tests and vet clean for `code/oz`**, and the **workspace validates** from the repo root. Match your change to the agent in **`AGENTS.md`** and follow that agent’s read-chain.
