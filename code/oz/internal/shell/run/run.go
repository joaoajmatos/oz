package run

import (
	"fmt"
	"strings"
	"time"

	"github.com/joaoajmatos/oz/internal/shell/envelope"
	ozexec "github.com/joaoajmatos/oz/internal/shell/exec"
	"github.com/joaoajmatos/oz/internal/shell/filter"
	"github.com/joaoajmatos/oz/internal/shell/tee"
	"github.com/joaoajmatos/oz/internal/shell/track"
)

func Execute(args []string, opts Options) (Result, error) {
	if len(args) == 0 {
		return Result{}, fmt.Errorf("missing command after --")
	}

	execResult, err := ozexec.Run(args)
	if err != nil {
		return Result{}, err
	}

	command := strings.Join(args, " ")
	warnings := make([]string, 0)
	outStdout := execResult.Stdout
	outStderr := execResult.Stderr
	matchedFilter := "none"
	mode := opts.Mode
	if mode == "" {
		mode = "compact"
	}

	if mode == "compact" {
		compactStdout, compactStderr, detectedFilter, compactErr := filter.Apply(args, execResult.Stdout, execResult.Stderr, execResult.ExitCode, opts.UltraCompact)
		matchedFilter = detectedFilter
		if compactErr != nil {
			warnings = append(warnings, fmt.Sprintf("%s failed; falling back to raw output", detectedFilter))
			fallbackStdout, fallbackStderr, _, fallbackErr := filter.Apply([]string{"unknown"}, execResult.Stdout, execResult.Stderr, execResult.ExitCode, opts.UltraCompact)
			if fallbackErr == nil {
				outStdout = fallbackStdout
				outStderr = fallbackStderr
				matchedFilter = filter.FilterGeneric
			} else {
				warnings = append(warnings, fmt.Sprintf("generic fallback failed: %v", fallbackErr))
			}
		} else {
			outStdout = compactStdout
			outStderr = compactStderr
		}
	}

	var rawOutputRef *string
	if shouldTee(opts.TeeMode, execResult.ExitCode) {
		ref, teeErr := tee.Write(command, execResult.Stdout, execResult.Stderr)
		if teeErr != nil {
			warnings = append(warnings, fmt.Sprintf("tee write failed: %v", teeErr))
		} else {
			rawOutputRef = ref
		}
	}

	runEnvelope := envelope.RunResult{
		SchemaVersion:  "1",
		Command:        command,
		Mode:           mode,
		MatchedFilter:  matchedFilter,
		ExitCode:       execResult.ExitCode,
		DurationMs:     execResult.DurationMs,
		TokenEstBefore: envelope.EstimateTokens(execResult.Stdout + execResult.Stderr),
		TokenEstAfter:  envelope.EstimateTokens(outStdout + outStderr),
		Stdout:         outStdout,
		Stderr:         outStderr,
		Warnings:       warnings,
		RawOutputRef:   rawOutputRef,
	}
	runEnvelope.TokenEstSaved = runEnvelope.TokenEstBefore - runEnvelope.TokenEstAfter
	if runEnvelope.TokenEstBefore > 0 {
		runEnvelope.TokenReductionPct = (float64(runEnvelope.TokenEstSaved) / float64(runEnvelope.TokenEstBefore)) * 100
	}

	if !opts.NoTrack {
		if err := insertTracking(runEnvelope); err != nil {
			runEnvelope.Warnings = append(runEnvelope.Warnings, fmt.Sprintf("tracking unavailable: %v", err))
		}
	}

	return Result{
		Envelope: runEnvelope,
		ExitCode: execResult.ExitCode,
	}, nil
}

func shouldTee(mode string, exitCode int) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	case "failures":
		return exitCode != 0
	default:
		return exitCode != 0
	}
}

func insertTracking(result envelope.RunResult) error {
	store, err := track.Open(track.DefaultPath())
	if err != nil {
		return err
	}
	defer func() {
		_ = store.Close()
	}()

	return store.Insert(track.Run{
		Command:       result.Command,
		RecordedAt:    time.Now().Unix(),
		DurationMs:    result.DurationMs,
		TokenBefore:   int64(result.TokenEstBefore),
		TokenAfter:    int64(result.TokenEstAfter),
		TokenSaved:    int64(result.TokenEstSaved),
		ReductionPct:  result.TokenReductionPct,
		MatchedFilter: result.MatchedFilter,
		ExitCode:      result.ExitCode,
	})
}
