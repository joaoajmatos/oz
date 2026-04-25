package langs

import (
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/shell/readfilter"
)

func TestMarkdownReaderCollapsesUnknownLongFence(t *testing.T) {
	in := strings.Join([]string{
		"# Title",
		"```weirdlang",
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13",
		"```",
	}, "\n")
	out, err := markdownReader{}.Filter(in, readfilter.Options{})
	if err != nil {
		t.Fatalf("Filter returned error: %v", err)
	}
	if !strings.Contains(out, "lines omitted") {
		t.Fatalf("expected collapsed fence output, got %q", out)
	}
}
