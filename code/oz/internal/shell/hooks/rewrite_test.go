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
