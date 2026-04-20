package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/oz-tools/oz/internal/audit"
	auditreport "github.com/oz-tools/oz/internal/audit/report"
	"github.com/spf13/cobra"
)

// errAuditFailed is returned when the exit-on policy triggers.
// main.go translates any non-nil error to exit code 1.
var errAuditFailed = errors.New("audit failed")

// Package-level variables bound to persistent flags so subcommands can read them.
var (
	auditJSON     bool
	auditSeverity string
	auditExitOn   string
	auditOnly     string
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

Subcommands run individual checks; 'audit' with no subcommand runs all checks.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          runAuditAll,
}

func init() {
	auditCmd.PersistentFlags().BoolVar(&auditJSON, "json", false, "output machine-readable JSON")
	auditCmd.PersistentFlags().StringVar(&auditSeverity, "severity", "info", "minimum severity to display (error|warn|info)")
	auditCmd.PersistentFlags().StringVar(&auditExitOn, "exit-on", "error", "set exit code on severity (error|warn|none)")
	auditCmd.Flags().StringVar(&auditOnly, "only", "", "comma-separated list of check names to run")

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

	r, err := audit.RunAll(root, checks, audit.Options{})
	if err != nil {
		return err
	}

	if err := renderReport(cmd, r); err != nil {
		return err
	}

	return applyExitPolicy(r, auditExitOn)
}

// runSingleCheck is a shared helper for subcommands that run a single check.
func runSingleCheck(cmd *cobra.Command, c audit.Check) error {
	if err := validateExitOn(auditExitOn); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return errAuditFailed
	}

	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	r, err := audit.RunAll(root, []audit.Check{c}, audit.Options{})
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
	if auditJSON {
		return auditreport.WriteJSON(cmd.OutOrStdout(), r)
	}
	auditreport.WriteHuman(cmd.OutOrStdout(), r, audit.Severity(auditSeverity))
	return nil
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
