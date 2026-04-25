package gain

import (
	"fmt"
	"sort"
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

func BuildDetailed(runs []track.Run, retentionDays int, period Period, now time.Time) DetailedReport {
	summary := Aggregate(runs, retentionDays, now)
	report := DetailedReport{
		Summary: summary,
		Period:  period,
	}
	if len(runs) == 0 {
		return report
	}

	type bucketAgg struct {
		invocations int64
		saved       int64
		reduction   float64
	}
	buckets := map[string]bucketAgg{}
	type cmdAgg struct {
		invocations int64
		saved       int64
		reduction   float64
	}
	commandAgg := map[string]cmdAgg{}
	type filterAgg struct {
		invocations int64
		saved       int64
		reduction   float64
	}
	filterBreakdown := map[string]filterAgg{}
	exitBreakdown := map[int]int64{}
	for _, run := range runs {
		label := bucketLabel(run.RecordedAt, period)
		b := buckets[label]
		b.invocations++
		b.saved += run.TokenSaved
		b.reduction += run.ReductionPct
		buckets[label] = b

		c := commandAgg[run.Command]
		c.invocations++
		c.saved += run.TokenSaved
		c.reduction += run.ReductionPct
		commandAgg[run.Command] = c

		matched := run.MatchedFilter
		if matched == "" {
			matched = "unknown"
		}
		f := filterBreakdown[matched]
		f.invocations++
		f.saved += run.TokenSaved
		f.reduction += run.ReductionPct
		filterBreakdown[matched] = f

		exitBreakdown[run.ExitCode]++
	}

	bucketKeys := make([]string, 0, len(buckets))
	for key := range buckets {
		bucketKeys = append(bucketKeys, key)
	}
	sort.Strings(bucketKeys)
	report.Trend = make([]TrendPoint, 0, len(bucketKeys))
	for _, key := range bucketKeys {
		b := buckets[key]
		avgReduction := 0.0
		if b.invocations > 0 {
			avgReduction = b.reduction / float64(b.invocations)
		}
		report.Trend = append(report.Trend, TrendPoint{
			Label:           key,
			InvocationCount: b.invocations,
			TokenSavedTotal: b.saved,
			ReductionPctAvg: avgReduction,
		})
	}

	commands := make([]CommandStat, 0, len(commandAgg))
	for command, agg := range commandAgg {
		avgReduction := 0.0
		if agg.invocations > 0 {
			avgReduction = agg.reduction / float64(agg.invocations)
		}
		commands = append(commands, CommandStat{
			Command:         command,
			InvocationCount: agg.invocations,
			TokenSavedTotal: agg.saved,
			ReductionPctAvg: avgReduction,
		})
	}

	report.CommandBreakdown = append([]CommandStat(nil), commands...)
	sort.Slice(report.CommandBreakdown, func(i, j int) bool {
		if report.CommandBreakdown[i].Command == report.CommandBreakdown[j].Command {
			return report.CommandBreakdown[i].InvocationCount < report.CommandBreakdown[j].InvocationCount
		}
		return report.CommandBreakdown[i].Command < report.CommandBreakdown[j].Command
	})

	filters := make([]FilterStat, 0, len(filterBreakdown))
	for matchedFilter, agg := range filterBreakdown {
		avgReduction := 0.0
		if agg.invocations > 0 {
			avgReduction = agg.reduction / float64(agg.invocations)
		}
		filters = append(filters, FilterStat{
			MatchedFilter:   matchedFilter,
			InvocationCount: agg.invocations,
			TokenSavedTotal: agg.saved,
			ReductionPctAvg: avgReduction,
		})
	}
	report.FilterBreakdown = append([]FilterStat(nil), filters...)
	sort.Slice(report.FilterBreakdown, func(i, j int) bool {
		if report.FilterBreakdown[i].MatchedFilter == report.FilterBreakdown[j].MatchedFilter {
			return report.FilterBreakdown[i].InvocationCount < report.FilterBreakdown[j].InvocationCount
		}
		return report.FilterBreakdown[i].MatchedFilter < report.FilterBreakdown[j].MatchedFilter
	})

	exitCodes := make([]int, 0, len(exitBreakdown))
	for code := range exitBreakdown {
		exitCodes = append(exitCodes, code)
	}
	sort.Ints(exitCodes)
	report.ExitBreakdown = make([]ExitStat, 0, len(exitCodes))
	for _, code := range exitCodes {
		report.ExitBreakdown = append(report.ExitBreakdown, ExitStat{
			ExitCode:        code,
			InvocationCount: exitBreakdown[code],
		})
	}

	report.TopSavers = append([]CommandStat(nil), commands...)
	sort.Slice(report.TopSavers, func(i, j int) bool {
		if report.TopSavers[i].TokenSavedTotal == report.TopSavers[j].TokenSavedTotal {
			return report.TopSavers[i].Command < report.TopSavers[j].Command
		}
		return report.TopSavers[i].TokenSavedTotal > report.TopSavers[j].TokenSavedTotal
	})
	if len(report.TopSavers) > 5 {
		report.TopSavers = report.TopSavers[:5]
	}

	return report
}

func bucketLabel(epoch int64, period Period) string {
	t := time.Unix(epoch, 0).UTC()
	switch period {
	case PeriodWeekly:
		year, week := t.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", year, week)
	case PeriodMonthly:
		return t.Format("2006-01")
	default:
		return t.Format("2006-01-02")
	}
}
