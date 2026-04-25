package hooks_test

import (
	"testing"

	"github.com/joaoajmatos/oz/internal/shell/hooks"
)

func TestDecideSuggestDefault(t *testing.T) {
	t.Parallel()

	decision := hooks.Decide("git status", hooks.DefaultConfig())
	if !decision.Allowed {
		t.Fatalf("expected allowed decision")
	}
	if decision.Mode != hooks.ModeSuggest {
		t.Fatalf("mode=%q, want suggest", decision.Mode)
	}
	if decision.Suggested == "" {
		t.Fatalf("expected suggested command")
	}
	if decision.Rewritten != "" {
		t.Fatalf("expected no rewritten command in suggest mode")
	}
}

func TestDecideRewriteOptIn(t *testing.T) {
	t.Parallel()

	cfg := hooks.DefaultConfig()
	cfg.Mode = hooks.ModeRewrite
	decision := hooks.Decide("go test ./...", cfg)
	if !decision.Allowed {
		t.Fatalf("expected allowed decision")
	}
	if decision.Rewritten == "" {
		t.Fatalf("expected rewritten command in rewrite mode")
	}
	if decision.Suggested != "" {
		t.Fatalf("expected no suggestion in rewrite mode")
	}
}

func TestDecideExclusion(t *testing.T) {
	t.Parallel()

	cfg := hooks.DefaultConfig()
	cfg.ExcludeCommands = []string{"git"}
	decision := hooks.Decide("git status", cfg)
	if decision.Allowed {
		t.Fatalf("expected excluded command to be blocked")
	}
}

func TestDecideFailOpenDisabled(t *testing.T) {
	t.Parallel()

	cfg := hooks.DefaultConfig()
	cfg.Enabled = false
	decision := hooks.Decide("echo hello", cfg)
	if decision.Allowed {
		t.Fatalf("expected disabled hooks to avoid rewrite")
	}
	if decision.Reason == "" {
		t.Fatalf("expected reason for fail-open no-op")
	}
}

func TestDecideAlreadyWrappedVariants(t *testing.T) {
	t.Parallel()

	cases := []string{
		"oz shell run -- git status",
		" OZ   SHELL   RUN -- git status ",
	}
	for _, in := range cases {
		decision := hooks.Decide(in, hooks.DefaultConfig())
		if decision.Allowed {
			t.Fatalf("expected already-wrapped command to be blocked: %q", in)
		}
		if decision.Reason != hooks.ReasonAlreadyWrapped {
			t.Fatalf("unexpected reason %q for %q", decision.Reason, in)
		}
	}
}

func TestDecideExclusionQuotedCommand(t *testing.T) {
	t.Parallel()

	cfg := hooks.DefaultConfig()
	cfg.ExcludeCommands = []string{"git"}
	decision := hooks.Decide(`"git" status`, cfg)
	if decision.Allowed {
		t.Fatalf("expected quoted excluded command to be blocked")
	}
}

func TestDecideRewriteCompoundOperators(t *testing.T) {
	t.Parallel()

	decision := hooks.Decide("git status && go test ./...; echo done", hooks.RewriteConfig())
	if !decision.Allowed {
		t.Fatalf("expected allowed decision")
	}
	want := "oz shell run -- git status && oz shell run -- go test ./...; oz shell run -- echo done"
	if decision.Rewritten != want {
		t.Fatalf("rewritten=%q, want %q", decision.Rewritten, want)
	}
}

func TestDecideAlreadyWrappedOzShellRead(t *testing.T) {
	t.Parallel()

	cases := []string{
		"oz shell read foo.go",
		"oz shell read --max-lines 50 foo.go",
		"OZ SHELL READ foo.go",
	}
	for _, in := range cases {
		decision := hooks.Decide(in, hooks.DefaultConfig())
		if decision.Allowed {
			t.Fatalf("expected oz shell read to be treated as already-wrapped: %q", in)
		}
		if decision.Reason != hooks.ReasonAlreadyWrapped {
			t.Fatalf("unexpected reason %q for %q", decision.Reason, in)
		}
	}
}

func TestDecideRewriteCatToOzShellRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"cat foo.go", "oz shell read --line-numbers foo.go"},
		{"cat a.go b.go", "oz shell read --line-numbers a.go b.go"},
		{"cat -", "oz shell read --line-numbers -"},
	}
	for _, tc := range cases {
		decision := hooks.Decide(tc.in, hooks.RewriteConfig())
		if !decision.Allowed {
			t.Fatalf("expected allowed for %q", tc.in)
		}
		if decision.Rewritten != tc.want {
			t.Fatalf("rewritten=%q, want %q", decision.Rewritten, tc.want)
		}
	}
}

func TestDecideRewriteHeadToOzShellRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"head foo.go", "oz shell read --max-lines 10 foo.go"},
		{"head -n 50 foo.go", "oz shell read --max-lines 50 foo.go"},
		{"head -50 foo.go", "oz shell read --max-lines 50 foo.go"},
	}
	for _, tc := range cases {
		decision := hooks.Decide(tc.in, hooks.RewriteConfig())
		if !decision.Allowed {
			t.Fatalf("expected allowed for %q", tc.in)
		}
		if decision.Rewritten != tc.want {
			t.Fatalf("rewritten=%q, want %q", decision.Rewritten, tc.want)
		}
	}
}

func TestDecideRewriteTailToOzShellRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"tail foo.go", "oz shell read --tail-lines 10 foo.go"},
		{"tail -n 20 foo.go", "oz shell read --tail-lines 20 foo.go"},
	}
	for _, tc := range cases {
		decision := hooks.Decide(tc.in, hooks.RewriteConfig())
		if !decision.Allowed {
			t.Fatalf("expected allowed for %q", tc.in)
		}
		if decision.Rewritten != tc.want {
			t.Fatalf("rewritten=%q, want %q", decision.Rewritten, tc.want)
		}
	}
}

func TestDecideNoRewriteForUnsupportedCatFlags(t *testing.T) {
	t.Parallel()

	cases := []string{
		"cat -n foo.go",     // line numbers flag — not passed to oz shell read
		"cat > foo.go",      // redirection
		"cat << EOF",        // heredoc
		"tail -f foo.log",   // follow mode
		"head -c 100 foo.go", // byte count
	}
	for _, in := range cases {
		decision := hooks.Decide(in, hooks.RewriteConfig())
		if decision.Allowed && !containsOzShellRead(decision.Rewritten) {
			// Rewritten to oz shell run is fine; rewritten to oz shell read is not expected here.
			continue
		}
		if decision.Allowed && containsOzShellRead(decision.Rewritten) {
			t.Fatalf("expected no oz shell read rewrite for %q, got %q", in, decision.Rewritten)
		}
	}
}

func containsOzShellRead(s string) bool {
	return len(s) >= len("oz shell read") &&
		s[:len("oz shell read")] == "oz shell read"
}

func TestDecideRewritePipeOnlyLeftSide(t *testing.T) {
	t.Parallel()

	decision := hooks.Decide("git status | wc -l", hooks.RewriteConfig())
	if !decision.Allowed {
		t.Fatalf("expected allowed decision")
	}
	want := "oz shell run -- git status | wc -l"
	if decision.Rewritten != want {
		t.Fatalf("rewritten=%q, want %q", decision.Rewritten, want)
	}
}
