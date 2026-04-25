package langs

import (
	"fmt"
	"strings"

	"github.com/joaoajmatos/oz/internal/shell/readfilter"
)

type markdownReader struct{}

func (markdownReader) Name() string { return "markdown" }
func (markdownReader) Extensions() []string {
	return []string{".md", ".markdown"}
}

func (markdownReader) Filter(content string, opts readfilter.Options) (string, error) {
	maxUnknownFence := 12
	if opts.UltraCompact {
		maxUnknownFence = 6
	}

	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	inFence := false
	fenceStart := 0
	fenceLang := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inFence {
				inFence = true
				fenceStart = i
				fenceLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				continue
			}
			blockLen := i - fenceStart - 1
			known := isKnownFenceLang(fenceLang)
			if known || blockLen <= maxUnknownFence {
				out = append(out, lines[fenceStart:i+1]...)
			} else {
				out = append(out, lines[fenceStart])
				firstOmitted := fenceStart + 2
				lastOmitted := i
				out = append(out, fmt.Sprintf("... (%d lines omitted, lines %d-%d)", blockLen, firstOmitted, lastOmitted))
				out = append(out, line)
			}
			inFence = false
			fenceLang = ""
			continue
		}
		if !inFence {
			out = append(out, line)
		}
	}
	if inFence {
		out = append(out, lines[fenceStart:]...)
	}
	return strings.Join(out, "\n"), nil
}

func isKnownFenceLang(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "go", "json", "md", "markdown", "bash", "sh", "shell", "zsh", "python", "js", "ts", "tsx", "yaml", "yml", "toml":
		return true
	default:
		return false
	}
}

func init() {
	readfilter.Register(markdownReader{})
}
