# oz

Open workspace convention and toolset for LLM-first development

## Getting started

<!-- Add setup instructions here. -->

## MCP server

`oz context serve` exposes an MCP (Model Context Protocol) stdio server that
lets Claude Code, Cursor, and other MCP-compatible clients query the workspace
context graph directly.

Add this to your `.mcp.json` (project-level) or `~/.claude/mcp.json` (global):

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

Available tools:

| Tool | Description |
|------|-------------|
| `query_graph` | Route a task to the best-matching agent (full routing packet) |
| `get_node` | Retrieve a structural graph node by ID |
| `get_neighbors` | List nodes adjacent to a given node |
| `agent_for_task` | Shorthand: task → agent name + confidence only |

Run `oz context build` in your workspace root before starting the server so
the graph is up to date.

## Development

This workspace follows the [oz convention](https://github.com/oz-tools/oz).
See `AGENTS.md` for workspace structure and agent definitions.
