package audit_test

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/oz-tools/oz/internal/audit"
)

// fakeCheck returns a fixed set of findings.
type fakeCheck struct {
	name     string
	findings []audit.Finding
}

func (f *fakeCheck) Name() string  { return f.name }
func (f *fakeCheck) Codes() []string { return nil }
func (f *fakeCheck) Run(_ string, _ audit.Options) ([]audit.Finding, error) {
	return f.findings, nil
}

// TestRunAllCounts asserts that Counts always contains all three severity keys
// and that Findings is a non-nil slice even when there are no findings.
func TestRunAllCounts(t *testing.T) {
	r, err := audit.RunAll(".", []audit.Check{&fakeCheck{name: "empty"}}, audit.Options{})
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}

	for _, sev := range []audit.Severity{audit.SeverityError, audit.SeverityWarn, audit.SeverityInfo} {
		if _, ok := r.Counts[sev]; !ok {
			t.Errorf("Counts missing key %q", sev)
		}
	}

	if r.Findings == nil {
		t.Error("Findings must be non-nil (want [] not null in JSON)")
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(r.Findings))
	}
	if r.SchemaVersion != "1" {
		t.Errorf("SchemaVersion = %q, want %q", r.SchemaVersion, "1")
	}
}

// TestSorter creates a shuffled slice of findings covering all orderable
// fields and verifies that RunAll produces a stable sorted order across
// 100 different shuffles.
func TestSorter(t *testing.T) {
	canonical := []audit.Finding{
		{Check: "a", Code: "A001", Severity: audit.SeverityError, File: "x.go", Line: 1, Message: "alpha"},
		{Check: "a", Code: "A001", Severity: audit.SeverityError, File: "x.go", Line: 1, Message: "beta"},
		{Check: "a", Code: "A001", Severity: audit.SeverityError, File: "x.go", Line: 2, Message: "alpha"},
		{Check: "a", Code: "A001", Severity: audit.SeverityError, File: "y.go", Line: 1, Message: "alpha"},
		{Check: "a", Code: "A002", Severity: audit.SeverityError, File: "x.go", Line: 1, Message: "alpha"},
		{Check: "b", Code: "B001", Severity: audit.SeverityError, File: "x.go", Line: 1, Message: "alpha"},
		{Check: "a", Code: "A001", Severity: audit.SeverityWarn, File: "x.go", Line: 1, Message: "alpha"},
		{Check: "a", Code: "A001", Severity: audit.SeverityInfo, File: "x.go", Line: 1, Message: "alpha"},
	}

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 100; i++ {
		shuffled := make([]audit.Finding, len(canonical))
		copy(shuffled, canonical)
		rng.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })

		r, err := audit.RunAll(".", []audit.Check{&fakeCheck{name: "sorter", findings: shuffled}}, audit.Options{})
		if err != nil {
			t.Fatalf("shuffle %d: RunAll: %v", i, err)
		}

		for j, f := range r.Findings {
			if !reflect.DeepEqual(f, canonical[j]) {
				t.Fatalf("shuffle %d: finding[%d] = %+v, want %+v", i, j, f, canonical[j])
			}
		}
	}
}

// TestRunAllCountsNonEmpty verifies counts are correct with mixed severities.
func TestRunAllCountsNonEmpty(t *testing.T) {
	findings := []audit.Finding{
		{Check: "x", Code: "X001", Severity: audit.SeverityError, Message: "e1"},
		{Check: "x", Code: "X002", Severity: audit.SeverityError, Message: "e2"},
		{Check: "x", Code: "X003", Severity: audit.SeverityWarn, Message: "w1"},
		{Check: "x", Code: "X004", Severity: audit.SeverityInfo, Message: "i1"},
	}

	r, err := audit.RunAll(".", []audit.Check{&fakeCheck{name: "mixed", findings: findings}}, audit.Options{})
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}

	if got := r.Counts[audit.SeverityError]; got != 2 {
		t.Errorf("Counts[error] = %d, want 2", got)
	}
	if got := r.Counts[audit.SeverityWarn]; got != 1 {
		t.Errorf("Counts[warn] = %d, want 1", got)
	}
	if got := r.Counts[audit.SeverityInfo]; got != 1 {
		t.Errorf("Counts[info] = %d, want 1", got)
	}
}
