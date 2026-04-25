#!/usr/bin/env bash
# oz-shell-rewrite.sh — PreToolUse(Bash) hook: rewrites commands through oz shell compression

set -euo pipefail

INPUT="${CLAUDE_TOOL_INPUT:-}"
if [[ -z "$INPUT" ]]; then
  exit 0
fi

CMD=$(printf '%s' "$INPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('tool_input',{}).get('command',''))" 2>/dev/null || true)
if [[ -z "$CMD" ]]; then
  exit 0
fi

REWRITTEN=$(oz shell rewrite "$CMD" 2>/dev/null) || exit 0

printf '{"updatedInput":{"command":"%s"}}' \
  "$(printf '%s' "$REWRITTEN" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read())[1:-1])")"
