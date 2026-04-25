package filter

import (
	"fmt"
	"strings"
)

type treeFilter struct{}

func (treeFilter) ID() ID { return FilterTree }

func (treeFilter) Match(args []string) bool {
	return len(args) >= 1 && trimArg(0, args) == "tree"
}

func (treeFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	raw := strings.Split(strings.ReplaceAll(stripANSI(stdout), "\r\n", "\n"), "\n")
	maxDepth := ternaryInt(ultraCompact, 1, 2)
	var out []string
	deep := 0
	for _, line := range raw {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		depth := treeLineDepth(line)
		if depth <= maxDepth {
			out = append(out, line)
			continue
		}
		deep++
	}
	if deep > 0 {
		out = append(out, fmt.Sprintf("… collapsed deeper tree lines: %d", deep))
	}
	if len(out) == 0 {
		out = append(out, "tree summary: (empty)")
	} else {
		out = append([]string{fmt.Sprintf("tree summary: lines=%d max_depth_shown=%d", len(out), maxDepth)}, out...)
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 30, 60))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}

func treeLineDepth(line string) int {
	s := line
	d := 0
	for strings.HasPrefix(s, "│   ") || strings.HasPrefix(s, "|   ") {
		if strings.HasPrefix(s, "│   ") {
			s = s[len("│   "):]
		} else {
			s = s[len("|   "):]
		}
		d++
	}
	for strings.HasPrefix(s, "    ") {
		s = s[4:]
		d++
	}
	if strings.HasPrefix(s, "├──") || strings.HasPrefix(s, "└──") || strings.HasPrefix(s, "`--") {
		d++
	}
	return d
}
