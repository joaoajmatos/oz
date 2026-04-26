#!/usr/bin/env bash
# oz-edit-allowlist-cursor.sh — Cursor beforeEditFile hook
#
# Adds the target file path to a temp allowlist so a subsequent
# beforeReadFile check can let through the Edit read precondition.
# Mirrors oz-edit-allowlist-claude.sh for Cursor environments.

set -euo pipefail

ALLOWLIST="/tmp/oz-edit-allowlist"

PAYLOAD="${1:-$(cat)}"
if [[ -z "$PAYLOAD" ]]; then
  echo '{}'
  exit 0
fi

if command -v jq &>/dev/null; then
  FILE_PATH=$(printf '%s' "$PAYLOAD" | jq -r '.file_path // empty' 2>/dev/null || true)
else
  FILE_PATH=$(printf '%s' "$PAYLOAD" | python3 -c "
import sys, json
d = json.loads(sys.argv[1]) if len(sys.argv) > 1 else json.load(sys.stdin)
print(d.get('file_path') or '')
" 2>/dev/null || true)
fi

if [[ -z "$FILE_PATH" ]]; then
  echo '{}'
  exit 0
fi

NORMALIZED=$(python3 -c "import os,sys; print(os.path.realpath(sys.argv[1]))" "$FILE_PATH" 2>/dev/null || true)
if [[ -n "$NORMALIZED" ]]; then
  printf '%s\n' "$NORMALIZED" >> "$ALLOWLIST"
fi

echo '{}'
