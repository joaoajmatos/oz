package readfilter

import "strings"

func applyMaxLines(content string, max *int) string {
	if max == nil {
		return content
	}
	n := *max
	if n <= 0 {
		return ""
	}
	lines, trailingNewline := splitContentLines(content)
	if len(lines) <= n {
		return content
	}
	return joinContentLines(lines[:n], false || (trailingNewline && n == len(lines)))
}

func applyTailLines(content string, tail *int) string {
	if tail == nil {
		return content
	}
	n := *tail
	if n <= 0 {
		return ""
	}
	lines, trailingNewline := splitContentLines(content)
	if len(lines) <= n {
		return content
	}
	start := len(lines) - n
	return joinContentLines(lines[start:], trailingNewline)
}

func splitContentLines(content string) ([]string, bool) {
	if content == "" {
		return nil, false
	}
	trailingNewline := strings.HasSuffix(content, "\n")
	trimmed := content
	if trailingNewline {
		trimmed = strings.TrimSuffix(content, "\n")
	}
	if trimmed == "" {
		return []string{""}, trailingNewline
	}
	return strings.Split(trimmed, "\n"), trailingNewline
}

func joinContentLines(lines []string, trailingNewline bool) string {
	if len(lines) == 0 {
		return ""
	}
	out := strings.Join(lines, "\n")
	if trailingNewline {
		out += "\n"
	}
	return out
}
