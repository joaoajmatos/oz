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
  <a href="./docs/guides/README.md">Guides</a> ·
  <a href="./specs/oz-project-specification.md">Specification</a> ·
  <a href="./docs/architecture.md">Architecture</a>
</p>

---

## Why oz?

LLM sessions are stateless by default. Every session starts cold, so teams repeatedly pay the cost of re-explaining structure, ownership, and rules.

`oz` fixes that with one open convention plus one Go binary:

- **Convention:** a predictable workspace layout (`AGENTS.md`, `agents/`, `specs/`, `docs/`, `context/`, `skills/`, `rules/`, `notes/`)
- **Tooling:** `oz` commands to scaffold, validate, audit, and query workspace context
- **Interoperability:** works with Claude Code, Cursor, and any setup that can read markdown and/or speak MCP

The result is a workspace that is easier for both humans and LLMs to navigate consistently.

## Who this is for

- Teams using LLMs daily and wanting fewer "cold start" mistakes
- Repos that need explicit ownership and role routing for agentic work
- Multi-repo orgs that want one shared convention workspace (with code repos under `code/`)
- OSS projects that want contributors and agents to ramp up quickly

## What is oz?

**oz** gives any LLM a predictable workspace: explicit **agents**, **specs-first** truth, a machine-checkable layout, and a small **Go binary** (`oz`) for validate, audit, and context graph workflows — without locking you to one editor or model vendor.

It is also a practical **shared workspace environment for teams**: people working at different levels (implementation, maintenance, and specification) operate inside the same conventions, routing model, and validation rules.

For multi-repo teams, `oz` works well as a **meta repository**: keep application repos separate and clean, mount them under `code/` as git submodules, and manage shared docs/agents/rules from one workspace root.

| Layer | What you get |
|--------|----------------|
| **Convention** | [`AGENTS.md`](./AGENTS.md), [`OZ.md`](./OZ.md), `agents/`, `specs/`, `docs/`, `context/`, `skills/`, `rules/`, `notes/`, optional `code/` |
| **CLI** | Single `oz` binary — see **CLI surface** below |

Normative spec: **[`specs/oz-project-specification.md`](./specs/oz-project-specification.md)**. This repo ships the CLI in **`code/oz`** and uses the convention itself.

### CLI surface

| Area | Commands |
|------|----------|
| **Scaffold & health** | `oz init` · `oz validate` · `oz repair` (restore missing defaults without overwriting) |
| **`oz add`** | **Integrations:** `claude`, `cursor` (editor hooks). **Optional packages** (bundled agent + skills in the binary): `maintainer`, `pm` — use **`oz add list`** for the full, current catalog |
| **`oz context`** | `build` (structural `context/graph.json`) · `query` (BM25F routing) · `scoring` + `describe` (tune **`context/scoring.toml`**) · `enrich` / `review` (optional semantic overlay `context/semantic.json`, needs `OPENROUTER_API_KEY`) · **`serve`** (MCP stdio) |
| **`oz audit`** | All checks by default, or `orphans`, `coverage`, `staleness`, `drift`, `graph-summary` |
| **Notes** | `oz crystallize` — classify `notes/` for promotion to `specs/`, `docs/`, or ADRs (optional LLM classifier; can use heuristics only) |
| **Shell** | `oz completion` — generate completion scripts (see **Shell completion** below) |

Run `oz` with no subcommand to print the banner. `oz --help` lists top-level commands; `oz <cmd> --help` for subcommands and flags.

---

## Quick start

### 1) Install `oz` locally

From a clone of this repository:

```bash
cd code/oz
make build          # writes ./bin/oz
# or
go install .
```

Add `oz` to your `PATH`, or invoke the binary by path.

### 2) Create a workspace

```bash
mkdir my-workspace && cd my-workspace
oz init
```

### 3) Validate and build context

```bash
oz validate
oz context build
oz audit
```

If those pass, your workspace is convention-valid, indexed, and ready for agentic work.

### Release binaries

The repository ships a tag-driven release workflow at `.github/workflows/release.yml`.
Pushing a tag like `v0.2.0` builds `oz` for Linux, macOS, and Windows (amd64/arm64),
then publishes a GitHub Release with archives and `checksums.txt`.

### Shell completion

Shell completion is optional. It enables tab-completion for `oz` commands, subcommands, and flags.

From `code/oz`, optional install targets are:

```bash
make install-completion-zsh
make install-completion-bash
make install-completion-fish
make install-completion-powershell
```

What each target does:

- `install-completion-zsh`: writes `_oz` to `~/.zsh/completions/`
- `install-completion-bash`: writes `oz` to `~/.local/share/bash-completion/completions/`
- `install-completion-fish`: writes `oz.fish` to `~/.config/fish/completions/`
- `install-completion-powershell`: writes `oz.ps1` to the current directory

`make install` installs only the binary and does not modify shell startup files.

---

## Feature highlights

