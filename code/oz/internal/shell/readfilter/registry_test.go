package readfilter

import (
	"testing"

	"github.com/joaoajmatos/oz/internal/codeindex"
)

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

type codeIndexOnlyPackage struct{}

func (codeIndexOnlyPackage) Language() string { return "codeindex-fallback-lang" }
func (codeIndexOnlyPackage) Extensions() []string {
	return []string{".ci-fallback"}
}
func (codeIndexOnlyPackage) Detect(string) codeindex.DetectResult { return codeindex.DetectResult{} }
func (codeIndexOnlyPackage) IndexFile(codeindex.DiscoveredCodeFile, codeindex.ProjectContext) (*codeindex.Result, error) {
	return &codeindex.Result{}, nil
}
func (codeIndexOnlyPackage) ExtractSemantics(codeindex.DiscoveredCodeFile, codeindex.ProjectContext) ([]codeindex.CodeConcept, error) {
	return nil, nil
}

func TestResolveFallsBackViaCodeIndexLanguage(t *testing.T) {
	codeindex.Register(codeIndexOnlyPackage{})
	Register(testReader{name: "codeindex-fallback-lang", exts: nil})

	got := Resolve("sample.ci-fallback")
	if got.Name() != "codeindex-fallback-lang" {
		t.Fatalf("Resolve codeindex fallback=%q, want codeindex-fallback-lang", got.Name())
	}
}
