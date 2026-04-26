#!/usr/bin/env bash
set -euo pipefail

# Warn-only policy: never block reads.
msg="oz policy: native Read is allowed in this workspace. Prefer `oz shell read <path>` (or `oz shell read -` for stdin) for read-heavy flows."

if command -v jq >/dev/null 2>&1; then
  jq -nc --arg m "$msg" '{continue:true, suppressOutput:false, decision:"approve", reason:$m}'
else
  printf '%s
' '{"continue":true,"suppressOutput":false,"decision":"approve","reason":"'"$msg"'"}'
fi
