package langs

import (
	"regexp"
	"strings"

	"github.com/joaoajmatos/oz/internal/shell/readfilter"
)

var (
	goBlockCommentsRE = regexp.MustCompile(`(?s)/\*.*?\*/`)
	goLineCommentRE   = regexp.MustCompile(`//.*$`)
)

type golangReader struct{}

func (golangReader) Name() string { return "go" }
func (golangReader) Extensions() []string {
	return []string{".go"}
}

func (golangReader) Filter(content string, _ readfilter.Options) (string, error) {
	content = goBlockCommentsRE.ReplaceAllString(content, "")
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	blankCount := 0
	for _, line := range lines {
		line = goLineCommentRE.ReplaceAllString(line, "")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			if blankCount > 1 {
				continue
			}
			out = append(out, "")
			continue
		}
		blankCount = 0
		out = append(out, strings.TrimRight(line, " \t"))
	}
	return strings.TrimSpace(strings.Join(out, "\n")) + "\n", nil
}

func init() {
	readfilter.Register(golangReader{})
}
