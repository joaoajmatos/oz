package filter

import (
	"fmt"
	"sort"
	"strings"
)

type envFilter struct{}

func (envFilter) ID() ID { return FilterEnv }

func (envFilter) Match(args []string) bool {
	if len(args) < 1 {
		return false
	}
	switch trimArg(0, args) {
	case "env", "printenv":
		return true
	default:
		return false
	}
}

func (envFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := normalizeLines(stripANSI(stdout))
	if len(lines) == 0 {
		lines = normalizeLines(stripANSI(stderr))
	}
	keys := make([]string, 0)
	redacted := 0
	total := 0
	for _, line := range lines {
		key, _, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		total++
		key = strings.TrimSpace(key)
		if isSecretKeyName(key) {
			redacted++
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	maxKeys := ternaryInt(ultraCompact, 30, 60)
	head, omitted := topN(keys, maxKeys)
	out := []string{
		fmt.Sprintf("env summary: vars=%d redacted=%d", total, redacted),
		"keys:" + formatOmittedSuffix(omitted),
	}
	for _, k := range head {
		out = append(out, "  "+k)
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	return strings.Join(stableUnique(out), "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
