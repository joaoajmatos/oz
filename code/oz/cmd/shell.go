package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/joaoajmatos/oz/internal/shell/readfilter/langs"

	"github.com/joaoajmatos/oz/internal/shell/gain"
	"github.com/joaoajmatos/oz/internal/shell/hooks"
	"github.com/joaoajmatos/oz/internal/shell/readfilter"
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
	shellReadMaxLines                int
	shellReadTailLines               int
	shellReadLineNumbers             bool
	shellReadNoFilter                bool
	shellReadJSON                    bool
	shellReadUltraCompact            bool
	shellReadNoTrack                 bool
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

var shellReadCmd = &cobra.Command{
	Use:          "read <file>... | -",
	Short:        "Read files or stdin with language-aware filtering",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE:         runShellRead,
}

var shellRewriteExclude []string

type shellReadEnvelope struct {
	SchemaVersion     string   `json:"schema_version"`
	Path              string   `json:"path"`
	Lang              string   `json:"lang"`
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
}

var shellRewriteCmd = &cobra.Command{
	Use:          "rewrite <command>",
	Short:        "Rewrite a command through shell compression hooks",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runShellRewrite,
}

func runShellRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = cmd.Flags().Args()
	}
	if len(args) == 0 {
		return fmt.Errorf("missing command after --")
	}
	mode := shellrun.Mode(strings.TrimSpace(shellMode))
	if mode == "" {
		mode = shellrun.ModeCompact
	}
	if mode != shellrun.ModeCompact && mode != shellrun.ModeRaw {
		return fmt.Errorf("invalid mode %q (must be compact|raw)", mode)
	}

	result, err := shellrun.Execute(args, shellrun.Options{
		Mode:         mode,
		TeeMode:      shellrun.TeeMode(shellTee),
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

func runShellRewrite(cmd *cobra.Command, args []string) error {
	cfg := hooks.RewriteConfig()
	if len(shellRewriteExclude) > 0 {
		cfg.ExcludeCommands = append([]string(nil), shellRewriteExclude...)
	}

	decision := hooks.Decide(args[0], cfg)
	if decision.Allowed && decision.Rewritten != "" {
		fmt.Fprintln(cmd.OutOrStdout(), decision.Rewritten)
		return nil
	}

	switch decision.Reason {
	case hooks.ReasonCommandExcluded, hooks.ReasonHooksDisabled:
		return shellExitError{code: 2}
	default:
		return shellExitError{code: 1}
	}
}

func runShellRead(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("max-lines") && cmd.Flags().Changed("tail-lines") {
		return fmt.Errorf("--max-lines and --tail-lines are mutually exclusive")
	}
	var maxLines *int
	var tailLines *int
	if cmd.Flags().Changed("max-lines") {
		maxLines = &shellReadMaxLines
	}
	if cmd.Flags().Changed("tail-lines") {
		tailLines = &shellReadTailLines
	}

	start := time.Now()
	stdinConsumed := false
	stdinContent := ""
	results := make([]readfilter.Result, 0, len(args))
	missing := make([]string, 0)
	globalWarnings := make([]string, 0)
	for _, target := range args {
		content := ""
		path := target
		stdin := false
		if target == "-" {
			stdin = true
			path = "-"
			if stdinConsumed {
				globalWarnings = append(globalWarnings, "duplicate stdin marker '-' ignored")
				continue
			}
			if stdinContent == "" {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
				stdinContent = string(data)
			}
			stdinConsumed = true
			content = stdinContent
		} else {
			abs := target
			if !filepath.IsAbs(target) {
				if wd, err := os.Getwd(); err == nil {
					abs = filepath.Join(wd, target)
				}
			}
			data, err := os.ReadFile(abs)
			if err != nil {
				missing = append(missing, target)
				continue
			}
			content = string(data)
			path = target
		}

		res, err := readfilter.Run(readfilter.Options{
			Path:         path,
			Stdin:        stdin,
			Content:      content,
			MaxLines:     maxLines,
			TailLines:    tailLines,
			LineNumbers:  shellReadLineNumbers,
			NoFilter:     shellReadNoFilter,
			UltraCompact: shellReadUltraCompact,
		})
		if err != nil {
			return err
		}
		results = append(results, res)
	}

	if shellReadJSON {
		payload := make([]shellReadEnvelope, 0, len(results))
		for _, res := range results {
			saved := res.TokenEstBefore - res.TokenEstAfter
			reduction := 0.0
			if res.TokenEstBefore > 0 {
				reduction = (float64(saved) / float64(res.TokenEstBefore)) * 100
			}
			payload = append(payload, shellReadEnvelope{
				SchemaVersion:     "1",
				Path:              res.Path,
				Lang:              res.Language,
				MatchedFilter:     "read." + res.Language,
				ExitCode:          0,
				DurationMs:        time.Since(start).Milliseconds(),
				TokenEstBefore:    res.TokenEstBefore,
				TokenEstAfter:     res.TokenEstAfter,
				TokenEstSaved:     saved,
				TokenReductionPct: reduction,
				Stdout:            res.Content,
				Stderr:            "",
				Warnings:          append([]string(nil), res.Warnings...),
			})
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal read json: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	} else {
		for i, res := range results {
			if len(results) > 1 {
				fmt.Fprintf(cmd.OutOrStdout(), "==> %s <==\n", res.Path)
			}
			if res.Content != "" {
				fmt.Fprintln(cmd.OutOrStdout(), res.Content)
			}
			for _, warning := range res.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning)
			}
			if i < len(results)-1 {
				fmt.Fprintln(cmd.OutOrStdout())
			}
		}
		for _, warning := range globalWarnings {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning)
		}
	}

	if !shellReadNoTrack {
		var beforeTotal, afterTotal int64
		for _, res := range results {
			beforeTotal += int64(res.TokenEstBefore)
			afterTotal += int64(res.TokenEstAfter)
		}
		matched := "read"
		if len(results) > 0 {
			langs := make([]string, 0, len(results))
			for _, res := range results {
				langs = append(langs, res.Language)
			}
			matched = "read." + strings.Join(langs, ",")
		}
		_ = insertShellTrackRow(formatShellReadTrackedCommand(args), beforeTotal, afterTotal, time.Since(start).Milliseconds(), matched, len(missing))
	}

	for _, miss := range missing {
		fmt.Fprintf(cmd.ErrOrStderr(), "read: missing file %s\n", miss)
	}
	if len(missing) > 0 {
		return shellExitError{code: 2}
	}
	return nil
}

