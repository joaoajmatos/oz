package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/validate"
	"github.com/joaoajmatos/oz/internal/workspace"
)

// errValidationFailed is returned when a workspace fails validation.
// main.go translates this to exit code 1.
var errValidationFailed = errors.New("validation failed")

var withContext bool

var validateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Lint a workspace against the oz convention",
	Long: `Validate checks a workspace for required files, directories, and structure.

With no argument, the current directory is used; the nearest ancestor containing
AGENTS.md and OZ.md is treated as the workspace root, so you can run this from
any subdirectory inside the workspace.

Exit code 0 = valid. Exit code 1 = invalid. Suitable for CI.`,
	Args:          cobra.MaximumNArgs(1),
	RunE:          runValidate,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	validateCmd.Flags().BoolVar(&withContext, "with-context", false, "run 'oz context build' after validation and report node/edge count")
}

func runValidate(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	ws, err := workspace.New(path)
	if err != nil {
		return fmt.Errorf("loading workspace: %w", err)
	}

	result := validate.Validate(ws)

	for _, f := range result.Findings {
		switch f.Severity {
		case validate.Error:
			fmt.Fprintf(os.Stderr, "  error   %s\n", f.Message)
		case validate.Warning:
			fmt.Fprintf(os.Stderr, "  warning %s\n", f.Message)
		}
	}

	if result.Valid() {
		fmt.Printf("ok  %s\n", ws.Root)
	} else {
		fmt.Fprintf(os.Stderr, "FAIL %s\n", ws.Root)
	}

	if withContext {
		br, buildErr := ozcontext.Build(ws.Root)
		if buildErr != nil {
			fmt.Fprintf(os.Stderr, "  warning context build failed: %v\n", buildErr)
		} else {
			fmt.Printf("context/graph.json: %d nodes, %d edges\n", br.NodeCount, br.EdgeCount)
		}
	}

	if !result.Valid() {
		return errValidationFailed
	}
	return nil
}
