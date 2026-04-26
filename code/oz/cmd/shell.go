package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	_ "github.com/joaoajmatos/oz/internal/shell/readfilter/langs"
	"github.com/mattn/go-isatty"

	"github.com/joaoajmatos/oz/internal/shell/envelope"
	"github.com/joaoajmatos/oz/internal/shell/filter"
	"github.com/joaoajmatos/oz/internal/shell/gain"
	"github.com/joaoajmatos/oz/internal/shell/hooks"
	"github.com/joaoajmatos/oz/internal/shell/readfilter"
	shellrun "github.com/joaoajmatos/oz/internal/shell/run"
	"github.com/joaoajmatos/oz/internal/shell/track"
	"github.com/joaoajmatos/oz/internal/termstyle"
	"github.com/spf13/cobra"
)

var (
	shellMode, shellTee              string
	shellJSON, shellNoTrack          bool
	shellUltraCompact, shellGainJSON bool
	shellGainAllTime                 bool
	shellGainInteractive             bool
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
	shellPipeFilter                  string
	shellPipePassthrough             bool
	shellPipeJSON                    bool
	shellPipeUltraCompact            bool
	shellPipeNoTrack                 bool
)

const shellPipeRawCap = 2 * 1024 * 1024

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

var shellPipeCmd = &cobra.Command{
	Use:          "pipe",
	Short:        "Compact stdin using shell output filters",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE:         runShellPipe,
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
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", termstyle.Brand.Render("oz shell gain"), termstyle.Subtle.Render(fmt.Sprintf("(%s, %s)", windowLabel, report.Period)))
	fmt.Fprintf(cmd.OutOrStdout(), "  invocations: %d\n", report.Summary.InvocationCount)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens before: %d\n", report.Summary.TokenBeforeTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens after: %d\n", report.Summary.TokenAfterTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens saved: %d\n", report.Summary.TokenSavedTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  avg reduction: %.2f%%\n", report.Summary.ReductionPctAvg)
	fmt.Fprintf(cmd.OutOrStdout(), "  avg duration: %.2fms\n", report.Summary.DurationMsAvg)

	if len(report.Trend) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintln(cmd.OutOrStdout(), termstyle.Section.Render("Trend"))
		fmt.Fprintln(cmd.OutOrStdout(), "  "+termstyle.Subtle.Render("bucket        invocations  saved   avg_reduction"))
		for _, row := range report.Trend {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s  %-11d  %-6d  %6.2f%%\n", row.Label, row.InvocationCount, row.TokenSavedTotal, row.ReductionPctAvg)
		}
	}

	if len(report.CommandBreakdown) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintln(cmd.OutOrStdout(), termstyle.Section.Render("Command breakdown"))
		for _, row := range report.CommandBreakdown {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", termstyle.Command.Render(row.Command))
			fmt.Fprintf(cmd.OutOrStdout(), "    invocations: %-11d  saved: %-8d  avg_reduction: %.2f%%\n", row.InvocationCount, row.TokenSavedTotal, row.ReductionPctAvg)
		}
	}

	if len(report.FilterBreakdown) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintln(cmd.OutOrStdout(), termstyle.Section.Render("Filter breakdown"))
		fmt.Fprintln(cmd.OutOrStdout(), "  "+termstyle.Subtle.Render("filter        invocations  saved   avg_reduction"))
		for _, row := range report.FilterBreakdown {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s  %-11d  %-6d  %6.2f%%\n", row.MatchedFilter, row.InvocationCount, row.TokenSavedTotal, row.ReductionPctAvg)
		}
	}

	if len(report.ExitBreakdown) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintln(cmd.OutOrStdout(), termstyle.Section.Render("Run outcomes"))
		for _, row := range report.ExitBreakdown {
			label := "success"
			if row.ExitCode != 0 {
				label = "non-zero"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  exit=%d (%s): %d\n", row.ExitCode, label, row.InvocationCount)
		}
	}

	topReducers := topByReduction(report.CommandBreakdown, 5)
	if len(topReducers) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintln(cmd.OutOrStdout(), termstyle.Section.Render("Top reducers"))
		for i, row := range topReducers {
			fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s (%d invocations, %.2f%% avg reduction)\n", i+1, row.Command, row.InvocationCount, row.ReductionPctAvg)
		}
	}

	if len(report.TopSavers) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintln(cmd.OutOrStdout(), termstyle.Section.Render("Top savers"))
		for i, row := range report.TopSavers {
			fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s (%d saved)\n", i+1, row.Command, row.TokenSavedTotal)
		}
	}
	if shellGainInteractive {
		if !canRunGainExplorer(cmd) {
			return fmt.Errorf("--interactive requires a TTY for stdin and stdout")
		}
		if err := runShellGainExplorer(cmd, report); err != nil {
			return err
		}
	}
	return nil
}

