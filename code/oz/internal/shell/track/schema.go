package track

type Run struct {
	ID            int64
	Command       string
	Session       string
	RecordedAt    int64 // unix epoch seconds
	DurationMs    int64
	TokenBefore   int64
	TokenAfter    int64
	TokenSaved    int64
	ReductionPct  float64
	MatchedFilter string
	ExitCode      int
}

type QueryOpts struct {
	Limit   int
	Since   int64  // epoch; 0 = no lower bound
	Command string // substring; "" = all
	Session string // exact match; "" = all
}

const createRunsTable = `
CREATE TABLE IF NOT EXISTS runs (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    command        TEXT    NOT NULL,
    session        TEXT    NOT NULL DEFAULT '',
    recorded_at    INTEGER NOT NULL,
    duration_ms    INTEGER NOT NULL,
    token_before   INTEGER NOT NULL,
    token_after    INTEGER NOT NULL,
    token_saved    INTEGER NOT NULL,
    reduction_pct  REAL    NOT NULL,
    matched_filter TEXT    NOT NULL,
    exit_code      INTEGER NOT NULL
);`
