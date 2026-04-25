package readfilter

import (
	"errors"
	"strings"
	"testing"
)

type emptyReader struct{}

func (emptyReader) Name() string { return "empty-reader" }
func (emptyReader) Extensions() []string {
	return []string{".empty-reader"}
}
func (emptyReader) Filter(_ string, _ Options) (string, error) { return "", nil }

type expandReader struct{}

func (expandReader) Name() string        { return "expand-reader" }
func (expandReader) Extensions() []string { return []string{".expand-reader"} }
func (expandReader) Filter(content string, _ Options) (string, error) {
	return content + "\n\n\nextra padding to make output larger than input\n\n\n", nil
}

type errReader struct{}

func (errReader) Name() string { return "err-reader" }
func (errReader) Extensions() []string {
	return []string{".err-reader"}
}
func (errReader) Filter(_ string, _ Options) (string, error) { return "", errors.New("boom") }

func TestRunSafetyFallbackWhenReaderEmptiesInput(t *testing.T) {
	Register(emptyReader{})
	res, err := Run(Options{
		Path:    "sample.empty-reader",
		Content: "package main\nfunc main() {}\n",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Content != "package main\nfunc main() {}\n" {
		t.Fatalf("content=%q, want raw fallback", res.Content)
	}
	if len(res.Warnings) == 0 || !strings.Contains(res.Warnings[0], "emptied non-empty input") {
		t.Fatalf("warnings=%v, expected empty-input fallback warning", res.Warnings)
	}
}

func TestRunSafetyFallbackWhenReaderExpandsInput(t *testing.T) {
	Register(expandReader{})
	const raw = "alpha\nbeta\n"
	res, err := Run(Options{
		Path:    "sample.expand-reader",
		Content: raw,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Content != raw {
		t.Fatalf("content=%q, want raw fallback when filter expands", res.Content)
	}
}

func TestRunReaderErrorFallsBackToRaw(t *testing.T) {
	Register(errReader{})
	res, err := Run(Options{
		Path:    "sample.err-reader",
		Content: "alpha\nbeta\n",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Content != "alpha\nbeta\n" {
		t.Fatalf("content=%q, want raw fallback", res.Content)
	}
	if len(res.Warnings) == 0 || !strings.Contains(res.Warnings[0], "failed") {
		t.Fatalf("warnings=%v, expected filter failure warning", res.Warnings)
	}
}
