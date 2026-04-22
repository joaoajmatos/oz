# First Workspace

Create a new `oz` workspace and verify the baseline convention flow.

## Goal

By the end, you have a new workspace that passes `oz validate`, has a fresh graph, and is ready for agent-routed work.

## Preconditions

- `oz` is installed and available in your `PATH`.
- You can run commands in a clean directory.

## Steps

1. Create and initialize a workspace.

```bash
mkdir my-workspace && cd my-workspace
oz init
```

1. If you selected a maintainer agent during init, use it for convention work (creating/updating agents, skills, and rules).
1. If your workspace does not include a maintainer agent, add it:

```bash
oz add maintainer
```

1. Open `AGENTS.md` and identify the agent that matches your task.
1. Open that agent's `AGENT.md` and follow its read-chain before making edits.
1. Validate convention health.

```bash
oz validate
```

1. Build structural context.

```bash
oz context build
```

## Verify

- `oz validate` exits successfully.
- `context/graph.json` exists after `oz context build`.
- `AGENTS.md`, `OZ.md`, and at least one `agents/*/AGENT.md` exist.
- `agents/maintainer/AGENT.md` exists when you opted in during init or ran `oz add maintainer`.

## Common pitfalls

- Skipping the agent read-chain and starting edits immediately.
- Forgetting to rebuild context after changing `docs/`, `specs/`, `agents/`, or indexed code.
- Treating `notes/` as authoritative instead of crystallizing important decisions into `specs/` or `docs/`.
