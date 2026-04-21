#!/usr/bin/env bash
# oz-after-edit.sh — afterFileEdit / PostToolUse(Edit|Write) hook for oz workspaces
#
# After any file edit:
#   1. Skips non-.md and non-.go files early.
#   2. Runs oz validate; surfaces failures as additional_context (non-blocking).
#   3. Runs oz context build --quiet to keep graph fresh.
#
# Input:  JSON object on stdin (file_path, edits[], ...)
# Output: JSON object with optional additional_context field

set -euo pipefail

INPUT=$(cat)

# ---------------------------------------------------------------------------
# Extract file path.
# ---------------------------------------------------------------------------
if command -v jq &>/dev/null; then
  FILE_PATH=$(echo "$INPUT" | jq -r '.file_path // ""')
  WORKSPACE_ROOT=$(echo "$INPUT" | jq -r '(.workspace_roots // [])[0] // ""')
else
  FILE_PATH=$(echo "$INPUT" | grep -o '"file_path":"[^"]*"' | cut -d'"' -f4 || true)
  WORKSPACE_ROOT=$(echo "$INPUT" | grep -o '"workspace_roots":\s*\["[^"]*"' | grep -o '"[^"]*"$' | tr -d '"' || true)
fi

# ---------------------------------------------------------------------------
# Only act on .md and .go files.
# ---------------------------------------------------------------------------
case "$FILE_PATH" in
  *.md|*.go) ;;
  *) echo '{}'; exit 0 ;;
esac

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
  echo '{}'
  exit 0
fi

# Change to workspace root for oz commands.
if [ -n "$WORKSPACE_ROOT" ]; then
  cd "$WORKSPACE_ROOT"
fi

# ---------------------------------------------------------------------------
# Run oz validate; capture output + exit code.
# ---------------------------------------------------------------------------
VALIDATE_OUT=""
VALIDATE_OK=true
if VALIDATE_OUT=$(eval "$OZ_BIN validate" 2>&1); then
  VALIDATE_OK=true
else
  VALIDATE_OK=false
fi

# ---------------------------------------------------------------------------
# Rebuild context graph (always, quiet).
# ---------------------------------------------------------------------------
eval "$OZ_BIN context build --quiet" 2>/dev/null || true

# ---------------------------------------------------------------------------
# Emit response.
# ---------------------------------------------------------------------------
if [ "$VALIDATE_OK" = false ] && [ -n "$VALIDATE_OUT" ]; then
  CONTEXT="oz validate failed after edit:\n${VALIDATE_OUT}\nFix convention errors before committing."
  if command -v jq &>/dev/null; then
    jq -n --arg ctx "$CONTEXT" '{"additional_context": $ctx}'
  else
    ESCAPED=$(printf '%s' "$CONTEXT" | sed 's/\\/\\\\/g; s/"/\\"/g')
    printf '{"additional_context": "%s"}\n' "$ESCAPED"
  fi
else
  echo '{}'
fi
