// Package audit defines the core types and runner for oz workspace audit checks.
package audit

import "sort"

// Severity is the severity level of an audit finding.
type Severity string

const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warn"
	SeverityInfo  Severity = "info"
)

// Finding is a single diagnostic produced by an audit check.
type Finding struct {
	Check    string   `json:"check"`
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	File     string   `json:"file,omitempty"`
	Line     int      `json:"line,omitempty"`
	Hint     string   `json:"hint,omitempty"`
	Refs     []string `json:"refs,omitempty"`
}

// Report is the aggregated result of running one or more checks.
type Report struct {
	SchemaVersion string           `json:"schema_version"`
	Counts        map[Severity]int `json:"counts"`
	Findings      []Finding        `json:"findings"`
}

// Options configures which workspace content checks should inspect.
type Options struct {
	IncludeTests bool
	IncludeDocs  bool
}

// Check is a single named audit check that can produce findings.
type Check interface {
	Name() string
	Codes() []string
	Run(root string, opts Options) ([]Finding, error)
}

func severityRank(s Severity) int {
	switch s {
	case SeverityError:
		return 0
	case SeverityWarn:
		return 1
	default:
		return 2
	}
}

// RunAll runs every check in order, aggregates their findings, and returns a
// sorted Report. The sort key is (severityRank, check, code, file, line, message).
// Counts always contains all three severity keys, even when zero.
// Findings is always a non-nil slice so JSON renders [] not null.
func RunAll(root string, checks []Check, opts Options) (*Report, error) {
	var all []Finding
	for _, c := range checks {
		findings, err := c.Run(root, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, findings...)
	}

	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if ra, rb := severityRank(a.Severity), severityRank(b.Severity); ra != rb {
			return ra < rb
		}
		if a.Check != b.Check {
			return a.Check < b.Check
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Message < b.Message
	})

	counts := map[Severity]int{
		SeverityError: 0,
		SeverityWarn:  0,
		SeverityInfo:  0,
	}
	for _, f := range all {
		counts[f.Severity]++
	}

	findings := all
	if findings == nil {
		findings = []Finding{}
	}

	return &Report{
		SchemaVersion: "1",
		Counts:        counts,
		Findings:      findings,
	}, nil
}
