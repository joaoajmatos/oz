package filter

import (
	"fmt"
	"strings"
)

func applyRG(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	grouped := map[string][]string{}
	for _, line := range normalizeLines(stdout) {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		file := parts[0]
		entry := fmt.Sprintf("L%s: %s", parts[1], parts[2])
		grouped[file] = append(grouped[file], entry)
	}

	out := make([]string, 0)
	maxFiles := ternaryInt(ultraCompact, 8, 20)
	maxLinesPerFile := ternaryInt(ultraCompact, 1, 1)
	files := keepHead(sortedKeys(grouped), maxFiles)
	for _, file := range files {
		out = append(out, fmt.Sprintf("%s (%d matches)", file, len(grouped[file])))
		for _, entry := range keepHead(stableUnique(grouped[file]), maxLinesPerFile) {
			out = append(out, "  "+entry)
		}
	}

	if len(out) == 0 {
		out = append(out, "rg summary: no parsed matches")
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	return strings.Join(stableUnique(out), "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}

type rgFilter struct{}

func (rgFilter) ID() ID { return FilterRG }

func (rgFilter) Match(args []string) bool {
	if len(args) < 1 {
		return false
	}
	switch trimArg(0, args) {
	case "rg", "grep":
		return true
	default:
		return false
	}
}

func (rgFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	return applyRG(stdout, stderr, exitCode, ultraCompact)
}
