package gain_test

import (
	"testing"
	"time"

	"github.com/joaoajmatos/oz/internal/shell/gain"
	"github.com/joaoajmatos/oz/internal/shell/track"
)

func TestAggregateEmpty(t *testing.T) {
	t.Parallel()

	report := gain.Aggregate(nil, 90, time.Unix(1700000000, 0))
	if !report.Empty() {
		t.Fatalf("expected empty report")
	}
	if report.RetentionDays != 90 {
		t.Fatalf("RetentionDays=%d, want 90", report.RetentionDays)
	}
}

func TestAggregateTotals(t *testing.T) {
	t.Parallel()

	runs := []track.Run{
		{TokenBefore: 100, TokenAfter: 40, TokenSaved: 60, ReductionPct: 60, DurationMs: 20},
		{TokenBefore: 200, TokenAfter: 80, TokenSaved: 120, ReductionPct: 60, DurationMs: 40},
	}
	report := gain.Aggregate(runs, 30, time.Unix(1700000000, 0))
	if report.InvocationCount != 2 {
		t.Fatalf("InvocationCount=%d, want 2", report.InvocationCount)
	}
	if report.TokenSavedTotal != 180 {
		t.Fatalf("TokenSavedTotal=%d, want 180", report.TokenSavedTotal)
	}
	if report.ReductionPctAvg != 60 {
		t.Fatalf("ReductionPctAvg=%f, want 60", report.ReductionPctAvg)
	}
	if report.DurationMsAvg != 30 {
		t.Fatalf("DurationMsAvg=%f, want 30", report.DurationMsAvg)
	}
}
