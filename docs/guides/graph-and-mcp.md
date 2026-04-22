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

2. Query routing from the CLI.

```bash
oz context query "add a validation rule for AGENT.md sections"
```

3. Start MCP server for editor/tool integration.

```bash
oz context serve
```

4. Configure your client to call `oz context serve` via stdio.
5. Re-run `oz context build` whenever agent/spec/doc/indexed code changes.

## Verify

- `oz context query` returns an agent, confidence, and context blocks.
- MCP client can call `query_graph` and `agent_for_task`.
- Routing changes track file updates after rebuilding graph data.

## Common pitfalls

- Expecting graph-driven routing to update without rebuilding.
- Treating semantic overlay as mandatory for baseline routing.
- Forgetting that stale semantic data is detected by graph hash mismatch.
