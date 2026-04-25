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
	shellGainDays                    int
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
	runs, err := store.QuerySinceDays(shellGainDays, now)
	if err != nil {
		return fmt.Errorf("query gain runs: %w", err)
	}
	report := gain.Aggregate(runs, shellGainDays, now)

	if shellGainJSON {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal gain json: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	if report.Empty() {
		fmt.Fprintf(cmd.OutOrStdout(), "oz shell gain: no tracked runs in the last %d days\n", shellGainDays)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "oz shell gain (%d days)\n", report.RetentionDays)
	fmt.Fprintf(cmd.OutOrStdout(), "  invocations: %d\n", report.InvocationCount)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens before: %d\n", report.TokenBeforeTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens after: %d\n", report.TokenAfterTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens saved: %d\n", report.TokenSavedTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  avg reduction: %.2f%%\n", report.ReductionPctAvg)
	fmt.Fprintf(cmd.OutOrStdout(), "  avg duration: %.2fms\n", report.DurationMsAvg)
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
	shellCmd.AddCommand(shellRunCmd, shellGainCmd)
}
