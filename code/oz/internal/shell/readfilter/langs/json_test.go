package langs

import (
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/shell/readfilter"
)

func TestJSONReaderSummarizesObjects(t *testing.T) {
	out, err := jsonReader{}.Filter(`{"a":1,"b":{"x":true},"c":[1,2,3]}`, readfilter.Options{})
	if err != nil {
		t.Fatalf("Filter returned error: %v", err)
	}
	if !strings.Contains(out, "json object: keys=3") {
		t.Fatalf("expected object summary, got %q", out)
	}
}
