package filter

import (
	"regexp"
	"strings"
)

var npmProgressRE = regexp.MustCompile(`[\x{2800}-\x{28FF}]+`)

type npmFilter struct{}

func (npmFilter) ID() ID { return FilterNpm }

func (npmFilter) Match(args []string) bool {
	if len(args) < 1 {
		return false
	}
	switch trimArg(0, args) {
	case "npm", "yarn", "pnpm":
		return true
	default:
		return false
	}
}

func (npmFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := append(normalizeLines(stderr), normalizeLines(stdout)...)
	out := make([]string, 0)
	for _, line := range lines {
		line = npmProgressRE.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "downloading") && strings.Contains(lower, "http") {
			continue
		}
		if strings.HasPrefix(lower, "added ") && strings.Contains(lower, "packages") {
			out = append(out, line)
			continue
		}
		if strings.Contains(lower, "npm warn") || strings.Contains(lower, "npm error") ||
			strings.Contains(lower, "yarn warn") || strings.Contains(lower, "error ") ||
			strings.Contains(lower, "audited") || strings.Contains(lower, "vulnerabilit") {
			out = append(out, line)
		}
	}
	if len(out) == 0 && exitCode == 0 {
		out = append(out, "npm summary: ok")
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 20, 40))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
