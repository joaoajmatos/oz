package gain

import (
	"time"

	"github.com/joaoajmatos/oz/internal/shell/track"
)

func Aggregate(runs []track.Run, retentionDays int, now time.Time) Report {
	report := Report{
		RetentionDays:    retentionDays,
		WindowStartEpoch: 0,
	}
	if retentionDays > 0 {
		report.WindowStartEpoch = now.AddDate(0, 0, -retentionDays).Unix()
	}

	var reductionSum float64
	var durationSum int64
	for _, run := range runs {
		report.InvocationCount++
		report.TokenBeforeTotal += run.TokenBefore
		report.TokenAfterTotal += run.TokenAfter
		report.TokenSavedTotal += run.TokenSaved
		reductionSum += run.ReductionPct
		durationSum += run.DurationMs
	}
	if report.InvocationCount > 0 {
		report.ReductionPctAvg = reductionSum / float64(report.InvocationCount)
		report.DurationMsAvg = float64(durationSum) / float64(report.InvocationCount)
	}
	return report
}