- **Single binary:** no daemon, no runtime service dependency
- **Spec-first and checkable:** `oz validate` and `oz audit` keep conventions honest
- **Context graph + retrieval:** `oz context build` + `oz context query` for practical routing
- **MCP-native:** `oz context serve` exposes query tools to editor agents
- **Editor hooks:** shared workflow guardrails for Claude Code and Cursor
- **Optional packages:** install focused agent+skill bundles (`maintainer`, `pm`) as needed

---

## Typical workflow

In active repositories, a simple loop works well:

1. Update code/spec/docs
2. Run `oz validate`
3. Rebuild graph with `oz context build`
4. Run `oz audit` for drift/staleness/orphans checks
5. Use `oz context query "<task>"` (or MCP tools) for routing and scoped context

This keeps the human-readable convention and machine-readable graph aligned as the repo evolves.

---

## Working in an oz workspace

Interactive prompts in `oz init` set the project name, description, code layout (**inline** vs **submodule**), and agents. This scaffolds `AGENTS.md`, `OZ.md`, a starter `README.md`, `agents/*/AGENT.md`, `skills/workspace-management/`, stubs under `docs/`, and (in **inline** mode) `code/README.md`.

`oz repair` recreates any missing default files from the same templates without overwriting files you already changed.

In practice, agentic sessions use `AGENTS.md`, each agent read-chain, and `skills/oz/` to decide when to run `oz` (alongside `go test`, edits, and the rest). Manual human edits can skip the read-chain and apply the same conventions directly.

| Intent | What “done” looks like |
|--------|-------------------------|
| Convention health | **`oz validate`** passes |
| Graph matches the tree | **`context/graph.json`** rebuilt after meaningful edits (`oz context build`) |
| Query / routing config | Optional **`context/scoring.toml`** in sync; use **`oz context scoring validate`** when tuning |
| Workspace health | **`oz audit`** (staleness, drift, orphans, coverage, …) |
| Go implementation | **`go test ./...`** and **`go vet ./...`** clean under **`code/oz`** |

`oz context query` and MCP (`query_graph`, `agent_for_task`, ...) expose the graph for routing and scoped context. ADR: **[`specs/decisions/0004-context-retrieval-ranking.md`](./specs/decisions/0004-context-retrieval-ranking.md)**. **[`docs/architecture.md`](./docs/architecture.md)** covers graph, query, and audit wiring.

---

## Integrations (editors)

These are **not** optional packages — they wire Claude Code or Cursor to shared hook scripts:

```bash
oz add claude   # CLAUDE.md + Claude Code hooks (.claude/settings.json)
oz add cursor   # Cursor hooks (.cursor/hooks.json)
```

Both install the same three hook scripts under **`.oz/hooks/`**; Cursor also writes **`.cursor/hooks.json`**, and Claude adds **`.claude/settings.json`** and **`CLAUDE.md`**:

| Hook | Role |
|------|------|
| `oz-session-init.sh` | Injects agent routing context at session start |
| `oz-after-edit.sh` | Runs `oz validate` + `oz context build` after `.md` / `.go` edits |
| `oz-pre-commit.sh` | Blocks `git commit` if `oz validate` or `oz audit staleness` fails |

`oz init` can install the same hooks during workspace creation. Optional **maintainer** / **pm** packages are installed with **`oz add maintainer`** / **`oz add pm`**.

---

## MCP (Model Context Protocol)

`oz context serve` is a stdio MCP server so clients can call **`query_graph`** (full routing packet), **`get_node`**, **`get_neighbors`**, and **`agent_for_task`** (task → agent + confidence) against the current workspace. See **[`specs/routing-packet.md`](./specs/routing-packet.md)** for the packet shape.

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

## Learn by doing

If you are evaluating oz for adoption, start with the practical guides:

- **Start here:** [`docs/guides/README.md`](./docs/guides/README.md)
- **First walkthrough:** [`docs/guides/first-workspace.md`](./docs/guides/first-workspace.md)
- **Existing repo migration:** [`docs/guides/adopt-existing-repo.md`](./docs/guides/adopt-existing-repo.md)

For convention maintenance work (creating/updating agents, skills, and rules), use the **oz-maintainer** agent in agentic sessions (or follow the same conventions manually). You can add the `maintainer` optional package at any time (also available at `oz init`):

```bash
oz add maintainer
```

Optional packages are small template bundles (extra `agents/`, `skills/`, and related files) shipped inside the `oz` binary. Besides `maintainer`, the `pm` package adds product-management skills and an agent for PRDs, rituals, and discovery workflows. To see every ID and one-line description:

```bash
oz add list
```

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

See **[`CONTRIBUTING.md`](./CONTRIBUTING.md)** (public-domain spirit, practical checklist). In short: **`go test` / `go vet`** clean under **`code/oz`**, and **`oz validate`** passes from the repo root. In agentic sessions, use **`AGENTS.md`** for intent-based routing; for manual edits, follow the same conventions directly.

## License

This project is released under **The Unlicense**. See [`UNLICENSE`](./UNLICENSE).
