package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joaoajmatos/oz/internal/shell/gain"
	"github.com/joaoajmatos/oz/internal/shell/track"
	"github.com/joaoajmatos/oz/internal/termstyle"
	"github.com/spf13/cobra"
)

type shellGainObserveState struct {
	Session   string `json:"session"`
	StartedAt int64  `json:"started_at"`
}

var shellGainObserveCmd = &cobra.Command{
	Use:   "observe",
	Short: "Start or stop observing a specific shell session",
}

var shellGainObserveStartCmd = &cobra.Command{
	Use:          "start <session>",
	Short:        "Start observing tracked shell gain for a session",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runShellGainObserveStart,
}

var shellGainObserveStopCmd = &cobra.Command{
	Use:          "stop",
	Short:        "Stop observing and print collected shell gain data",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE:         runShellGainObserveStop,
}

func init() {
	shellGainObserveCmd.AddCommand(shellGainObserveStartCmd, shellGainObserveStopCmd)
	shellGainCmd.AddCommand(shellGainObserveCmd)
}

func observeStatePath() string {
	return filepath.Join(filepath.Dir(track.DefaultPath()), "shell-observe-state.json")
}

func readObserveState() (shellGainObserveState, bool, error) {
	path := observeStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return shellGainObserveState{}, false, nil
		}
		return shellGainObserveState{}, false, fmt.Errorf("read observe state: %w", err)
	}
	var st shellGainObserveState
	if err := json.Unmarshal(data, &st); err != nil {
		return shellGainObserveState{}, false, fmt.Errorf("parse observe state: %w", err)
	}
	if strings.TrimSpace(st.Session) == "" || st.StartedAt <= 0 {
		return shellGainObserveState{}, false, nil
	}
	return st, true, nil
}

func writeObserveState(st shellGainObserveState) error {
	path := observeStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create observe state dir: %w", err)
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal observe state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write observe state: %w", err)
	}
	return nil
}

func clearObserveState() error {
	path := observeStatePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear observe state: %w", err)
	}
	return nil
}

func runShellGainObserveStart(cmd *cobra.Command, args []string) error {
	session := strings.TrimSpace(args[0])
	if session == "" {
		return fmt.Errorf("session must not be empty")
	}
	if current, ok, err := readObserveState(); err != nil {
		return err
	} else if ok {
		return fmt.Errorf("observe already active for session %q", current.Session)
	}
	st := shellGainObserveState{Session: session, StartedAt: time.Now().Unix()}
	if err := writeObserveState(st); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "oz shell gain observe: started session %q\n", session)
	return nil
}

func runShellGainObserveStop(cmd *cobra.Command, _ []string) error {
	st, ok, err := readObserveState()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("observe is not active")
	}
	store, err := track.Open(track.DefaultPath())
	if err != nil {
		return fmt.Errorf("open tracking store: %w", err)
	}
	defer func() { _ = store.Close() }()
	runs, err := store.Query(track.QueryOpts{Since: st.StartedAt, Session: st.Session})
	if err != nil {
		return fmt.Errorf("query observed runs: %w", err)
	}
	report := gain.BuildDetailed(runs, 0, gain.PeriodDaily, time.Now())
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", termstyle.Brand.Render("oz shell gain observe stop"), termstyle.Subtle.Render("(session "+st.Session+")"))
	fmt.Fprintf(cmd.OutOrStdout(), "  invocations: %d\n", report.Summary.InvocationCount)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens before: %d\n", report.Summary.TokenBeforeTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens after: %d\n", report.Summary.TokenAfterTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  tokens saved: %d\n", report.Summary.TokenSavedTotal)
	fmt.Fprintf(cmd.OutOrStdout(), "  avg reduction: %.2f%%\n", report.Summary.ReductionPctAvg)
	if err := clearObserveState(); err != nil {
		return err
	}
	return nil
}

func activeObserveSession() string {
	st, ok, err := readObserveState()
	if err != nil || !ok {
		return ""
	}
	return st.Session
}
