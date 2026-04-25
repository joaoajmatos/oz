package gain

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

func (r Report) Empty() bool {
	return r.InvocationCount == 0
}
