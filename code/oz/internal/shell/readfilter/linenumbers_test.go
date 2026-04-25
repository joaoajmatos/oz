package readfilter

import (
	"strings"
	"testing"
)

func TestApplyLineNumbersRightAligned(t *testing.T) {
	lines := make([]string, 0, 110)
	for i := 0; i < 110; i++ {
		lines = append(lines, "x")
	}
	in := strings.Join(lines, "\n")
	got := applyLineNumbers(in)
	all := strings.Split(got, "\n")
	if all[0] != "  1|x" {
		t.Fatalf("first line=%q, want %q", all[0], "  1|x")
	}
	if all[99] != "100|x" {
		t.Fatalf("line 100=%q, want %q", all[99], "100|x")
	}
}
