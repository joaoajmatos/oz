package exec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"time"
)

type Result struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
}

// Run executes command args exactly once and captures stdout/stderr/exit code.
func Run(args []string) (Result, error) {
	if len(args) == 0 {
		return Result{}, fmt.Errorf("no command provided")
	}

	cmd := osexec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	durationMs := time.Since(start).Milliseconds()

	result := Result{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   0,
		DurationMs: durationMs,
	}

	if err == nil {
		return result, nil
	}

	var exitErr *osexec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	return result, fmt.Errorf("start command: %w", err)
}
