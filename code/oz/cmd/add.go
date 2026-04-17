package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/oz-tools/oz/internal/scaffold"
	"github.com/oz-tools/oz/internal/workspace"
)

var addForceFlag bool

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add optional integrations to an existing oz workspace",
}

var addClaudeCmd = &cobra.Command{
	Use:   "claude [path]",
	Short: "Add CLAUDE.md for Claude Code native integration",
	Long: `Generate a CLAUDE.md file in an existing oz workspace.

CLAUDE.md is loaded automatically by Claude Code and imports the oz
read-chain so every session starts with full workspace context.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAddClaude,
}

func init() {
	addClaudeCmd.Flags().BoolVar(&addForceFlag, "force", false, "overwrite CLAUDE.md if it already exists")
	addCmd.AddCommand(addClaudeCmd)
}

func runAddClaude(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	ws, err := workspace.New(path)
	if err != nil {
		return fmt.Errorf("loading workspace: %w", err)
	}
	if !ws.Valid() {
		return fmt.Errorf("%s is not an oz workspace (missing AGENTS.md or OZ.md)", ws.Root)
	}

	dest := filepath.Join(ws.Root, "CLAUDE.md")
	if _, err := os.Stat(dest); err == nil && !addForceFlag {
		return fmt.Errorf("CLAUDE.md already exists — use --force to overwrite")
	}

	manifest, err := ws.ReadManifest()
	if err != nil {
		return fmt.Errorf("reading OZ.md: %w", err)
	}

	if err := scaffold.WriteCLAUDEMD(ws.Root, manifest.Name, manifest.Description); err != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", err)
	}

	fmt.Printf("Added CLAUDE.md to %s\n", ws.Root)
	return nil
}
