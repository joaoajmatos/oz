#!/usr/bin/env bash
# oz-shell-rewrite-cursor.sh — Cursor beforeShellExecution hook

set -euo pipefail

INPUT=$(cat || true)
if [[ -z "$INPUT" ]]; then
  echo '{}'
  exit 0
fi

if command -v jq &>/dev/null; then
  CMD=$(printf '%s' "$INPUT" | jq -r '.command // .tool_input.command // empty' 2>/dev/null || true)
else
  CMD=$(printf '%s' "$INPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('command') or d.get('tool_input',{}).get('command',''))" 2>/dev/null || true)
fi
if [[ -z "$CMD" ]]; then
  echo '{}'
  exit 0
fi

REWRITTEN=$(oz shell rewrite "$CMD" 2>/dev/null) || { echo '{}'; exit 0; }
if [[ -z "$REWRITTEN" || "$REWRITTEN" == "$CMD" ]]; then
  echo '{}'
  exit 0
fi

if command -v jq &>/dev/null; then
  jq -n --arg cmd "$REWRITTEN" '{
    "permission": "allow",
    "updated_input": { "command": $cmd }
  }'
else
  ESCAPED=$(printf '%s' "$REWRITTEN" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read())[1:-1])")
  printf '{"permission":"allow","updated_input":{"command":"%s"}}\n' "$ESCAPED"
fi