func insertShellTrackRow(command string, before, after int64, durationMs int64, matchedFilter string, missingCount int) error {
	store, err := track.Open(track.DefaultPath())
	if err != nil {
		return err
	}
	defer func() {
		_ = store.Close()
	}()
	exitCode := 0
	if missingCount > 0 {
		exitCode = 2
	}
	saved := before - after
	reduction := 0.0
	if before > 0 {
		reduction = (float64(saved) / float64(before)) * 100
	}
	return store.Insert(track.Run{
		Command:       command,
		RecordedAt:    time.Now().Unix(),
		DurationMs:    durationMs,
		TokenBefore:   before,
		TokenAfter:    after,
		TokenSaved:    saved,
		ReductionPct:  reduction,
		MatchedFilter: matchedFilter,
		ExitCode:      exitCode,
	})
}

func formatShellReadTrackedCommand(args []string) string {
	if len(args) == 0 {
		return "oz shell read"
	}
	return "oz shell read -- " + strings.Join(args, " ")
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
	shellRunCmd.Flags().StringVar(&shellMode, "mode", string(shellrun.ModeCompact), "output mode: compact|raw")
	shellRunCmd.Flags().BoolVar(&shellJSON, "json", false, "emit JSON envelope")
	shellRunCmd.Flags().BoolVar(&shellNoTrack, "no-track", false, "skip tracking DB write")
	shellRunCmd.Flags().StringVar(&shellTee, "tee", string(shellrun.TeeModeFailures), "tee raw output: failures|always|never")
	shellRunCmd.Flags().CountVarP(&shellVerbosity, "verbose", "v", "verbosity (-v, -vv, -vvv)")
	shellRunCmd.Flags().BoolVarP(&shellUltraCompact, "ultra-compact", "u", false, "maximum token reduction")
	shellGainCmd.Flags().BoolVar(&shellGainJSON, "json", false, "emit JSON")
	shellGainCmd.Flags().IntVar(&shellGainDays, "days", 90, "retention window in days (0 = all)")
	shellGainCmd.Flags().BoolVar(&shellGainAllTime, "all-time", false, "use all tracked history")
	shellGainCmd.Flags().StringVar(&shellGainPeriod, "period", string(gain.PeriodDaily), "trend period: daily|weekly|monthly")
	shellReadCmd.Flags().IntVar(&shellReadMaxLines, "max-lines", 0, "show at most N lines")
	shellReadCmd.Flags().IntVar(&shellReadTailLines, "tail-lines", 0, "show last N lines")
	shellReadCmd.Flags().BoolVarP(&shellReadLineNumbers, "line-numbers", "n", false, "include line numbers")
	shellReadCmd.Flags().BoolVar(&shellReadNoFilter, "no-filter", false, "skip language-aware filtering")
	shellReadCmd.Flags().BoolVar(&shellReadJSON, "json", false, "emit JSON output")
	shellReadCmd.Flags().BoolVarP(&shellReadUltraCompact, "ultra-compact", "u", false, "maximum token reduction")
	shellReadCmd.Flags().BoolVar(&shellReadNoTrack, "no-track", false, "skip tracking DB write")
	shellRewriteCmd.Flags().StringSliceVar(&shellRewriteExclude, "exclude", nil, "commands to bypass rewrite")
	shellCmd.AddCommand(shellRunCmd, shellGainCmd, shellReadCmd, shellRewriteCmd)
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
