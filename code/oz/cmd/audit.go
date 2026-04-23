package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/joaoajmatos/oz/internal/audit"
	auditreport "github.com/joaoajmatos/oz/internal/audit/report"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// errAuditFailed is returned when the exit-on policy triggers.
// main.go translates any non-nil error to exit code 1.
var errAuditFailed = errors.New("audit failed")

// Package-level variables bound to persistent flags so subcommands can read them.
var (
	auditJSON     bool
	auditNoColor  bool
	auditSeverity string
	auditExitOn   string
	auditOnly     string

	// Drift-only options (also apply when running the parent `oz audit` aggregator).
	auditIncludeDriftTests bool
	auditIncludeDriftDocs  bool
)

// registeredChecks holds all checks registered via registerCheck.
var registeredChecks []audit.Check

// registerCheck adds a check to the global registry used by runAuditAll.
func registerCheck(c audit.Check) {
	registeredChecks = append(registeredChecks, c)
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit workspace health",
	Long: `Run audit checks against the workspace and report findings.

The workspace root is found by walking up from the current directory until
AGENTS.md and OZ.md exist, so this works from any subdirectory.

Subcommands run individual checks; 'audit' with no subcommand runs all checks.

Use --json for machine-readable output (automation, CI, and tools that need a
stable schema, including LLM context). Interactive terminals get lipgloss-styled
output (aligned with the rest of the oz CLI) unless output is not a TTY, NO_COLOR
is set, or --no-color is passed; those cases use plain text matching the
pre-styling format. Other commands use charm/huh for interactive forms; audit
stays non-interactive so --json and piped output remain stable.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          runAuditAll,
}

func init() {
	auditCmd.PersistentFlags().BoolVar(&auditJSON, "json", false, "output machine-readable JSON (recommended for tools and LLMs)")
	auditCmd.PersistentFlags().BoolVar(&auditNoColor, "no-color", false, "force plain text; also implied when stdout is not a TTY or NO_COLOR is set")
	auditCmd.PersistentFlags().StringVar(&auditSeverity, "severity", "info", "minimum severity to display (error|warn|info)")
	auditCmd.PersistentFlags().StringVar(&auditExitOn, "exit-on", "error", "set exit code on severity (error|warn|none)")
	auditCmd.Flags().StringVar(&auditOnly, "only", "", "comma-separated list of check names to run")
	auditCmd.PersistentFlags().BoolVar(&auditIncludeDriftTests, "include-tests", false, "drift check: include exported symbols from *_test.go files")
	auditCmd.PersistentFlags().BoolVar(&auditIncludeDriftDocs, "include-docs", false, "drift check: also scan docs/ for code references")

	auditCmd.AddCommand(auditOrphansCmd)
	auditCmd.AddCommand(auditCoverageCmd)
	auditCmd.AddCommand(auditStalenessCmd)
	auditCmd.AddCommand(auditDriftCmd)
	auditCmd.AddCommand(auditSummaryCmd)
}

func runAuditAll(cmd *cobra.Command, _ []string) error {
	if err := validateExitOn(auditExitOn); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return errAuditFailed
	}

	checks, err := resolveChecks(auditOnly)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return errAuditFailed
	}

	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	r, err := audit.RunAll(root, checks, audit.Options{
		IncludeTests: auditIncludeDriftTests,
		IncludeDocs:  auditIncludeDriftDocs,
	})
	if err != nil {
		return err
	}

	if err := renderReport(cmd, r); err != nil {
		return err
	}

	return applyExitPolicy(r, auditExitOn)
}

// runSingleCheck is a shared helper for subcommands that run a single check.
// opts controls optional workspace content inclusion; pass audit.Options{} for defaults.
func runSingleCheck(cmd *cobra.Command, c audit.Check, opts audit.Options) error {
	if err := validateExitOn(auditExitOn); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return errAuditFailed
	}

	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	r, err := audit.RunAll(root, []audit.Check{c}, opts)
	if err != nil {
		return err
	}

	if err := renderReport(cmd, r); err != nil {
		return err
	}

	return applyExitPolicy(r, auditExitOn)
}

func resolveChecks(only string) ([]audit.Check, error) {
	if only == "" {
		return registeredChecks, nil
	}

	names := strings.Split(only, ",")
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[strings.TrimSpace(n)] = true
	}

	known := make(map[string]audit.Check, len(registeredChecks))
	for _, c := range registeredChecks {
		known[c.Name()] = c
	}
	for name := range nameSet {
		if _, ok := known[name]; !ok {
			return nil, fmt.Errorf("unknown check name %q (known: %s)", name, knownCheckNames())
		}
	}

	var out []audit.Check
	for _, c := range registeredChecks {
		if nameSet[c.Name()] {
			out = append(out, c)
		}
	}
	return out, nil
}

func knownCheckNames() string {
	names := make([]string, len(registeredChecks))
	for i, c := range registeredChecks {
		names[i] = c.Name()
	}
	return strings.Join(names, ", ")
}

func validateExitOn(v string) error {
	switch v {
	case "error", "warn", "none":
		return nil
	default:
		return fmt.Errorf("unknown --exit-on value %q (must be error|warn|none)", v)
	}
}

func renderReport(cmd *cobra.Command, r *audit.Report) error {
	out := cmd.OutOrStdout()
	if auditJSON {
		return auditreport.WriteJSON(out, r)
	}
	if shouldUseAuditTTY(out) {
		auditreport.WriteHumanStyled(out, r, audit.Severity(auditSeverity))
		return nil
	}
	auditreport.WriteHuman(out, r, audit.Severity(auditSeverity))
	return nil
}

// shouldUseAuditTTY reports whether styled terminal output is appropriate.
// Favor --json (or NO_COLOR / non-TTY / --no-color) for anything that must not
// contain ANSI or depends on a stable, copy-pastable text layout.
func shouldUseAuditTTY(out io.Writer) bool {
	if auditNoColor {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}

func applyExitPolicy(r *audit.Report, exitOn string) error {
	switch exitOn {
	case "error":
		if r.Counts[audit.SeverityError] > 0 {
			return errAuditFailed
		}
	case "warn":
		if r.Counts[audit.SeverityError]+r.Counts[audit.SeverityWarn] > 0 {
			return errAuditFailed
		}
	}
	return nil
}
