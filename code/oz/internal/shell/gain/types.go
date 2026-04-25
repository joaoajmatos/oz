package gain

type Period string

const (
	PeriodDaily   Period = "daily"
	PeriodWeekly  Period = "weekly"
	PeriodMonthly Period = "monthly"
)

type Report struct {
	InvocationCount  int64   `json:"invocation_count"`
	TokenBeforeTotal int64   `json:"token_before_total"`
	TokenAfterTotal  int64   `json:"token_after_total"`
	TokenSavedTotal  int64   `json:"token_saved_total"`
	ReductionPctAvg  float64 `json:"reduction_pct_avg"`
	DurationMsAvg    float64 `json:"duration_ms_avg"`
	RetentionDays    int     `json:"retention_days"`
	WindowStartEpoch int64   `json:"window_start_epoch"`
}

type TrendPoint struct {
	Label           string  `json:"label"`
	InvocationCount int64   `json:"invocation_count"`
	TokenSavedTotal int64   `json:"token_saved_total"`
	ReductionPctAvg float64 `json:"reduction_pct_avg"`
}

type CommandStat struct {
	Command         string  `json:"command"`
	InvocationCount int64   `json:"invocation_count"`
	TokenSavedTotal int64   `json:"token_saved_total"`
	ReductionPctAvg float64 `json:"reduction_pct_avg"`
}

type FilterStat struct {
	MatchedFilter   string  `json:"matched_filter"`
	InvocationCount int64   `json:"invocation_count"`
	TokenSavedTotal int64   `json:"token_saved_total"`
	ReductionPctAvg float64 `json:"reduction_pct_avg"`
}

type ExitStat struct {
	ExitCode        int   `json:"exit_code"`
	InvocationCount int64 `json:"invocation_count"`
}

type DetailedReport struct {
	Summary          Report        `json:"summary"`
	Period           Period        `json:"period"`
	Trend            []TrendPoint  `json:"trend"`
	CommandBreakdown []CommandStat `json:"command_breakdown"`
	FilterBreakdown  []FilterStat  `json:"filter_breakdown"`
	ExitBreakdown    []ExitStat    `json:"exit_breakdown"`
	TopSavers        []CommandStat `json:"top_savers"`
}

func (r Report) Empty() bool {
	return r.InvocationCount == 0
}
