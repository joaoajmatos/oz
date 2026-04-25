package filter

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/joaoajmatos/oz/internal/shell/compact"
)

var (
	wsRE      = regexp.MustCompile(`\s+`)
	failureRE = regexp.MustCompile(`(?i)(fail|error|fatal|panic)`)
)

func normalizeLines(s string) []string {
	raw := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = wsRE.ReplaceAllString(line, " ")
		lines = append(lines, line)
	}
	return lines
}

func stableUnique(lines []string) []string {
	seen := make(map[string]struct{}, len(lines))
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	return out
}

func keepHead(lines []string, n int) []string {
	if n <= 0 || len(lines) <= n {
		return lines
	}
	return lines[:n]
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func appendFailureContext(out []string, stdout, stderr string) []string {
	candidates := append(normalizeLines(stderr), normalizeLines(stdout)...)
	for _, line := range candidates {
		if failureRE.MatchString(line) {
			out = append(out, line)
		}
	}
	return stableUnique(out)
}

func fallbackGeneric(stdout, stderr string, ultraCompact bool) (string, string, error) {
	return compact.ApplyGeneric(stdout, stderr, ultraCompact)
}

func renderSection(title string, lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return fmt.Sprintf("%s\n- %s", title, strings.Join(lines, "\n- "))
}

func ternaryInt(cond bool, whenTrue, whenFalse int) int {
	if cond {
		return whenTrue
	}
	return whenFalse
}
