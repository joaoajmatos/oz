#!/usr/bin/env bash
# oz-edit-allowlist-claude.sh — Claude Code PreToolUse(Edit|Write) hook
#
# Adds the target file path to a temp allowlist so the subsequent
# PreToolUse(Read) precondition check (required by Edit/Write) is let through
# by oz-read-policy-claude.sh instead of being denied.
#
# The allowlist entry is consumed (one-shot) by oz-read-policy-claude.sh.

set -euo pipefail

ALLOWLIST="/tmp/oz-edit-allowlist"

INPUT="${CLAUDE_TOOL_INPUT:-$(cat)}"
if [[ -z "$INPUT" ]]; then
  exit 0
fi

if command -v jq &>/dev/null; then
  FILE_PATH=$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // .file_path // empty' 2>/dev/null || true)
else
  FILE_PATH=$(printf '%s' "$INPUT" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(d.get('tool_input', {}).get('file_path') or d.get('file_path') or '')
" 2>/dev/null || true)
fi

if [[ -z "$FILE_PATH" ]]; then
  exit 0
fi

NORMALIZED=$(python3 -c "import os,sys; print(os.path.realpath(sys.argv[1]))" "$FILE_PATH" 2>/dev/null || true)
if [[ -z "$NORMALIZED" ]]; then
  exit 0
fi

printf '%s\n' "$NORMALIZED" >> "$ALLOWLIST"