func canRunGainExplorer(cmd *cobra.Command) bool {
	stdinFile, stdinOK := cmd.InOrStdin().(*os.File)
	stdoutFile, stdoutOK := cmd.OutOrStdout().(*os.File)
	if !stdinOK || !stdoutOK {
		return false
	}
	return isatty.IsTerminal(stdinFile.Fd()) && isatty.IsTerminal(stdoutFile.Fd())
}

type gainViewChoice string

const (
	gainViewSummary   gainViewChoice = "summary"
	gainViewTrend     gainViewChoice = "trend"
	gainViewBreakdown gainViewChoice = "breakdown"
	gainViewReducers  gainViewChoice = "reducers"
	gainViewSavers    gainViewChoice = "savers"
	gainViewQuit      gainViewChoice = "quit"
)

func runShellGainExplorer(cmd *cobra.Command, report gain.DetailedReport) error {
	choice := gainViewSummary
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[gainViewChoice]().
				Title("Inspect shell gain report").
				Description("Choose a section to focus on. Select quit to finish.").
				Options(
					huh.NewOption("Summary", gainViewSummary),
					huh.NewOption("Trend", gainViewTrend),
					huh.NewOption("Command breakdown", gainViewBreakdown),
					huh.NewOption("Top reducers", gainViewReducers),
					huh.NewOption("Top savers", gainViewSavers),
					huh.NewOption("Quit", gainViewQuit),
				).
				Value(&choice),
		),
	).WithTheme(ozTheme())

	for {
		if err := form.Run(); err != nil {
			if err == huh.ErrUserAborted {
				return nil
			}
			return fmt.Errorf("interactive gain explorer: %w", err)
		}
		switch choice {
		case gainViewSummary:
			fmt.Fprintln(cmd.OutOrStdout(), termstyle.Section.Render("Summary focus"))
			fmt.Fprintf(cmd.OutOrStdout(), "  tokens saved: %d  |  avg reduction: %.2f%%  |  avg duration: %.2fms\n", report.Summary.TokenSavedTotal, report.Summary.ReductionPctAvg, report.Summary.DurationMsAvg)
		case gainViewTrend:
			fmt.Fprintln(cmd.OutOrStdout(), termstyle.Section.Render("Trend focus"))
			for _, row := range report.Trend {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s  saved=%d  reduction=%.2f%%\n", row.Label, row.TokenSavedTotal, row.ReductionPctAvg)
			}
		case gainViewBreakdown:
			fmt.Fprintln(cmd.OutOrStdout(), termstyle.Section.Render("Command breakdown focus"))
			for _, row := range report.CommandBreakdown {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", row.Command)
			}
		case gainViewReducers:
			fmt.Fprintln(cmd.OutOrStdout(), termstyle.Section.Render("Top reducers focus"))
			for i, row := range topByReduction(report.CommandBreakdown, 5) {
				fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s (%.2f%%)\n", i+1, row.Command, row.ReductionPctAvg)
			}
		case gainViewSavers:
			fmt.Fprintln(cmd.OutOrStdout(), termstyle.Section.Render("Top savers focus"))
			for i, row := range report.TopSavers {
				fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s (%d saved)\n", i+1, row.Command, row.TokenSavedTotal)
			}
		default:
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}
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
			if !shellReadNoFilter && res.TokenEstBefore > res.TokenEstAfter {
				saved := res.TokenEstBefore - res.TokenEstAfter
				pct := int((float64(saved) / float64(res.TokenEstBefore)) * 100)
				fmt.Fprintf(cmd.ErrOrStderr(), "[oz] %s: %dt omitted (%d%%) — full: oz shell read --no-filter --line-numbers %s\n",
					res.Path, saved, pct, res.Path)
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

func runShellPipe(cmd *cobra.Command, _ []string) error {
	if shellPipePassthrough {
		_, err := io.Copy(cmd.OutOrStdout(), cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("pipe passthrough: %w", err)
		}
		return nil
	}

	var raw strings.Builder
	_, err := io.Copy(&raw, io.LimitReader(cmd.InOrStdin(), shellPipeRawCap+1))
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	input := raw.String()
	if len(input) > shellPipeRawCap {
		return fmt.Errorf("stdin exceeds %d byte limit", shellPipeRawCap)
	}

	args, matched, resolveErr := resolvePipeFilterArgs(shellPipeFilter, input)
	if resolveErr != nil {
		return resolveErr
	}
	outStdout, outStderr, matchedFilter, applyErr := filter.Apply(args, input, "", 0, shellPipeUltraCompact)
	warnings := make([]string, 0)
	if applyErr != nil {
		warnings = append(warnings, fmt.Sprintf("%s failed; falling back to raw output", matched))
		outStdout = input
		outStderr = ""
		matchedFilter = filter.FilterGeneric
	}
	beforeTokens := envelope.EstimateTokens(input)
	afterTokens := envelope.EstimateTokens(outStdout + outStderr)
	tokenSaved := beforeTokens - afterTokens
	reductionPct := 0.0
	if beforeTokens > 0 {
		reductionPct = (float64(tokenSaved) / float64(beforeTokens)) * 100
	}

	if !shellPipeNoTrack {
		_ = insertShellTrackEnvelope("oz shell pipe", 0, int64(beforeTokens), int64(afterTokens), string(matchedFilter), 0)
	}

	if shellPipeJSON {
		result := envelope.RunResult{
			SchemaVersion:  "1",
			Command:        "oz shell pipe",
			Mode:           "compact",
			MatchedFilter:  string(matchedFilter),
			ExitCode:       0,
			DurationMs:     0,
			TokenEstBefore: beforeTokens,
			TokenEstAfter:  afterTokens,
			Stdout:         outStdout,
			Stderr:         outStderr,
			Warnings:       warnings,
		}
		result.TokenEstSaved = tokenSaved
		result.TokenReductionPct = reductionPct
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal pipe json: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	if outStdout != "" {
		fmt.Fprintln(cmd.OutOrStdout(), outStdout)
	}
	if outStderr != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), outStderr)
	}
	for _, warning := range warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning)
	}
	return nil
}

