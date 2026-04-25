package filter

import (
	"regexp"
	"strings"
)

var dockerPullRE = regexp.MustCompile(`(?i)^(Pulling|Downloading|Extracting|Verifying|Waiting)\b`)

type dockerFilter struct{}

func (dockerFilter) ID() ID { return FilterDocker }

func (dockerFilter) Match(args []string) bool {
	if len(args) < 2 || trimArg(0, args) != "docker" {
		return false
	}
	switch trimArg(1, args) {
	case "build", "run":
		return true
	default:
		return false
	}
}

func (dockerFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := append(normalizeLines(stderr), normalizeLines(stdout)...)
	out := make([]string, 0)
	for _, line := range lines {
		if dockerPullRE.MatchString(line) {
			continue
		}
		if strings.Contains(line, "Step ") && strings.Contains(line, "/") {
			out = append(out, line)
			continue
		}
		if strings.Contains(strings.ToLower(line), "error") || strings.Contains(line, "ERROR") {
			out = append(out, line)
			continue
		}
		if strings.Contains(line, "Successfully built") || strings.Contains(line, "digest:") {
			out = append(out, line)
		}
	}
	if len(out) == 0 && exitCode == 0 {
		out = append(out, "docker summary: ok")
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 25, 50))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
