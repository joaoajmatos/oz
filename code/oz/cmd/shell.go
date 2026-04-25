package cmd

import (
	"fmt"

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
	Use:   "run -- <cmd...>",
	Short: "Run a shell command and optionally compress its output",
	Args:  cobra.ArbitraryArgs,
	RunE:  runShellRun,
}

var shellGainCmd = &cobra.Command{
	Use:   "gain",
	Short: "Show cumulative token savings from tracked shell runs",
	RunE:  runShellGain,
}

func runShellRun(cmd *cobra.Command, _ []string) error {
	fmt.Fprintln(cmd.OutOrStdout(), "oz shell run: not yet implemented")
	return nil
}

func runShellGain(cmd *cobra.Command, _ []string) error {
	fmt.Fprintln(cmd.OutOrStdout(), "oz shell gain: not yet implemented")
	return nil
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
