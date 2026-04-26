package scaffold_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/scaffold"
)

func TestCursorRewriteHook_Contracts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := scaffold.WriteCursorHooks(root); err != nil {
		t.Fatalf("WriteCursorHooks: %v", err)
	}

	pathEnv := installFakeOz(t, map[string]string{
		"git status": "oz shell run -- git status",
	})
	script := filepath.Join(root, ".oz", "hooks", "oz-shell-rewrite-cursor.sh")

	t.Run("rewrite emits updated_input", func(t *testing.T) {
		t.Parallel()
		out, err := runHook(t, script, `{"command":"git status"}`, map[string]string{
			"PATH": pathEnv,
		})
		if err != nil {
			t.Fatalf("run hook: %v", err)
		}
		for _, want := range []string{
			`"permission": "allow"`,
			`"updated_input"`,
			`"command": "oz shell run -- git status"`,
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("expected %q in output:\n%s", want, out)
			}
		}
	})

	t.Run("no rewrite emits empty json", func(t *testing.T) {
		t.Parallel()
		out, err := runHook(t, script, `{"command":"echo hi"}`, map[string]string{
			"PATH": pathEnv,
		})
		if err != nil {
			t.Fatalf("run hook: %v", err)
		}
		if strings.TrimSpace(out) != "{}" {
			t.Fatalf("expected {}, got %q", strings.TrimSpace(out))
		}
	})
}

func TestClaudeRewriteHook_Contracts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}
	if err := scaffold.WriteClaudeHooks(root); err != nil {
		t.Fatalf("WriteClaudeHooks: %v", err)
	}

	pathEnv := installFakeOz(t, map[string]string{
		"git status": "oz shell run -- git status",
	})
	script := filepath.Join(root, ".oz", "hooks", "oz-shell-rewrite-claude.sh")

	t.Run("rewrite emits hookSpecificOutput", func(t *testing.T) {
		t.Parallel()
		out, err := runHook(t, script, "", map[string]string{
			"PATH":              pathEnv,
			"CLAUDE_TOOL_INPUT": `{"tool_input":{"command":"git status"}}`,
		})
		if err != nil {
			t.Fatalf("run hook: %v", err)
		}
		for _, want := range []string{
			`"hookSpecificOutput"`,
			`"hookEventName": "PreToolUse"`,
			`"permissionDecision": "allow"`,
			`"updatedInput"`,
			`"command": "oz shell run -- git status"`,
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("expected %q in output:\n%s", want, out)
			}
		}
	})

	t.Run("no rewrite emits nothing", func(t *testing.T) {
		t.Parallel()
		out, err := runHook(t, script, "", map[string]string{
			"PATH":              pathEnv,
			"CLAUDE_TOOL_INPUT": `{"tool_input":{"command":"echo hi"}}`,
		})
		if err != nil {
			t.Fatalf("run hook: %v", err)
		}
		if strings.TrimSpace(out) != "" {
			t.Fatalf("expected empty output, got %q", strings.TrimSpace(out))
		}
	})
}

func TestRewriteShim_DispatchesByProvider(t *testing.T) {
	t.Parallel()

	t.Run("cursor path via stdin", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		if err := scaffold.WriteCursorHooks(root); err != nil {
			t.Fatalf("WriteCursorHooks: %v", err)
		}

		pathEnv := installFakeOz(t, map[string]string{
			"git status": "oz shell run -- git status",
		})
		script := filepath.Join(root, ".oz", "hooks", "oz-shell-rewrite.sh")

		out, err := runHook(t, script, `{"command":"git status"}`, map[string]string{
			"PATH": pathEnv,
		})
		if err != nil {
			t.Fatalf("run shim: %v", err)
		}
		for _, want := range []string{
			`"permission": "allow"`,
			`"updated_input"`,
			`"command": "oz shell run -- git status"`,
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("expected %q in output:\n%s", want, out)
			}
		}
	})

	t.Run("claude path via env payload", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
			t.Fatalf("mkdir .claude: %v", err)
		}
		if err := scaffold.WriteClaudeHooks(root); err != nil {
			t.Fatalf("WriteClaudeHooks: %v", err)
		}

		pathEnv := installFakeOz(t, map[string]string{
			"git status": "oz shell run -- git status",
		})
		script := filepath.Join(root, ".oz", "hooks", "oz-shell-rewrite.sh")

		out, err := runHook(t, script, "", map[string]string{
			"PATH":              pathEnv,
			"CLAUDE_TOOL_INPUT": `{"tool_input":{"command":"git status"}}`,
		})
		if err != nil {
			t.Fatalf("run shim: %v", err)
		}
		for _, want := range []string{
			`"hookSpecificOutput"`,
			`"hookEventName": "PreToolUse"`,
			`"permissionDecision": "allow"`,
			`"updatedInput"`,
			`"command": "oz shell run -- git status"`,
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("expected %q in output:\n%s", want, out)
			}
		}
	})
}

