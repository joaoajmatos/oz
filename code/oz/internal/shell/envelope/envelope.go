package envelope

type RunResult struct {
	SchemaVersion     string   `json:"schema_version"`
	Command           string   `json:"command"`
	Mode              string   `json:"mode"`
	MatchedFilter     string   `json:"matched_filter"`
	ExitCode          int      `json:"exit_code"`
	DurationMs        int64    `json:"duration_ms"`
	TokenEstBefore    int      `json:"token_est_before"`
	TokenEstAfter     int      `json:"token_est_after"`
	TokenEstSaved     int      `json:"token_est_saved"`
	TokenReductionPct float64  `json:"token_reduction_pct"`
	Stdout            string   `json:"stdout"`
	Stderr            string   `json:"stderr"`
	Warnings          []string `json:"warnings"`
	RawOutputRef      *string  `json:"raw_output_ref"`
}
