package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/joaoajmatos/oz/internal/scaffold"
	"github.com/joaoajmatos/oz/internal/termstyle"
	"github.com/joaoajmatos/oz/internal/workspace"
)

var addForceFlag bool
var addPackageForce bool

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add integrations or optional packages to an existing oz workspace",
	Long: strings.TrimSpace(`
Add subcommands fall into two groups:

  • Integrations: ` + "`oz add claude`" + `, ` + "`oz add cursor`" + ` — IDE / editor hook wiring.
  • Optional packages: ` + "`oz add <package>`" + ` — bundled agent + skill trees shipped with the oz binary.
    V1 package IDs: ` + strings.Join(scaffold.ValidPackageIDs(), ", ") + `.

Use ` + "`oz add list`" + ` (alias: ` + "`oz add ls`" + `) for a formatted table of integrations and packages.

With no path argument, the current directory is used; the nearest ancestor containing
AGENTS.md and OZ.md is treated as the workspace root.`),
}

var addClaudeCmd = &cobra.Command{
	Use:   "claude [path]",
	Short: "Add Claude Code integration (CLAUDE.md + hooks)",
	Long: `Add Claude Code integration to an existing oz workspace.

Writes CLAUDE.md (loaded automatically by Claude Code) and installs Claude Code
hook configuration (.claude/settings.json) plus the shared hook scripts under
.oz/hooks/. Installs oz skills in both workspace skills/ and ~/.cursor/skills-cursor/.
Hooks enforce oz convention on every edit and gate git commits.

With no argument, the current directory is used; the nearest ancestor containing
AGENTS.md and OZ.md is treated as the workspace root.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAddClaude,
}

var addCursorCmd = &cobra.Command{
	Use:   "cursor [path]",
	Short: "Add Cursor integration (hooks)",
	Long: `Add Cursor integration to an existing oz workspace.

Writes .cursor/hooks.json, installs the shared hook scripts under .oz/hooks/, and
installs oz skills in both workspace skills/ and ~/.cursor/skills-cursor/.
Hooks enforce oz convention on every edit and gate git commits.

With no argument, the current directory is used; the nearest ancestor containing
AGENTS.md and OZ.md is treated as the workspace root.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAddCursor,
}

func init() {
	addCmd.AddCommand(addListCmd)

	addClaudeCmd.Flags().BoolVar(&addForceFlag, "force", false, "overwrite CLAUDE.md if it already exists")
	addCmd.AddCommand(addClaudeCmd)
	addCmd.AddCommand(addCursorCmd)

	for _, id := range scaffold.ValidPackageIDs() {
		id := id
		c := &cobra.Command{
			Use:   id + " [path]",
			Short: fmt.Sprintf("Add optional package %q (agent + skills)", id),
			Long: fmt.Sprintf(`Install optional package %q into an existing oz workspace.

Writes package files from templates embedded in the oz binary. Refuses to overwrite
existing package files unless --force is set.

With no path argument, the current directory is used; the nearest ancestor containing
AGENTS.md and OZ.md is treated as the workspace root.`, id),
			Args: cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runAddPackage(cmd, args, id)
			},
		}
		c.Flags().BoolVar(&addPackageForce, "force", false, "overwrite package files if they already exist")
		addCmd.AddCommand(c)
	}
}

func runAddPackage(cmd *cobra.Command, args []string, id string) error {
	root, err := resolveWorkspaceRoot(args)
	if err != nil {
		return err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	paths, err := scaffold.InstallPackage(id, root, force)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%s %s %s\n", termstyle.OK.Render("✓"), termstyle.Subtle.Render("Added package"), termstyle.Command.Render(id))
	fmt.Fprintf(out, "  %s %s\n", termstyle.Subtle.Render("workspace"), root)
	for _, p := range paths {
		fmt.Fprintf(out, "  %s\n", p)
	}
	return nil
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
	if err := scaffold.WriteCursorSkills(root); err != nil {
		return fmt.Errorf("writing Cursor skills: %w", err)
	}

	fmt.Printf("%s %s\n", termstyle.OK.Render("✓"), termstyle.Subtle.Render("Added Claude Code integration"))
	fmt.Printf("  %s %s\n", termstyle.Subtle.Render("workspace"), root)
	fmt.Println("  CLAUDE.md")
	fmt.Println("  .claude/settings.json")
	fmt.Println("  .oz/hooks/oz-session-init.sh")
	fmt.Println("  .oz/hooks/oz-after-edit.sh")
	fmt.Println("  .oz/hooks/oz-pre-commit.sh")
	fmt.Println("  .oz/hooks/oz-shell-rewrite-claude.sh")
	fmt.Println("  .oz/hooks/oz-shell-rewrite-cursor.sh")
	fmt.Println("  .oz/hooks/oz-read-rewrite-cursor.sh")
	fmt.Println("  .oz/hooks/oz-read-policy-cursor.sh")
	fmt.Println("  .oz/hooks/oz-shell-rewrite.sh")
	fmt.Println("  skills/oz/SKILL.md")
	fmt.Println("  skills/oz/references/audit-and-validate.md")
	fmt.Println("  skills/oz/references/context-and-mcp.md")
	fmt.Println("  skills/oz-shell/SKILL.md")
	fmt.Println("  ~/.cursor/skills-cursor/oz/SKILL.md")
	fmt.Println("  ~/.cursor/skills-cursor/oz/references/audit-and-validate.md")
	fmt.Println("  ~/.cursor/skills-cursor/oz/references/context-and-mcp.md")
	fmt.Println("  ~/.cursor/skills-cursor/oz-shell/SKILL.md")
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
	if err := scaffold.WriteCursorSkills(root); err != nil {
		return fmt.Errorf("writing Cursor skills: %w", err)
	}

	fmt.Printf("%s %s\n", termstyle.OK.Render("✓"), termstyle.Subtle.Render("Added Cursor integration"))
	fmt.Printf("  %s %s\n", termstyle.Subtle.Render("workspace"), root)
	fmt.Println("  .cursor/hooks.json")
	fmt.Println("  .oz/hooks/oz-session-init.sh")
	fmt.Println("  .oz/hooks/oz-after-edit.sh")
	fmt.Println("  .oz/hooks/oz-pre-commit.sh")
	fmt.Println("  .oz/hooks/oz-shell-rewrite-cursor.sh")
	fmt.Println("  .oz/hooks/oz-read-rewrite-cursor.sh")
	fmt.Println("  .oz/hooks/oz-read-policy-cursor.sh")
	fmt.Println("  .oz/hooks/oz-shell-rewrite-claude.sh")
	fmt.Println("  .oz/hooks/oz-shell-rewrite.sh")
	fmt.Println("  skills/oz/SKILL.md")
	fmt.Println("  skills/oz/references/audit-and-validate.md")
	fmt.Println("  skills/oz/references/context-and-mcp.md")
	fmt.Println("  skills/oz-shell/SKILL.md")
	fmt.Println("  ~/.cursor/skills-cursor/oz/SKILL.md")
	fmt.Println("  ~/.cursor/skills-cursor/oz/references/audit-and-validate.md")
	fmt.Println("  ~/.cursor/skills-cursor/oz/references/context-and-mcp.md")
	fmt.Println("  ~/.cursor/skills-cursor/oz-shell/SKILL.md")
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
