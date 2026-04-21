#!/usr/bin/env bash
# oz-pre-commit.sh — beforeShellExecution / PreToolUse(Bash) hook for oz workspaces
#
# Gates 'git commit' commands:
#   1. Passes through immediately if the command is not a git commit.
#   2. Runs oz validate; blocks commit on failure (exit 2).
#   3. Runs oz audit staleness; blocks commit on error-level findings.
#
# Input:  JSON object on stdin (command, cwd, workspace_roots, ...)
# Output: JSON object with permission field ("allow" | "deny")
#         Cursor: uses "allow"/"deny" — exit 2 also blocks.
#         Claude Code: exit 2 blocks; stdout JSON is surfaced to the agent.

set -euo pipefail

INPUT=$(cat)

# ---------------------------------------------------------------------------
# Extract the command string.
# ---------------------------------------------------------------------------
if command -v jq &>/dev/null; then
  COMMAND=$(echo "$INPUT" | jq -r '.command // .tool_input.command // ""')
  WORKSPACE_ROOT=$(echo "$INPUT" | jq -r '(.workspace_roots // [])[0] // (.cwd // "")')
else
  COMMAND=$(echo "$INPUT" | grep -o '"command":"[^"]*"' | head -1 | cut -d'"' -f4 || true)
  WORKSPACE_ROOT=$(echo "$INPUT" | grep -o '"cwd":"[^"]*"' | cut -d'"' -f4 || true)
fi

# ---------------------------------------------------------------------------
# Only gate git commit commands.
# ---------------------------------------------------------------------------
if ! echo "$COMMAND" | grep -q 'git commit'; then
  echo '{"permission": "allow"}'
  exit 0
fi

# ---------------------------------------------------------------------------
# Resolve oz binary.
# ---------------------------------------------------------------------------
OZ_BIN=""
if command -v oz &>/dev/null; then
  OZ_BIN="oz"
elif [ -n "$WORKSPACE_ROOT" ] && [ -d "$WORKSPACE_ROOT/code/oz" ]; then
  OZ_BIN="go run -C $WORKSPACE_ROOT/code/oz ."
fi

if [ -z "$OZ_BIN" ]; then
  # Can't check — allow through.
  echo '{"permission": "allow"}'
  exit 0
fi

# Change to workspace root.
if [ -n "$WORKSPACE_ROOT" ]; then
  cd "$WORKSPACE_ROOT"
fi

# ---------------------------------------------------------------------------
# oz validate gate.
# ---------------------------------------------------------------------------
VALIDATE_OUT=""
if ! VALIDATE_OUT=$(eval "$OZ_BIN validate" 2>&1); then
  MSG="oz validate failed — fix convention errors before committing.\n\n${VALIDATE_OUT}"
  if command -v jq &>/dev/null; then
    jq -n --arg msg "$MSG" '{
      "permission": "deny",
      "user_message": $msg,
      "agent_message": $msg
    }'
  else
    ESCAPED=$(printf '%s' "$MSG" | sed 's/\\/\\\\/g; s/"/\\"/g')
    printf '{"permission":"deny","user_message":"%s","agent_message":"%s"}\n' "$ESCAPED" "$ESCAPED"
  fi
  exit 2
fi

# ---------------------------------------------------------------------------
# oz audit staleness gate.
# ---------------------------------------------------------------------------
AUDIT_OUT=""
if ! AUDIT_OUT=$(eval "$OZ_BIN audit staleness --exit-on=error" 2>&1); then
  MSG="oz audit staleness found errors — resolve before committing.\n\n${AUDIT_OUT}"
  if command -v jq &>/dev/null; then
    jq -n --arg msg "$MSG" '{
      "permission": "deny",
      "user_message": $msg,
      "agent_message": $msg
    }'
  else
    ESCAPED=$(printf '%s' "$MSG" | sed 's/\\/\\\\/g; s/"/\\"/g')
    printf '{"permission":"deny","user_message":"%s","agent_message":"%s"}\n' "$ESCAPED" "$ESCAPED"
  fi
  exit 2
fi

echo '{"permission": "allow"}'
