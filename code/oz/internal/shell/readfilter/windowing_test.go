package readfilter

import "testing"

func intPtr(v int) *int { return &v }

func TestApplyTailLinesNoTrailingNewline(t *testing.T) {
	in := "a\nb\nc"
	got := applyTailLines(in, intPtr(2))
	if got != "b\nc" {
		t.Fatalf("applyTailLines=%q, want %q", got, "b\nc")
	}
}

func TestApplyTailLinesZero(t *testing.T) {
	in := "a\nb\nc\n"
	got := applyTailLines(in, intPtr(0))
	if got != "" {
		t.Fatalf("applyTailLines=%q, want empty string", got)
	}
}

func TestApplyMaxLines(t *testing.T) {
	in := "a\nb\nc\n"
	got := applyMaxLines(in, intPtr(2))
	if got != "a\nb" {
		t.Fatalf("applyMaxLines=%q, want %q", got, "a\nb")
	}
}
