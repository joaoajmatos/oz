# Graph and MCP

Use `oz` structural context and MCP tools to route tasks with low token overhead.

## Goal

Understand when to rebuild graph data, how to query routing, and how to expose the same capabilities through MCP.

## Preconditions

- Workspace initialized and valid.
- `oz context build` has run at least once.

## Steps

1. Rebuild graph after meaningful edits.

```bash
oz context build
```

1. Query routing from the CLI.

```bash
oz context query "add a validation rule for AGENT.md sections"
```

1. Use the right command shape for the question:
   - use `oz context query` for normal routing + context packets,
   - use `oz context query --raw` when debugging routing/retrieval scoring,
   - use `oz context query --include-notes` for one-off note-tier retrieval.

```bash
oz context query --raw "add a validation rule for AGENT.md sections"
```

1. Start MCP server for editor/tool integration.

```bash
oz context serve
```

1. Configure your client to call `oz context serve` via stdio.
1. Choose MCP tools by token budget and task depth:
   - `agent_for_task` for lightweight task -> owner decisions,
   - `query_graph` for full routing packet (agent, confidence, context blocks),
   - `get_node` for one known node by ID,
   - `get_neighbors` for adjacency exploration from a known node.
1. Re-run `oz context build` whenever agent/spec/doc/indexed code changes.

## Verify

- `oz context query` returns an agent, confidence, and context blocks.
- MCP client can call all four tools (`query_graph`, `agent_for_task`, `get_node`, `get_neighbors`).
- Routing changes track file updates after rebuilding graph data.

## Common pitfalls

- Expecting graph-driven routing to update without rebuilding.
- Always using `query_graph` even when `agent_for_task` is enough.
- Treating semantic overlay as mandatory for baseline routing.
- Forgetting that stale semantic data is detected by graph hash mismatch.
