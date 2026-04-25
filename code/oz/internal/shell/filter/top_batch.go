package filter

import (
	"fmt"
	"strings"
)

type topBatchFilter struct{}

func (topBatchFilter) ID() ID { return FilterPs }

func (topBatchFilter) Match(args []string) bool {
	if len(args) < 1 || trimArg(0, args) != "top" {
		return false
	}
	hasB, hasN := false, false
	for i := 1; i < len(args); i++ {
		a := trimArg(i, args)
		switch a {
		case "-b":
			hasB = true
		case "-n":
			if i+1 < len(args) {
				hasN = true
			}
		default:
			if strings.HasPrefix(a, "-n") && len(strings.TrimSpace(a)) > 2 {
				hasN = true
			}
		}
	}
	return hasB && hasN
}

func (topBatchFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := normalizeLines(stripANSI(stdout))
	out := keepHead(lines, ternaryInt(ultraCompact, 15, 30))
	prefix := fmt.Sprintf("top batch summary: lines=%d", len(out))
	return prefix + "\n" + strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
