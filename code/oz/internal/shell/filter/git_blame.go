package filter

import (
	"fmt"
	"regexp"
	"strings"
)

var hexPrefixRE = regexp.MustCompile(`^([0-9a-f]{7,40})\b`)

type gitBlameFilter struct{}

func (gitBlameFilter) ID() ID { return FilterGitBlame }

func (gitBlameFilter) Match(args []string) bool {
	return len(args) >= 2 && trimArg(0, args) == "git" && trimArg(1, args) == "blame"
}

func (gitBlameFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	raw := strings.Split(strings.ReplaceAll(stripANSI(stdout), "\r\n", "\n"), "\n")
	type seg struct {
		start, end int
		hash       string
		sample     string
	}
	var segs []seg
	lineNo := 0
	for _, line := range raw {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lineNo++
		trim := strings.TrimSpace(line)
		m := hexPrefixRE.FindStringSubmatch(trim)
		if len(m) != 2 {
			continue
		}
		h := m[1]
		sample := strings.TrimSpace(strings.TrimPrefix(trim, h))
		if len(segs) > 0 && segs[len(segs)-1].hash == h {
			segs[len(segs)-1].end = lineNo
			continue
		}
		segs = append(segs, seg{start: lineNo, end: lineNo, hash: h, sample: sample})
	}
	out := []string{fmt.Sprintf("git blame summary: ranges=%d", len(segs))}
	maxRanges := ternaryInt(ultraCompact, 20, 40)
	head, omitted := topN(segs, maxRanges)
	for _, s := range head {
		h := shortHash(s.hash)
		if s.start == s.end {
			out = append(out, fmt.Sprintf("L%d %s %s", s.start, h, truncateRunes(s.sample, 60)))
		} else {
			out = append(out, fmt.Sprintf("L%d-L%d %s %s", s.start, s.end, h, truncateRunes(s.sample, 60)))
		}
	}
	if omitted > 0 {
		out = append(out, fmt.Sprintf("…(%d ranges omitted)", omitted))
	}
	if len(segs) == 0 {
		out = append(out, "git blame summary: (no ranges parsed)")
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	return strings.Join(stableUnique(out), "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
