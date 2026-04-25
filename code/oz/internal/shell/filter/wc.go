package filter

import (
	"fmt"
	"strconv"
	"strings"
)

type wcFilter struct{}

func (wcFilter) ID() ID { return FilterWc }

func (wcFilter) Match(args []string) bool {
	return len(args) >= 1 && trimArg(0, args) == "wc"
}

func (wcFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := normalizeLines(stripANSI(stdout))
	if len(lines) == 0 {
		lines = normalizeLines(stripANSI(stderr))
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 2 {
		out := []string{"wc summary: " + strings.Join(lines, " ")}
		if exitCode != 0 {
			out = appendFailureContext(out, stdout, stderr)
		}
		return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
	}
	var nums []int64
	i := 0
	for i < len(fields) {
		n, err := strconv.ParseInt(fields[i], 10, 64)
		if err != nil {
			break
		}
		nums = append(nums, n)
		i++
	}
	file := "total"
	if i < len(fields) {
		file = strings.Join(fields[i:], " ")
	}
	var summary string
	switch len(nums) {
	case 1:
		summary = fmt.Sprintf("wc %d %s", nums[0], file)
	case 2:
		summary = fmt.Sprintf("wc L=%d B=%d %s", nums[0], nums[1], file)
	case 3:
		summary = fmt.Sprintf("wc L=%d W=%d B=%d %s", nums[0], nums[1], nums[2], file)
	default:
		summary = "wc: " + strings.Join(lines, " ")
	}
	out := []string{summary}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
