---
name: oz-shell
description: >-
  Uses oz shell compression commands and hook wiring (`oz shell run`, `oz shell rewrite`, `oz shell gain`) so LLM shell usage is token-efficient while preserving command semantics and exit codes.
triggers:
  - oz shell run
  - oz shell rewrite
  - oz shell gain
  - shell compression
  - transparent interception
  - PreToolUse Bash
  - command rewrite hook
---

# Skill: oz-shell

> Use oz shell compression intentionally (not only passively via hooks) for lower-token shell execution with deterministic output and preserved failure signal.

## When to invoke

Invoke this skill when you need to:

- Execute command-heavy loops where compact output improves LLM throughput.
- Debug or verify transparent interception behavior (`suggest` vs `rewrite`).
- Install or validate Claude/Cursor hook wiring for shell rewrite behavior.
- Measure practical savings with `oz shell gain`.

Do not use this skill for non-shell workflows; use `skills/oz/` for broader workspace validation, context graph, audit, and MCP flows.

---

## Steps

1. **Use explicit wrapper mode when you control the command call site.**
   - Run `oz shell run -- <command...>` for deterministic compact output.
   - Use `--mode raw` only when full uncompressed output is needed.
   - Use `--json` when downstream logic needs structured output fields.

2. **Use rewrite mode for transparent interception paths.**
   - Run `oz shell rewrite "<command string>"` when emulating hook behavior.
   - Interpret exits: `0` rewritten, `1` passthrough, `2` excluded or disabled.
   - For temporary bypasses, pass repeatable exclusions: `--exclude git --exclude make`.

3. **Verify hook-based interception wiring when required.**
   - Ensure `.oz/hooks/oz-shell-rewrite.sh` exists in the workspace.
   - Ensure `.claude/settings.json` includes the `PreToolUse` matcher `Bash` entry for `.oz/hooks/oz-shell-rewrite.sh`.
   - Treat hook failures as fail-open passthrough unless policy says otherwise.

4. **Track impact.**
   - Run `oz shell gain` (and `--json` when needed) to review token savings and reduction trends.
   - Use gain output to decide whether command families should stay in wrapper mode.

5. **Close the loop on workspace artifacts.**
   - After changing shell behavior, run `oz validate`.
   - If markdown or Go changed, run `oz context build` so graph context stays fresh.
