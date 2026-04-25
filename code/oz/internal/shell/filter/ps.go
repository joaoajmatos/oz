package filter

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type psRow struct {
	line string
	cpu  float64
	mem  float64
}

type psFilter struct{}

func (psFilter) ID() ID { return FilterPs }

func (psFilter) Match(args []string) bool {
	return len(args) >= 1 && trimArg(0, args) == "ps"
}

func (psFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	raw := strings.Split(strings.ReplaceAll(stripANSI(stdout), "\r\n", "\n"), "\n")
	var header string
	var rows []psRow
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if strings.EqualFold(fields[0], "USER") || fields[0] == "PID" {
			header = line
			continue
		}
		if strings.HasPrefix(fields[0], "[") {
			continue
		}
		cpu, err1 := strconv.ParseFloat(fields[2], 64)
		mem, err2 := strconv.ParseFloat(fields[3], 64)
		if err1 != nil || err2 != nil {
			rows = append(rows, psRow{line: line, cpu: 0, mem: 0})
			continue
		}
		rows = append(rows, psRow{line: line, cpu: cpu, mem: mem})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].cpu == rows[j].cpu {
			return rows[i].mem > rows[j].mem
		}
		return rows[i].cpu > rows[j].cpu
	})
	max := ternaryInt(ultraCompact, 10, 20)
	head, omitted := topN(rows, max)
	out := make([]string, 0)
	if header != "" {
		out = append(out, header)
	}
	if len(head) == 0 && header == "" {
		out = append(out, "ps summary: (no rows parsed)")
	} else {
		out = append(out, fmt.Sprintf("ps summary: shown=%d", len(head)))
	}
	for _, r := range head {
		out = append(out, r.line)
	}
	if omitted > 0 {
		out = append(out, fmt.Sprintf("…(%d processes omitted)", omitted))
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	return strings.Join(stableUnique(out), "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
