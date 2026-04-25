package filter

import "strings"

func applyGoTest(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := normalizeLines(stdout)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "--- FAIL:") ||
			strings.HasPrefix(line, "FAIL") ||
			strings.HasPrefix(line, "panic:") ||
			strings.Contains(line, "FAIL\t") {
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(line, "ok\t") && !ultraCompact {
			continue
		}
	}

	if len(out) == 0 {
		if exitCode == 0 {
			out = append(out, "go test summary: pass")
		} else {
			out = appendFailureContext(out, stdout, stderr)
		}
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 20, 40))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}

type goTestFilter struct{}

func (goTestFilter) ID() ID { return FilterGoTest }

func (goTestFilter) Match(args []string) bool {
	return len(args) >= 2 && trimArg(0, args) == "go" && trimArg(1, args) == "test"
}

func (goTestFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	return applyGoTest(stdout, stderr, exitCode, ultraCompact)
}
