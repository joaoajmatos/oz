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
  <a href="./docs/guides/README.md">Guides</a> ·
  <a href="./specs/oz-project-specification.md">Specification</a> ·
  <a href="./docs/architecture.md">Architecture</a>
</p>

---

## What is oz?

**oz** gives any LLM a predictable workspace: explicit **agents**, **specs-first** truth, a machine-checkable layout, and a small **Go binary** (`oz`) for validate, audit, and context graph workflows — without locking you to one editor or model vendor.

| Layer | What you get |
|--------|----------------|
| **Convention** | [`AGENTS.md`](./AGENTS.md), [`OZ.md`](./OZ.md), `agents/`, `specs/`, `docs/`, `context/`, `skills/`, `rules/`, `notes/`, optional `code/` |
| **CLI** | `oz init`, `oz validate`, `oz repair`, `oz context` (build / query / enrich / review / **serve** for MCP), `oz audit`, and more as the project grows |

Normative spec: **[`specs/oz-project-specification.md`](./specs/oz-project-specification.md)**. This repo ships the CLI in **`code/oz`** and uses the convention itself.

---

## Install

From a clone of this repository:

```bash
cd code/oz
make build          # writes ./bin/oz
# or
go install .
```

Add `oz` to your `PATH`, or invoke the binary by path.

---

## Quick start: new workspace

```bash
mkdir my-workspace && cd my-workspace
oz init
```

Interactive prompts set the project name, description, code layout (**inline** vs **submodule**), and agents. That scaffolds `AGENTS.md`, `OZ.md`, a starter **README.md**, `agents/*/AGENT.md`, `skills/workspace-management/`, stubs under `docs/`, and (in **inline** mode) `code/README.md`.

**`oz repair`** recreates any missing default files from the same templates **without overwriting** files you already changed.

---

## Learn by doing

If you are evaluating oz for adoption, start with the practical guides:

- **Start here:** [`docs/guides/README.md`](./docs/guides/README.md)
- **First walkthrough:** [`docs/guides/first-workspace.md`](./docs/guides/first-workspace.md)
- **Existing repo migration:** [`docs/guides/adopt-existing-repo.md`](./docs/guides/adopt-existing-repo.md)

For convention maintenance work (creating/updating agents, skills, and rules), use a maintainer agent. You can include one at `oz init` time, or add it later:

```bash
oz add maintainer
```

---

## Working in an oz workspace

In practice, models use **`AGENTS.md`**, each agent read-chain, and **`skills/oz/`** to decide when to run **`oz`** (alongside `go test`, edits, and the rest).

| Intent | What “done” looks like |
|--------|-------------------------|
| Convention health | **`oz validate`** passes |
| Graph matches the tree | **`context/graph.json`** rebuilt after meaningful edits (`oz context build`) |
| Workspace health | **`oz audit`** (staleness, drift, orphans, …) |
| Go implementation | **`go test ./...`** and **`go vet ./...`** clean under **`code/oz`** |

`oz context query` and MCP (**`query_graph`**, **`agent_for_task`**, …) expose the graph for routing and scoped context. **[`docs/architecture.md`](./docs/architecture.md)** covers graph, query, and audit wiring.

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

`oz init` can install the same hooks during workspace creation.

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

After agents, specs, docs, or indexed code change, run **`oz context build`** so audits and routing match the tree.

---

## Documentation map

| Doc | Purpose |
|-----|---------|
| [`AGENTS.md`](./AGENTS.md) | LLM entry — roles and agent definitions |
| [`specs/oz-project-specification.md`](./specs/oz-project-specification.md) | Normative workspace standard |
| [`docs/architecture.md`](./docs/architecture.md) | Implementation: graph, query, audit |
| [`docs/guides/README.md`](./docs/guides/README.md) | Practical adoption guides and walkthroughs |
| [`docs/open-items.md`](./docs/open-items.md) | Gaps and follow-ups |

---

## Contributing

Issues and PRs are welcome. Before merging: **`go test` / `go vet`** clean under **`code/oz`**, and **`oz validate`** passes from the repo root. Use **`AGENTS.md`** to pick the agent whose read-chain fits your change.