func resolvePipeFilterArgs(name, input string) ([]string, filter.ID, error) {
	trimmed := strings.TrimSpace(strings.ToLower(name))
	switch trimmed {
	case "", "auto":
		args, detected := autoDetectPipeFilter(input)
		return args, detected, nil
	case "cargo", "cargo-test":
		return []string{"cargo", "test"}, filter.FilterCargo, nil
	case "pytest":
		return []string{"pytest"}, filter.FilterPytest, nil
	case "go-test":
		return []string{"go", "test"}, filter.FilterGoTest, nil
	case "go-build":
		return []string{"go", "build"}, filter.FilterGoBuild, nil
	case "go-vet", "staticcheck":
		return []string{"go", "vet"}, filter.FilterGoVet, nil
	case "grep", "rg":
		return []string{"rg"}, filter.FilterRG, nil
	case "find", "fd":
		return []string{"find"}, filter.FilterFind, nil
	case "git-log":
		return []string{"git", "log"}, filter.FilterGitLog, nil
	case "git-diff":
		return []string{"git", "diff"}, filter.FilterGitDiff, nil
	case "git-status":
		return []string{"git", "status"}, filter.FilterGitStatus, nil
	case "git-blame":
		return []string{"git", "blame"}, filter.FilterGitBlame, nil
	case "git-show":
		return []string{"git", "show"}, filter.FilterGitShow, nil
	case "ls":
		return []string{"ls"}, filter.FilterLs, nil
	case "tree":
		return []string{"tree"}, filter.FilterTree, nil
	case "json":
		return []string{"jq"}, filter.FilterJSON, nil
	case "make":
		return []string{"make"}, filter.FilterMake, nil
	case "npm", "yarn", "pnpm":
		return []string{"npm"}, filter.FilterNpm, nil
	case "docker":
		return []string{"docker", "build"}, filter.FilterDocker, nil
	case "http", "curl", "wget":
		return []string{"curl"}, filter.FilterHTTP, nil
	case "env", "printenv":
		return []string{"env"}, filter.FilterEnv, nil
	case "wc":
		return []string{"wc"}, filter.FilterWc, nil
	case "diff", "patch":
		return []string{"diff"}, filter.FilterDiff, nil
	case "ps", "top":
		return []string{"ps"}, filter.FilterPs, nil
	case "df":
		return []string{"df"}, filter.FilterDf, nil
	default:
		return nil, filter.FilterNone, fmt.Errorf("unknown pipe filter %q", name)
	}
}

