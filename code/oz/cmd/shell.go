package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	shellrun "github.com/joaoajmatos/oz/internal/shell/run"
	"github.com/spf13/cobra"
)

var (
	shellMode, shellTee              string
	shellJSON, shellNoTrack          bool
	shellUltraCompact, shellGainJSON bool
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
	fmt.Fprintln(cmd.OutOrStdout(), "oz shell gain: not yet implemented")
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
	shellCmd.AddCommand(shellRunCmd, shellGainCmd)
}
