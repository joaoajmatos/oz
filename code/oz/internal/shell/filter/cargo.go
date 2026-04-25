package filter

import (
	"regexp"
	"strings"
)

var cargoNoiseRE = regexp.MustCompile(`(?i)^(Compiling|Downloading|Updating|Finished|Running)\b`)

type cargoFilter struct{}

func (cargoFilter) ID() ID { return FilterCargo }

func (cargoFilter) Match(args []string) bool {
	return len(args) >= 1 && trimArg(0, args) == "cargo"
}

func (cargoFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := append(normalizeLines(stderr), normalizeLines(stdout)...)
	out := make([]string, 0)
	for _, line := range lines {
		if cargoNoiseRE.MatchString(line) {
			continue
		}
		if strings.Contains(line, "error[E") || strings.Contains(line, "--> ") || strings.Contains(line, "error: ") {
			out = append(out, line)
			continue
		}
		if strings.Contains(line, "warning:") && !ultraCompact {
			out = append(out, line)
		}
	}
	if len(out) == 0 && exitCode == 0 {
		out = append(out, "cargo summary: ok")
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 30, 60))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
