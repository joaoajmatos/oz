#!/usr/bin/env bash
# oz-shell-rewrite-claude.sh — Claude Code PreToolUse(Bash) hook

set -euo pipefail

INPUT="${CLAUDE_TOOL_INPUT:-}"
if [[ -z "$INPUT" ]]; then
  exit 0
fi

if command -v jq &>/dev/null; then
  CMD=$(printf '%s' "$INPUT" | jq -r '.tool_input.command // .command // empty' 2>/dev/null || true)
else
  CMD=$(printf '%s' "$INPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('tool_input',{}).get('command','') or d.get('command',''))" 2>/dev/null || true)
fi
if [[ -z "$CMD" ]]; then
  exit 0
fi

REWRITTEN=$(oz shell rewrite "$CMD" 2>/dev/null) || exit 0
if [[ -z "$REWRITTEN" || "$REWRITTEN" == "$CMD" ]]; then
  exit 0
fi

if command -v jq &>/dev/null; then
  jq -n --arg cmd "$REWRITTEN" '{
    "hookSpecificOutput": {
      "hookEventName": "PreToolUse",
      "permissionDecision": "allow",
      "permissionDecisionReason": "oz shell rewrite",
      "updatedInput": { "command": $cmd }
    }
  }'
else
  ESCAPED=$(printf '%s' "$REWRITTEN" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read())[1:-1])")
  printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","permissionDecisionReason":"oz shell rewrite","updatedInput":{"command":"%s"}}}\n' "$ESCAPED"
fi
