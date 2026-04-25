package codeindex_test

import (
	"testing"

	"github.com/joaoajmatos/oz/internal/codeindex"
)

// fakePackage is a minimal LanguagePackage for registry tests.
type fakePackage struct {
	lang       string
	confidence float64
}

func (f *fakePackage) Language() string   { return f.lang }
func (f *fakePackage) Extensions() []string { return []string{"." + f.lang} }
func (f *fakePackage) Detect(_ string) codeindex.DetectResult {
	return codeindex.DetectResult{Confidence: f.confidence}
}
func (f *fakePackage) IndexFile(_ codeindex.DiscoveredCodeFile, _ codeindex.ProjectContext) (*codeindex.Result, error) {
	return &codeindex.Result{}, nil
}
func (f *fakePackage) ExtractSemantics(_ codeindex.DiscoveredCodeFile, _ codeindex.ProjectContext) ([]codeindex.CodeConcept, error) {
	return nil, nil
}

func TestRegister_DeduplicatesBySameLanguage(t *testing.T) {
	// Use a fresh registry state by registering two packages with the same language.
	// The All() count check relies on the fact that duplicate language is silently dropped.
	before := len(codeindex.All())

	a := &fakePackage{lang: "testlang-dup", confidence: 1.0}
	b := &fakePackage{lang: "testlang-dup", confidence: 1.0}
	codeindex.Register(a)
	codeindex.Register(b)

	after := len(codeindex.All())
	if after != before+1 {
		t.Errorf("All() count: got %d, want %d (duplicate should be ignored)", after, before+1)
	}
}

func TestDetect_ReturnsOnlyActivePackages(t *testing.T) {
	active := &fakePackage{lang: "testlang-active", confidence: 1.0}
	inactive := &fakePackage{lang: "testlang-inactive", confidence: 0}
	codeindex.Register(active)
	codeindex.Register(inactive)

	results := codeindex.Detect(t.TempDir())
	foundActive, foundInactive := false, false
	for _, p := range results {
		switch p.Language() {
		case "testlang-active":
			foundActive = true
		case "testlang-inactive":
			foundInactive = true
		}
	}
	if !foundActive {
		t.Error("Detect: expected testlang-active in results")
	}
	if foundInactive {
		t.Error("Detect: testlang-inactive should not appear (Confidence == 0)")
	}
}

func TestAll_ReturnsAllRegistered(t *testing.T) {
	before := len(codeindex.All())
	codeindex.Register(&fakePackage{lang: "testlang-all1", confidence: 1.0})
	codeindex.Register(&fakePackage{lang: "testlang-all2", confidence: 0})
	after := len(codeindex.All())
	if after != before+2 {
		t.Errorf("All() count: got %d, want %d", after, before+2)
	}
}
