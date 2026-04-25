package filter

import (
	"regexp"
	"strings"
)

var goBuildSignalRE = regexp.MustCompile(`(?i)(cannot|undefined|syntax error|imported and not used|unused import|invalid operation|does not exist)`)

type goBuildFilter struct{}

func (goBuildFilter) ID() ID { return FilterGoBuild }

func (goBuildFilter) Match(args []string) bool {
	if len(args) < 2 || trimArg(0, args) != "go" {
		return false
	}
	switch trimArg(1, args) {
	case "build", "run", "install":
		return true
	default:
		return false
	}
}

func (goBuildFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	combined := append(normalizeLines(stderr), normalizeLines(stdout)...)
	out := make([]string, 0)
	for _, line := range combined {
		if strings.Contains(line, ":") && (strings.Contains(line, ".go:") || strings.Contains(line, ".go ")) {
			out = append(out, line)
			continue
		}
		if goBuildSignalRE.MatchString(line) {
			out = append(out, line)
		}
	}
	if len(out) == 0 && exitCode == 0 {
		out = append(out, "go build summary: ok")
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 24, 40))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
