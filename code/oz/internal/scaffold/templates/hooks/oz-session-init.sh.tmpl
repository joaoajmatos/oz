#!/usr/bin/env bash
# oz-session-init.sh — sessionStart hook for oz workspaces
#
# Injects oz workspace routing context at the start of every agent session.
# Called by Cursor (sessionStart) and Claude Code (UserPromptSubmit or session hooks).
#
# Input:  JSON object on stdin (workspace_roots, prompt, etc.)
# Output: JSON object with additional_context field

set -euo pipefail

# ---------------------------------------------------------------------------
# Resolve oz binary: prefer PATH, fall back to go run in the project dir.
# Resolution happens again after we know WORKSPACE_ROOT (see below).
# ---------------------------------------------------------------------------
OZ_BIN=""
if command -v oz &>/dev/null; then
  OZ_BIN="oz"
fi

# ---------------------------------------------------------------------------
# Parse workspace root from stdin JSON.
# ---------------------------------------------------------------------------
INPUT=$(cat)

# Detect jq availability.
if command -v jq &>/dev/null; then
  WORKSPACE_ROOT=$(echo "$INPUT" | jq -r '(.workspace_roots // [])[0] // ""')
else
  # Fallback: crude extraction without jq.
  WORKSPACE_ROOT=$(echo "$INPUT" | grep -o '"workspace_roots":\s*\["[^"]*"' | grep -o '"[^"]*"$' | tr -d '"' || true)
fi

# If oz is not on PATH, try go run from workspace.
if [ -z "$OZ_BIN" ] && [ -n "$WORKSPACE_ROOT" ] && [ -d "$WORKSPACE_ROOT/code/oz" ]; then
  OZ_BIN="go run -C $WORKSPACE_ROOT/code/oz ."
fi

# No oz available — emit minimal context and exit cleanly.
if [ -z "$OZ_BIN" ]; then
  cat <<'EOF'
{"additional_context": "oz workspace detected. Read AGENTS.md and follow the read-chain before starting any task. Run 'oz validate' after convention file changes."}
EOF
  exit 0
fi

# ---------------------------------------------------------------------------
# Run oz context query (no args = workspace overview).
# ---------------------------------------------------------------------------
ROUTING=""
if [ -n "$WORKSPACE_ROOT" ]; then
  cd "$WORKSPACE_ROOT"
  ROUTING=$(eval "$OZ_BIN context query" 2>/dev/null || true)
  # If empty (e.g. installed binary too old), fall back to go run from source.
  if [ -z "$ROUTING" ] && [ -d "code/oz" ]; then
    ROUTING=$(go run -C code/oz . context query 2>/dev/null || true)
  fi
fi

# Build a readable summary from the routing result.
AGENTS_SUMMARY=""
if command -v jq &>/dev/null && [ -n "$ROUTING" ]; then
  AGENTS_SUMMARY=$(echo "$ROUTING" | jq -r '
    if .candidate_agents then
      "Available agents: " + ([.candidate_agents[].name] | join(", "))
    else
      ""
    end
  ' 2>/dev/null || true)
fi

CONTEXT="oz workspace detected.\n"
if [ -n "$AGENTS_SUMMARY" ]; then
  CONTEXT="${CONTEXT}${AGENTS_SUMMARY}.\n"
fi
CONTEXT="${CONTEXT}Before starting: read AGENTS.md, find your agent role, and follow its read-chain.\nAfter editing convention files (OZ.md, AGENTS.md, AGENT.md, SKILL.md): run 'oz validate'.\nAfter editing .md or .go files: run 'oz context build' to keep the graph fresh."

# Emit JSON response.
if command -v jq &>/dev/null; then
  jq -n --arg ctx "$CONTEXT" '{"additional_context": $ctx}'
else
  # Manual JSON encode (escape backslashes and quotes).
  ESCAPED=$(printf '%s' "$CONTEXT" | sed 's/\\/\\\\/g; s/"/\\"/g; s/$/\\n/' | tr -d '\n')
  printf '{"additional_context": "%s"}\n' "$ESCAPED"
fi
