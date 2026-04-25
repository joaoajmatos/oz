package filter

import (
	"regexp"
	"strings"
)

var makeNoiseRE = regexp.MustCompile(`^make\[\d+\]:`)

type makeFilter struct{}

func (makeFilter) ID() ID { return FilterMake }

func (makeFilter) Match(args []string) bool {
	return len(args) >= 1 && trimArg(0, args) == "make"
}

func (makeFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := append(normalizeLines(stderr), normalizeLines(stdout)...)
	out := make([]string, 0)
	for _, line := range lines {
		if makeNoiseRE.MatchString(line) {
			continue
		}
		if strings.HasPrefix(line, "cd ") && strings.Contains(line, "&&") {
			continue
		}
		if strings.Contains(strings.ToLower(line), "error") ||
			strings.Contains(line, "*** [") ||
			strings.Contains(line, "Stop.") ||
			strings.HasPrefix(line, "make:") {
			out = append(out, line)
		}
	}
	if len(out) == 0 && exitCode == 0 {
		out = append(out, "make summary: ok")
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 20, 40))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
