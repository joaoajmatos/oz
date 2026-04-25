package readfilter

import (
	"fmt"
	"strings"

	"github.com/joaoajmatos/oz/internal/shell/envelope"
)

func Run(opts Options) (Result, error) {
	if opts.MaxLines != nil && opts.TailLines != nil {
		return Result{}, fmt.Errorf("--max-lines and --tail-lines are mutually exclusive")
	}
	reader := Resolve(opts.Path)
	warnings := make([]string, 0)
	output := opts.Content
	if !opts.NoFilter {
		filtered, err := reader.Filter(opts.Content, opts)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("readfilter %s failed: %v; falling back to raw", reader.Name(), err))
		} else {
			output = filtered
		}
		if strings.TrimSpace(opts.Content) != "" && strings.TrimSpace(output) == "" {
			warnings = append(warnings, fmt.Sprintf("readfilter %s emptied non-empty input; falling back to raw", reader.Name()))
			output = opts.Content
		}
		if envelope.EstimateTokens(output) > envelope.EstimateTokens(opts.Content) {
			output = opts.Content
		}
	}

	output = applyMaxLines(output, opts.MaxLines)
	output = applyTailLines(output, opts.TailLines)
	if opts.LineNumbers {
		output = applyLineNumbers(output)
	}

	return Result{
		Path:           opts.Path,
		Language:       reader.Name(),
		Content:        output,
		Warnings:       warnings,
		TokenEstBefore: envelope.EstimateTokens(opts.Content),
		TokenEstAfter:  envelope.EstimateTokens(output),
	}, nil
}
