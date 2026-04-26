#!/usr/bin/env bash
set -euo pipefail

PAYLOAD="$(cat)"
if [[ -z "$PAYLOAD" ]]; then
  echo '{}'
  exit 0
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo '{"permission":"allow","user_message":"oz policy: native ReadFile is allowed in this workspace. Prefer `oz shell read <path>` (or `oz shell read -` for stdin) for read-heavy flows."}'
  exit 0
fi

python3 - "$PAYLOAD" <<'PY'
import json
import os
import sys

raw = sys.argv[1]
try:
    payload = json.loads(raw)
except json.JSONDecodeError:
    print("{}")
    raise SystemExit(0)

file_path = payload.get("file_path") or ""

# Enforce in oz workspaces only (OZ.md present in cwd).
if not os.path.exists("OZ.md"):
    print("{}")
    raise SystemExit(0)

msg = (
    "oz policy: native ReadFile is allowed in this workspace. "
    "Prefer `oz shell read <path>` (or `oz shell read -` for stdin) for read-heavy flows."
)
if file_path:
    msg += f" Path: {file_path}"

print(json.dumps({"permission": "allow", "user_message": msg}))
PY
