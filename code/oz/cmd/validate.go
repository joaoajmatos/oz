package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/oz-tools/oz/internal/validate"
	"github.com/oz-tools/oz/internal/workspace"
)

var validateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Lint a workspace against the oz convention",
	Long: `Validate checks a workspace for required files, directories, and structure.

Exit code 0 = valid. Exit code 1 = invalid. Suitable for CI.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
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
		return nil
	}

	fmt.Fprintf(os.Stderr, "FAIL %s\n", ws.Root)
	os.Exit(1)
	return nil
}
