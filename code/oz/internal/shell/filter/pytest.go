package filter

import "strings"

type pytestFilter struct{}

func (pytestFilter) ID() ID { return FilterPytest }

func (pytestFilter) Match(args []string) bool {
	if len(args) >= 1 && trimArg(0, args) == "pytest" {
		return true
	}
	if len(args) >= 3 && trimArg(0, args) == "python" && trimArg(1, args) == "-m" && trimArg(2, args) == "pytest" {
		return true
	}
	return false
}

func (pytestFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := append(normalizeLines(stderr), normalizeLines(stdout)...)
	out := make([]string, 0)
	inTrace := false
	traceBudget := ternaryInt(ultraCompact, 40, 80)
	traceUsed := 0
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.HasPrefix(line, "===") && strings.Contains(lower, "fail") {
			out = append(out, line)
			inTrace = true
			continue
		}
		if strings.HasPrefix(line, "FAILED") || strings.HasPrefix(line, "ERROR") {
			out = append(out, line)
			inTrace = true
			continue
		}
		if strings.Contains(line, "E ") && strings.Contains(line, "Error") {
			out = append(out, line)
			inTrace = true
			continue
		}
		if inTrace && strings.HasPrefix(line, "_") {
			if traceUsed < traceBudget {
				out = append(out, line)
				traceUsed++
			}
			continue
		}
		if strings.Contains(lower, "traceback") {
			inTrace = true
			if traceUsed < traceBudget {
				out = append(out, line)
				traceUsed++
			}
			continue
		}
		if strings.Contains(line, "passed") && strings.Contains(line, "failed") {
			out = append(out, line)
			inTrace = false
		}
	}
	if len(out) == 0 && exitCode == 0 {
		out = append(out, "pytest summary: ok")
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 50, 100))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
