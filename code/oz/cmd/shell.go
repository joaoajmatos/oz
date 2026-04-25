package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/joaoajmatos/oz/internal/shell/gain"
	shellrun "github.com/joaoajmatos/oz/internal/shell/run"
	"github.com/joaoajmatos/oz/internal/shell/track"
	"github.com/spf13/cobra"
)

var (
	shellMode, shellTee              string
	shellJSON, shellNoTrack          bool
	shellUltraCompact, shellGainJSON bool
	shellGainAllTime                 bool
	shellGainDays                    int
	shellGainPeriod                  string
	shellVerbosity                   int
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Run and track shell commands with output compression",
}

var shellRunCmd = &cobra.Command{
	Use:          "run -- <cmd...>",
	Short:        "Run a shell command and optionally compress its output",
	Args:         cobra.ArbitraryArgs,
	SilenceUsage: true,
	RunE:         runShellRun,
}

var shellGainCmd = &cobra.Command{
	Use:   "gain",
	Short: "Show cumulative token savings from tracked shell runs",
	RunE:  runShellGain,
}

func runShellRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = cmd.Flags().Args()
	}
	if len(args) == 0 {
		return fmt.Errorf("missing command after --")
	}
	mode := strings.TrimSpace(shellMode)
	if mode == "" {
		mode = "compact"
	}
	if mode != "compact" && mode != "raw" {
		return fmt.Errorf("invalid mode %q (must be compact|raw)", mode)
	}

	result, err := shellrun.Execute(args, shellrun.Options{
		Mode:         mode,
		TeeMode:      shellTee,
		NoTrack:      shellNoTrack,
		UltraCompact: shellUltraCompact,
	})
	if err != nil {
		return err
	}

	if shellJSON {
		data, err := json.MarshalIndent(result.Envelope, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal shell json: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	} else {
		if result.Envelope.Stdout != "" {
			fmt.Fprintln(cmd.OutOrStdout(), result.Envelope.Stdout)
		}
		if result.Envelope.Stderr != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), result.Envelope.Stderr)
		}
		if result.Envelope.RawOutputRef != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "raw output saved: %s\n", *result.Envelope.RawOutputRef)
		}
	}

	if result.ExitCode != 0 {
		return shellExitError{code: result.ExitCode}
	}
	return nil
}

func runShellGain(cmd *cobra.Command, _ []string) error {
	store, err := track.Open(track.DefaultPath())
	if err != nil {
		return fmt.Errorf("open tracking store: %w", err)
	}
	defer func() {
		_ = store.Close()
	}()

	now := time.Now()
	days := shellGainDays
	if shellGainAllTime {
		days = 0
	}
	runs, err := store.QuerySinceDays(days, now)
	if err != nil {
		return fmt.Errorf("query gain runs: %w", err)
	}

	period := gain.Period(shellGainPeriod)
	switch period {
	case gain.PeriodDaily, gain.PeriodWeekly, gain.PeriodMonthly:
	default:
		return fmt.Errorf("invalid period %q (must be daily|weekly|monthly)", shellGainPeriod)
	}

	report := gain.BuildDetailed(runs, days, period, now)

	if shellGainJSON {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal gain json: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	if report.Summary.Empty() {
		if days == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "oz shell gain: no tracked runs")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "oz shell gain: no tracked runs in the last %d days\n", days)
		}
		return nil
	}

	windowLabel := fmt.Sprintf("%d days", report.Summary.RetentionDays)
	if report.Summary.RetentionDays == 0 {
		windowLabel = "all-time"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "oz shell gain (%s, %s)\n", windowLabel, report.Period)
	fmt.Fprintf(cmd.OutOrStdout(), "  invocations: %d\n", report.Summary.InvocationCount)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens before: %d\n", report.Summary.TokenBeforeTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens after: %d\n", report.Summary.TokenAfterTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens saved: %d\n", report.Summary.TokenSavedTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  avg reduction: %.2f%%\n", report.Summary.ReductionPctAvg)
	fmt.Fprintf(cmd.OutOrStdout(), "  avg duration: %.2fms\n", report.Summary.DurationMsAvg)

	if len(report.Trend) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintln(cmd.OutOrStdout(), "Trend")
		fmt.Fprintln(cmd.OutOrStdout(), "  bucket        invocations  saved   avg_reduction")
		for _, row := range report.Trend {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s  %-11d  %-6d  %6.2f%%\n", row.Label, row.InvocationCount, row.TokenSavedTotal, row.ReductionPctAvg)
		}
	}

	if len(report.CommandBreakdown) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintln(cmd.OutOrStdout(), "Command breakdown")
		fmt.Fprintln(cmd.OutOrStdout(), "  command                      invocations  saved   avg_reduction")
		for _, row := range report.CommandBreakdown {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-27s  %-11d  %-6d  %6.2f%%\n", truncateRunes(row.Command, 27), row.InvocationCount, row.TokenSavedTotal, row.ReductionPctAvg)
		}
	}

	if len(report.TopSavers) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintln(cmd.OutOrStdout(), "Top savers")
		for i, row := range report.TopSavers {
			fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s (%d saved)\n", i+1, row.Command, row.TokenSavedTotal)
		}
	}
	return nil
}

type shellExitError struct {
	code int
}

func (e shellExitError) Error() string {
	return fmt.Sprintf("shell command exited with code %d", e.code)
}

func (e shellExitError) ExitCode() int {
	return e.code
}

func init() {
	shellRunCmd.Flags().StringVar(&shellMode, "mode", "compact", "output mode: compact|raw")
	shellRunCmd.Flags().BoolVar(&shellJSON, "json", false, "emit JSON envelope")
	shellRunCmd.Flags().BoolVar(&shellNoTrack, "no-track", false, "skip tracking DB write")
	shellRunCmd.Flags().StringVar(&shellTee, "tee", "failures", "tee raw output: failures|always|never")
	shellRunCmd.Flags().CountVarP(&shellVerbosity, "verbose", "v", "verbosity (-v, -vv, -vvv)")
	shellRunCmd.Flags().BoolVarP(&shellUltraCompact, "ultra-compact", "u", false, "maximum token reduction")
	shellGainCmd.Flags().BoolVar(&shellGainJSON, "json", false, "emit JSON")
	shellGainCmd.Flags().IntVar(&shellGainDays, "days", 90, "retention window in days (0 = all)")
	shellGainCmd.Flags().BoolVar(&shellGainAllTime, "all-time", false, "use all tracked history")
	shellGainCmd.Flags().StringVar(&shellGainPeriod, "period", "daily", "trend period: daily|weekly|monthly")
	shellCmd.AddCommand(shellRunCmd, shellGainCmd)
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return string(runes[:n])
	}
	return string(runes[:n-1]) + "…"
}
