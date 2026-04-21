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
	Short: "Add integrations to an existing oz workspace",
}

var addClaudeCmd = &cobra.Command{
	Use:   "claude [path]",
	Short: "Add Claude Code integration (CLAUDE.md + hooks)",
	Long: `Add Claude Code integration to an existing oz workspace.

Writes CLAUDE.md (loaded automatically by Claude Code) and installs Claude Code
hook configuration (.claude/settings.json) plus the shared hook scripts under
.cursor/hooks/. Hooks enforce oz convention on every edit and gate git commits.

With no argument, the current directory is used; the nearest ancestor containing
AGENTS.md and OZ.md is treated as the workspace root.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAddClaude,
}

var addCursorCmd = &cobra.Command{
	Use:   "cursor [path]",
	Short: "Add Cursor integration (hooks)",
	Long: `Add Cursor integration to an existing oz workspace.

Writes .cursor/hooks.json and installs the shared hook scripts under .cursor/hooks/.
Hooks enforce oz convention on every edit and gate git commits.

With no argument, the current directory is used; the nearest ancestor containing
AGENTS.md and OZ.md is treated as the workspace root.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAddCursor,
}

func init() {
	addClaudeCmd.Flags().BoolVar(&addForceFlag, "force", false, "overwrite CLAUDE.md if it already exists")
	addCmd.AddCommand(addClaudeCmd)
	addCmd.AddCommand(addCursorCmd)
}

func runAddClaude(_ *cobra.Command, args []string) error {
	root, err := resolveWorkspaceRoot(args)
	if err != nil {
		return err
	}

	dest := filepath.Join(root, "CLAUDE.md")
	if _, err := os.Stat(dest); err == nil && !addForceFlag {
		return fmt.Errorf("CLAUDE.md already exists — use --force to overwrite")
	}

	ws, err := workspace.New(root)
	if err != nil {
		return fmt.Errorf("loading workspace: %w", err)
	}
	manifest, err := ws.ReadManifest()
	if err != nil {
		return fmt.Errorf("reading OZ.md: %w", err)
	}

	if err := scaffold.WriteCLAUDEMD(root, manifest.Name, manifest.Description); err != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", err)
	}
	if err := scaffold.WriteClaudeHooks(root); err != nil {
		return fmt.Errorf("writing Claude Code hooks: %w", err)
	}

	fmt.Printf("Added Claude Code integration to %s\n", root)
	fmt.Println("  CLAUDE.md")
	fmt.Println("  .claude/settings.json")
	fmt.Println("  .cursor/hooks/oz-session-init.sh")
	fmt.Println("  .cursor/hooks/oz-after-edit.sh")
	fmt.Println("  .cursor/hooks/oz-pre-commit.sh")
	return nil
}

func runAddCursor(_ *cobra.Command, args []string) error {
	root, err := resolveWorkspaceRoot(args)
	if err != nil {
		return err
	}

	if err := scaffold.WriteCursorHooks(root); err != nil {
		return fmt.Errorf("writing Cursor hooks: %w", err)
	}

	fmt.Printf("Added Cursor integration to %s\n", root)
	fmt.Println("  .cursor/hooks.json")
	fmt.Println("  .cursor/hooks/oz-session-init.sh")
	fmt.Println("  .cursor/hooks/oz-after-edit.sh")
	fmt.Println("  .cursor/hooks/oz-pre-commit.sh")
	return nil
}

func resolveWorkspaceRoot(args []string) (string, error) {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	ws, err := workspace.New(path)
	if err != nil {
		return "", fmt.Errorf("loading workspace: %w", err)
	}
	if !ws.Valid() {
		return "", fmt.Errorf("%s is not an oz workspace (missing AGENTS.md or OZ.md)", ws.Root)
	}
	return ws.Root, nil
}
