package readfilter

import "testing"

type testReader struct {
	name string
	exts []string
}

func (r testReader) Name() string { return r.name }
func (r testReader) Extensions() []string {
	return r.exts
}
func (r testReader) Filter(content string, _ Options) (string, error) { return content, nil }

func TestResolvePrefersDirectExtension(t *testing.T) {
	Register(testReader{name: "direct-hit", exts: []string{".direct-readfilter"}})
	got := Resolve("sample.direct-readfilter")
	if got.Name() != "direct-hit" {
		t.Fatalf("Resolve direct extension=%q, want direct-hit", got.Name())
	}
}

func TestResolveFallsBackToGeneric(t *testing.T) {
	Register(testReader{name: "generic", exts: nil})
	got := Resolve("sample.unknown-readfilter-ext")
	if got.Name() != "generic" {
		t.Fatalf("Resolve fallback=%q, want generic", got.Name())
	}
}
