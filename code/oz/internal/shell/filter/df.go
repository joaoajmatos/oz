package filter

import (
	"fmt"
	"strings"
)

type dfFilter struct{}

func (dfFilter) ID() ID { return FilterDf }

func (dfFilter) Match(args []string) bool {
	return len(args) >= 1 && trimArg(0, args) == "df"
}

func (dfFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := normalizeLines(stripANSI(stdout))
	var header string
	rows := make([]string, 0)
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "filesystem") {
			header = line
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		fs := fields[0]
		lower := strings.ToLower(fs)
		if strings.Contains(lower, "tmpfs") || lower == "devfs" || lower == "udev" ||
			lower == "none" || strings.Contains(lower, "overlay") {
			continue
		}
		usep := fields[4]
		mount := strings.Join(fields[5:], " ")
		rows = append(rows, fmt.Sprintf("%s use=%s mount=%s", fs, usep, mount))
	}
	out := []string{fmt.Sprintf("df summary: rows=%d", len(rows))}
	if header != "" {
		out = append(out, header)
	}
	out = append(out, rows...)
	if len(rows) == 0 && header == "" {
		out = []string{"df summary: (no rows parsed)"}
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 15, 30))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
