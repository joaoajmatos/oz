---
name: oz-shell
description: >-
  Uses oz shell compression commands and hook wiring (`oz shell read`, `oz shell run`, `oz shell rewrite`, `oz shell gain`) so LLM shell usage is token-efficient while preserving command semantics and exit codes.
triggers:
  - oz shell read
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

- Read source files or stdin in a language-aware, token-efficient way.
- Execute command-heavy loops where compact output improves LLM throughput.
- Debug or verify transparent interception behavior (`suggest` vs `rewrite`).
- Install or validate Claude/Cursor hook wiring for shell rewrite behavior.
- Measure practical savings with `oz shell gain`.

Do not use this skill for non-shell workflows; use `skills/oz/` for broader workspace validation, context graph, audit, and MCP flows.

---

## Steps

1. **Use `oz shell read` instead of `cat` when reading code files.**
   - Run `oz shell read <file...>` to read files through the language-aware compaction pipeline.
   - Use `oz shell read -` to read from stdin (e.g. piped output from another command).
   - Pass `--ultra-compact` for signatures-only output on Go files (maximum compression).
   - Pass `--max-lines N` or `--tail-lines N` to window the output before compression.
   - Pass `--line-numbers` when the downstream task needs line references.
   - The hook transparently rewrites `cat`, `head`, and `tail` to `oz shell read` when active, but calling `oz shell read` directly is always preferred and guaranteed.

2. **Use explicit wrapper mode for command output.**
   - Run `oz shell run -- <command...>` for deterministic compact command output.
   - Use `--mode raw` only when full uncompressed output is needed.
   - Use `--json` when downstream logic needs structured output fields.

3. **Use rewrite mode for transparent interception paths.**
   - Run `oz shell rewrite "<command string>"` when emulating hook behavior.
   - Interpret exits: `0` rewritten, `1` passthrough, `2` excluded or disabled.
   - For temporary bypasses, pass repeatable exclusions: `--exclude git --exclude make`.

4. **Verify hook-based interception wiring when required.**
   - Ensure `.oz/hooks/oz-shell-rewrite.sh` exists in the workspace.
   - Ensure `.claude/settings.json` includes the `PreToolUse` matcher `Bash` entry for `.oz/hooks/oz-shell-rewrite.sh`.
   - Treat hook failures as fail-open passthrough unless policy says otherwise.

5. **Track impact.**
   - Run `oz shell gain` (and `--json` when needed) to review token savings and reduction trends.
   - Use gain output to decide whether command families should stay in wrapper mode.

6. **Close the loop on workspace artifacts.**
   - After changing shell behavior, run `oz validate`.
   - If markdown or Go changed, run `oz context build` so graph context stays fresh.
