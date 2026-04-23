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

func TestWriteHumanStyled_TruncatesLongSection(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	var findings []audit.Finding
	for i := 0; i < maxFindingsPerSeverityStyled+3; i++ {
		findings = append(findings, audit.Finding{
			Code:     "I001",
			Severity: audit.SeverityInfo,
			Message:  "msg",
			File:     "f.md",
			Line:     i,
		})
	}
	r := &audit.Report{
		SchemaVersion: "1",
		Counts: map[audit.Severity]int{
			audit.SeverityError: 0,
			audit.SeverityWarn:  0,
			audit.SeverityInfo:  len(findings),
		},
		Findings: findings,
	}
	WriteHumanStyled(&buf, r, audit.SeverityInfo)
	s := buf.String()
	if !strings.Contains(s, "showing 25 of 28") {
		t.Fatalf("expected section cap hint, got:\n%s", s)
	}
	if !strings.Contains(s, "… and 3 more") {
		t.Fatalf("expected overflow line, got:\n%s", s)
	}
}
