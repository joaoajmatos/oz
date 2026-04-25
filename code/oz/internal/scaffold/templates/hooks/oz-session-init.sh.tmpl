#!/usr/bin/env bash
# oz-session-init.sh — UserPromptSubmit / sessionStart hook for oz workspaces
#
# Runs oz context query against the user's actual task prompt so the LLM
# receives task-specific routing (recommended agent + ranked files) before
# touching any tools.  Falls back to a workspace overview when no prompt
# is available (Cursor sessionStart).
#
# Input:  JSON object on stdin  (workspace_roots, prompt, cwd, ...)
# Output: JSON object with additional_context field

set -euo pipefail

# ---------------------------------------------------------------------------
# Resolve oz binary: prefer PATH, fall back to go run in the project dir.
# ---------------------------------------------------------------------------
OZ_BIN=""
if command -v oz &>/dev/null; then
  OZ_BIN="oz"
fi

# ---------------------------------------------------------------------------
# Parse workspace root and prompt from stdin JSON.
# ---------------------------------------------------------------------------
INPUT=$(cat)

if command -v jq &>/dev/null; then
  WORKSPACE_ROOT=$(echo "$INPUT" | jq -r '(.workspace_roots // [])[0] // (.cwd // "")')
  PROMPT=$(echo "$INPUT" | jq -r '.prompt // ""')
elif command -v python3 &>/dev/null; then
  WORKSPACE_ROOT=$(printf '%s' "$INPUT" | python3 -c "
import sys, json
d = json.load(sys.stdin)
r = d.get('workspace_roots', [])
print(r[0] if r else d.get('cwd', ''))
" 2>/dev/null || true)
  PROMPT=$(printf '%s' "$INPUT" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(d.get('prompt', ''))
" 2>/dev/null || true)
else
  WORKSPACE_ROOT=$(echo "$INPUT" | grep -o '"workspace_roots":\s*\["[^"]*"' | grep -o '"[^"]*"$' | tr -d '"' || true)
  PROMPT=""
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
# Sanitize prompt for safe shell argument passing.
# Strip characters that could break single-quoted eval, collapse whitespace,
# and truncate to 200 chars so the query stays focused.
# ---------------------------------------------------------------------------
SAFE_PROMPT=""
if [ -n "$PROMPT" ]; then
  SAFE_PROMPT=$(printf '%s' "$PROMPT" | tr -d "'\"\`\\\\" | tr -s ' \t\n' ' ' | cut -c1-200)
fi

# ---------------------------------------------------------------------------
# Run oz context query — with task prompt when available.
# ---------------------------------------------------------------------------
ROUTING=""
if [ -n "$WORKSPACE_ROOT" ]; then
  cd "$WORKSPACE_ROOT"
  if [ -n "$SAFE_PROMPT" ]; then
    ROUTING=$(eval "$OZ_BIN context query '$SAFE_PROMPT'" 2>/dev/null || true)
  else
    ROUTING=$(eval "$OZ_BIN context query" 2>/dev/null || true)
  fi
  # If empty (installed binary too old), fall back to go run from source.
  if [ -z "$ROUTING" ] && [ -d "code/oz" ]; then
    if [ -n "$SAFE_PROMPT" ]; then
      ROUTING=$(go run -C code/oz . context query "$SAFE_PROMPT" 2>/dev/null || true)
    else
      ROUTING=$(go run -C code/oz . context query 2>/dev/null || true)
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Build a readable routing summary from the JSON result.
# Task-specific query (prompt provided): show agent + ranked files.
# Workspace overview (no prompt): show available agents.
# ---------------------------------------------------------------------------
ROUTING_SUMMARY=""
if [ -n "$ROUTING" ]; then
  if command -v jq &>/dev/null; then
    ROUTING_SUMMARY=$(echo "$ROUTING" | jq -r '
      if (.confidence // 0) > 0 then
        "Recommended agent: " + (.agent // "unknown") +
        " (confidence: " + ((.confidence // 0) * 100 | floor | tostring) + "%)" +
        if (.context_blocks | length) > 0 then
          "\nRead first:\n" + ([.context_blocks[:5][] |
            "  - " + .file + (if .section != "" then " § " + .section else "" end)
          ] | join("\n"))
        else "" end
      elif (.candidate_agents | length) > 0 then
        "Available agents: " + ([.candidate_agents[].name] | join(", "))
      else "" end
    ' 2>/dev/null || true)
  elif command -v python3 &>/dev/null; then
    ROUTING_SUMMARY=$(printf '%s' "$ROUTING" | python3 -c "
import sys, json
d = json.load(sys.stdin)
conf = d.get('confidence') or 0
if conf > 0:
    agent = d.get('agent', 'unknown')
    lines = ['Recommended agent: ' + agent + ' (confidence: ' + str(int(conf * 100)) + '%)']
    blocks = d.get('context_blocks', [])[:5]
    if blocks:
        lines.append('Read first:')
        for b in blocks:
            sec = ' \xc2\xa7 ' + b['section'] if b.get('section') else ''
            lines.append('  - ' + b['file'] + sec)
    print('\n'.join(lines))
else:
    agents = d.get('candidate_agents', [])
    if agents:
        print('Available agents: ' + ', '.join(a['name'] for a in agents))
" 2>/dev/null || true)
  fi
fi

CONTEXT="oz workspace — task context ready.\n"
if [ -n "$ROUTING_SUMMARY" ]; then
  CONTEXT="${CONTEXT}${ROUTING_SUMMARY}\n"
fi
CONTEXT="${CONTEXT}Follow the read-chain in your agent's AGENT.md before starting.\nAfter editing convention files (OZ.md, AGENTS.md, AGENT.md, SKILL.md): run 'oz validate'.\nAfter editing .md or .go files: run 'oz context build' to keep the graph fresh."

# Emit JSON response.
if command -v jq &>/dev/null; then
  jq -n --arg ctx "$CONTEXT" '{"additional_context": $ctx}'
else
  ESCAPED=$(printf '%s' "$CONTEXT" | sed 's/\\/\\\\/g; s/"/\\"/g; s/$/\\n/' | tr -d '\n')
  printf '{"additional_context": "%s"}\n' "$ESCAPED"
fi
