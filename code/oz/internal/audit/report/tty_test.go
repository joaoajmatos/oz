package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/audit"
)

func TestWriteHumanStyled_Sample(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	r := &audit.Report{
		SchemaVersion: "1",
		Counts: map[audit.Severity]int{
			audit.SeverityError: 1,
			audit.SeverityWarn:  0,
			audit.SeverityInfo:  0,
		},
		Findings: []audit.Finding{{
			Check:    "drift",
			Code:     "D001",
			Severity: audit.SeverityError,
			Message:  "message",
			File:     "specs/x.md",
			Line:     1,
		}},
	}
	WriteHumanStyled(&buf, r, audit.SeverityInfo)
	s := buf.String()
	if !strings.Contains(s, "D001") || !strings.Contains(s, "message") {
		t.Fatalf("expected code and message in output:\n%s", s)
	}
}

func TestWriteHumanStyled_Empty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	r := &audit.Report{
		SchemaVersion: "1",
		Counts: map[audit.Severity]int{
			audit.SeverityError: 0,
			audit.SeverityWarn:  0,
			audit.SeverityInfo:  0,
		},
		Findings: nil,
	}
	WriteHumanStyled(&buf, r, audit.SeverityInfo)
	if !strings.Contains(buf.String(), "All clear") {
		t.Fatalf("expected empty state: %q", buf.String())
	}
}