func autoDetectPipeFilter(input string) ([]string, filter.ID) {
	sample := input
	if len(sample) > 1024 {
		sample = sample[:1024]
	}
	lines := strings.Split(sample, "\n")
	nonEmpty := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			nonEmpty = append(nonEmpty, line)
		}
		if len(nonEmpty) >= 5 {
			break
		}
	}

	if strings.Contains(sample, "test result:") && strings.Contains(sample, "passed;") {
		return []string{"cargo", "test"}, filter.FilterCargo
	}
	if strings.Contains(sample, "=== test session starts") || strings.Contains(sample, "collected ") {
		return []string{"pytest"}, filter.FilterPytest
	}
	if strings.Contains(sample, "\"Action\"") {
		return []string{"go", "test"}, filter.FilterGoTest
	}
	if strings.Contains(sample, "diff --git ") {
		return []string{"git", "diff"}, filter.FilterGitDiff
	}
	for _, line := range nonEmpty {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) == 3 {
			return []string{"rg"}, filter.FilterRG
		}
	}
	pathLike := 0
	for _, line := range nonEmpty {
		if strings.Contains(line, "/") && !strings.Contains(line, ":") {
			pathLike++
		}
	}
	if len(nonEmpty) >= 3 && pathLike == len(nonEmpty) {
		return []string{"find"}, filter.FilterFind
	}
	if strings.HasPrefix(strings.TrimSpace(sample), "{") || strings.HasPrefix(strings.TrimSpace(sample), "[") {
		return []string{"jq"}, filter.FilterJSON
	}
	return []string{"unknown"}, filter.FilterGeneric
}

func insertShellTrackRow(command string, before, after int64, durationMs int64, matchedFilter string, missingCount int) error {
	exitCode := 0
	if missingCount > 0 {
		exitCode = 2
	}
	return insertShellTrackEnvelope(command, durationMs, before, after, matchedFilter, exitCode)
}

func insertShellTrackEnvelope(command string, durationMs int64, before, after int64, matchedFilter string, exitCode int) error {
	store, err := track.Open(track.DefaultPath())
	if err != nil {
		return err
	}
	defer func() {
		_ = store.Close()
	}()
	saved := before - after
	reduction := 0.0
	if before > 0 {
		reduction = (float64(saved) / float64(before)) * 100
	}
	return store.Insert(track.Run{
		Command:       command,
		Session:       activeObserveSession(),
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
	shellGainCmd.Flags().BoolVar(&shellGainInteractive, "interactive", false, "open interactive section explorer (huh)")
	shellReadCmd.Flags().IntVar(&shellReadMaxLines, "max-lines", 0, "show at most N lines")
	shellReadCmd.Flags().IntVar(&shellReadTailLines, "tail-lines", 0, "show last N lines")
	shellReadCmd.Flags().BoolVarP(&shellReadLineNumbers, "line-numbers", "n", false, "include line numbers")
	shellReadCmd.Flags().BoolVar(&shellReadNoFilter, "no-filter", false, "skip language-aware filtering")
	shellReadCmd.Flags().BoolVar(&shellReadJSON, "json", false, "emit JSON output")
	shellReadCmd.Flags().BoolVarP(&shellReadUltraCompact, "ultra-compact", "u", false, "maximum token reduction")
	shellReadCmd.Flags().BoolVar(&shellReadNoTrack, "no-track", false, "skip tracking DB write")
	shellPipeCmd.Flags().StringVar(&shellPipeFilter, "filter", "auto", "filter id or 'auto'")
	shellPipeCmd.Flags().BoolVar(&shellPipePassthrough, "passthrough", false, "relay stdin to stdout without filtering")
	shellPipeCmd.Flags().BoolVar(&shellPipeJSON, "json", false, "emit JSON envelope")
	shellPipeCmd.Flags().BoolVarP(&shellPipeUltraCompact, "ultra-compact", "u", false, "maximum token reduction")
	shellPipeCmd.Flags().BoolVar(&shellPipeNoTrack, "no-track", false, "skip tracking DB write")
	shellRewriteCmd.Flags().StringSliceVar(&shellRewriteExclude, "exclude", nil, "commands to bypass rewrite")
	shellCmd.AddCommand(shellRunCmd, shellGainCmd, shellReadCmd, shellPipeCmd, shellRewriteCmd)
}

func topByReduction(rows []gain.CommandStat, limit int) []gain.CommandStat {
	if len(rows) == 0 || limit <= 0 {
		return nil
	}
	sorted := append([]gain.CommandStat(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ReductionPctAvg == sorted[j].ReductionPctAvg {
			if sorted[i].InvocationCount == sorted[j].InvocationCount {
				return sorted[i].Command < sorted[j].Command
			}
			return sorted[i].InvocationCount > sorted[j].InvocationCount
		}
		return sorted[i].ReductionPctAvg > sorted[j].ReductionPctAvg
	})
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}