func TestCursorReadRewriteHook_AllowsReadWithGuidance(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := scaffold.WriteCursorHooks(root); err != nil {
		t.Fatalf("WriteCursorHooks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "OZ.md"), []byte("name: test\n"), 0o644); err != nil {
		t.Fatalf("write OZ.md: %v", err)
	}

	script := filepath.Join(root, ".oz", "hooks", "oz-read-rewrite-cursor.sh")
	out, err := runHook(t, script, `{"tool_name":"Read","tool_input":{"file_path":"/tmp/example.txt"}}`, nil)
	if err != nil {
		t.Fatalf("run hook: %v", err)
	}

	for _, want := range []string{
		`"permission": "allow"`,
		`"agent_message":`,
		`oz shell read <path>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output:\n%s", want, out)
		}
	}
}

func TestCursorReadPolicyHook_AllowsWorkspaceReadWithGuidance(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := scaffold.WriteCursorHooks(root); err != nil {
		t.Fatalf("WriteCursorHooks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "OZ.md"), []byte("name: test\n"), 0o644); err != nil {
		t.Fatalf("write OZ.md: %v", err)
	}
	file := filepath.Join(root, "README.md")
	if err := os.WriteFile(file, []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}

	script := filepath.Join(root, ".oz", "hooks", "oz-read-policy-cursor.sh")
	out, err := runHook(t, script, `{"file_path":"`+file+`"}`, nil)
	if err != nil {
		t.Fatalf("run hook: %v", err)
	}
	for _, want := range []string{
		`"permission": "allow"`,
		`"user_message":`,
		`oz shell read <path>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output:\n%s", want, out)
		}
	}
}

func TestCursorReadPolicyHook_AllowsOutOfWorkspaceReadWithGuidance(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := scaffold.WriteCursorHooks(root); err != nil {
		t.Fatalf("WriteCursorHooks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "OZ.md"), []byte("name: test\n"), 0o644); err != nil {
		t.Fatalf("write OZ.md: %v", err)
	}

	script := filepath.Join(root, ".oz", "hooks", "oz-read-policy-cursor.sh")
	out, err := runHook(t, script, `{"file_path":"/outside/example.txt"}`, nil)
	if err != nil {
		t.Fatalf("run hook: %v", err)
	}

	for _, want := range []string{
		`"permission": "allow"`,
		`"user_message":`,
		`oz shell read <path>`,
		`Path: /outside/example.txt`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output:\n%s", want, out)
		}
	}
}

func installFakeOz(t *testing.T, rewrites map[string]string) string {
	t.Helper()

	binDir := t.TempDir()
	ozPath := filepath.Join(binDir, "oz")
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -euo pipefail\n")
	b.WriteString("if [[ \"$#\" -ge 3 && \"$1\" == \"shell\" && \"$2\" == \"rewrite\" ]]; then\n")
	b.WriteString("  cmd=\"$3\"\n")
	for in, out := range rewrites {
		b.WriteString("  if [[ \"$cmd\" == ")
		b.WriteString(shellQuote(in))
		b.WriteString(" ]]; then\n")
		b.WriteString("    printf '%s\\n' ")
		b.WriteString(shellQuote(out))
		b.WriteString("\n")
		b.WriteString("    exit 0\n")
		b.WriteString("  fi\n")
	}
	b.WriteString("  exit 1\n")
	b.WriteString("fi\n")
	b.WriteString("exit 1\n")

	if err := os.WriteFile(ozPath, []byte(b.String()), 0o755); err != nil {
		t.Fatalf("write fake oz: %v", err)
	}
	return binDir + string(os.PathListSeparator) + os.Getenv("PATH")
}

func runHook(t *testing.T, script, stdin string, extraEnv map[string]string) (string, error) {
	t.Helper()

	cmd := exec.Command(script)
	cmd.Dir = filepath.Dir(filepath.Dir(filepath.Dir(script)))
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = os.Environ()
	for k, v := range extraEnv {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
