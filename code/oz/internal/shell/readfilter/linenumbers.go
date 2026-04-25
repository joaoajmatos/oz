package readfilter

import (
	"fmt"
	"strconv"
)

func applyLineNumbers(content string) string {
	lines, trailingNewline := splitContentLines(content)
	if len(lines) == 0 {
		return content
	}
	width := len(strconv.Itoa(len(lines)))
	numbered := make([]string, 0, len(lines))
	for i, line := range lines {
		numbered = append(numbered, fmt.Sprintf("%*d|%s", width, i+1, line))
	}
	return joinContentLines(numbered, trailingNewline)
}
