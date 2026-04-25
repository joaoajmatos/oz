package langs

import (
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/shell/readfilter"
)

func TestGolangReaderStripsComments(t *testing.T) {
	r := golangReader{}
	out, err := r.Filter("package main\n// keep\nfunc main() {}\n", readfilter.Options{})
	if err != nil {
		t.Fatalf("Filter returned error: %v", err)
	}
	if strings.Contains(out, "// keep") {
		t.Fatalf("expected comments removed, got %q", out)
	}
	if !strings.Contains(out, "func main() {}") {
		t.Fatalf("expected function signature preserved, got %q", out)
	}
}
