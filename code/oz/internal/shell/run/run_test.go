package run_test

import (
	"strings"
	"testing"

	shellrun "github.com/joaoajmatos/oz/internal/shell/run"
)

func TestExecuteCompactGeneric(t *testing.T) {
	t.Parallel()

	result, err := shellrun.Execute([]string{"sh", "-c", "echo hello; echo hello; echo err >&2"}, shellrun.Options{
		Mode:    "compact",
		TeeMode: "never",
		NoTrack: true,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Envelope.MatchedFilter != "generic" {
		t.Fatalf("MatchedFilter=%q, want generic", result.Envelope.MatchedFilter)
	}
	if !strings.Contains(result.Envelope.Stdout, "hello") {
		t.Fatalf("expected compact stdout to keep signal, got %q", result.Envelope.Stdout)
	}
}

func TestExecuteCompactFallbackToRaw(t *testing.T) {
	t.Parallel()

	result, err := shellrun.Execute([]string{"sh", "-c", "echo __OZ_COMPACT_ERROR__"}, shellrun.Options{
		Mode:    "compact",
		TeeMode: "never",
		NoTrack: true,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(result.Envelope.Warnings) == 0 {
		t.Fatalf("expected warning when compact fails")
	}
	if !strings.Contains(result.Envelope.Stdout, "__OZ_COMPACT_ERROR__") {
		t.Fatalf("expected raw fallback stdout, got %q", result.Envelope.Stdout)
	}
}

func TestExecuteTeeFailuresOnly(t *testing.T) {
	t.Parallel()

	okResult, err := shellrun.Execute([]string{"sh", "-c", "echo ok"}, shellrun.Options{
		Mode:    "raw",
		TeeMode: "failures",
		NoTrack: true,
	})
	if err != nil {
		t.Fatalf("Execute success case error: %v", err)
	}
	if okResult.Envelope.RawOutputRef != nil {
		t.Fatalf("expected no tee artifact for successful command")
	}

	failResult, err := shellrun.Execute([]string{"sh", "-c", "echo fail >&2; exit 3"}, shellrun.Options{
		Mode:    "raw",
		TeeMode: "failures",
		NoTrack: true,
	})
	if err != nil {
		t.Fatalf("Execute failure case error: %v", err)
	}
	if failResult.Envelope.RawOutputRef == nil || *failResult.Envelope.RawOutputRef == "" {
		t.Fatalf("expected tee artifact path for failing command")
	}
}

func TestExecuteLargeOutputNoPanic(t *testing.T) {
	t.Parallel()

	builder := strings.Repeat("line\n", 5000)
	cmd := "cat <<'EOF'\n" + builder + "EOF"
	result, err := shellrun.Execute([]string{"sh", "-c", cmd}, shellrun.Options{
		Mode:    "compact",
		TeeMode: "never",
		NoTrack: true,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Envelope.TokenEstBefore == 0 {
		t.Fatalf("expected non-zero token estimate")
	}
}
