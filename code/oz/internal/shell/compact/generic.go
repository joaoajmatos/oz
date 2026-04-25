package compact

import (
	"fmt"
	"regexp"
	"strings"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// ApplyGeneric runs deterministic generic compaction over stdout/stderr.
func ApplyGeneric(stdout, stderr string, ultraCompact bool) (string, string, error) {
	if strings.Contains(stdout, "__OZ_COMPACT_ERROR__") || strings.Contains(stderr, "__OZ_COMPACT_ERROR__") {
		return "", "", fmt.Errorf("forced compact error")
	}
	return compactStream(stdout, ultraCompact), compactStream(stderr, ultraCompact), nil
}

func compactStream(s string, ultraCompact bool) string {
	if s == "" {
		return ""
	}

	maxLines := 120
	if ultraCompact {
		maxLines = 60
	}
	maxLineLen := 500

	lines := strings.Split(s, "\n")
	compacted := make([]string, 0, len(lines))
	lastLine := ""
	repeatCount := 0

	flushRepeat := func() {
		if repeatCount > 1 {
			compacted = append(compacted, fmt.Sprintf("[repeated %d times] %s", repeatCount, lastLine))
		} else if repeatCount == 1 {
			compacted = append(compacted, lastLine)
		}
		repeatCount = 0
	}

	for _, line := range lines {
		clean := strings.TrimSpace(ansiRE.ReplaceAllString(line, ""))
		if clean == "" {
			continue
		}
		if len(clean) > maxLineLen {
			clean = clean[:maxLineLen] + "...(truncated)"
		}
		if repeatCount == 0 {
			lastLine = clean
			repeatCount = 1
			continue
		}
		if clean == lastLine {
			repeatCount++
			continue
		}
		flushRepeat()
		lastLine = clean
		repeatCount = 1
	}
	flushRepeat()

	if len(compacted) <= maxLines {
		return strings.Join(compacted, "\n")
	}

	headCount := maxLines / 2
	tailCount := maxLines - headCount - 1
	head := compacted[:headCount]
	tail := compacted[len(compacted)-tailCount:]
	out := make([]string, 0, maxLines)
	out = append(out, head...)
	out = append(out, fmt.Sprintf("...(%d lines omitted)...", len(compacted)-headCount-tailCount))
	out = append(out, tail...)
	return strings.Join(out, "\n")
}
