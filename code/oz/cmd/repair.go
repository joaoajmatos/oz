package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/joaoajmatos/oz/internal/convention"
	"github.com/joaoajmatos/oz/internal/scaffold"
	"github.com/joaoajmatos/oz/internal/workspace"
)

var repairCmd = &cobra.Command{
	Use:   "repair [path]",
	Short: "Restore missing default workspace files",
	Long: `Repair checks an existing oz workspace for missing default files and recreates them.
Existing files are never overwritten.

With no argument, the current directory is used; the nearest ancestor containing
AGENTS.md and OZ.md is treated as the workspace root, so you can run this from
any subdirectory inside the workspace.`,
	Args:          cobra.MaximumNArgs(1),
	RunE:          runRepair,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func runRepair(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	ws, err := workspace.New(path)
	if err != nil {
		return fmt.Errorf("loading workspace: %w", err)
	}

	if !ws.Valid() {
		return fmt.Errorf("not an oz workspace (missing AGENTS.md or OZ.md). Run `oz init` to create one")
	}

	ozPath := filepath.Join(ws.Root, "OZ.md")
	if _, err := os.Stat(ozPath); err != nil {
		return fmt.Errorf("OZ.md not found at %s — run `oz init` to scaffold a new workspace", ozPath)
	}

	manifest, err := ws.ReadManifest()
	if err != nil {
		return fmt.Errorf("reading OZ.md: %w", err)
	}

	name := manifest.Name
	if name == "" {
		name = filepath.Base(ws.Root)
	}

	agentNames, err := ws.Agents()
	if err != nil || len(agentNames) == 0 {
		agentNames = convention.DefaultAgents
	}

	var agents []scaffold.AgentConfig
	for _, n := range agentNames {
		t := ""
		if n == "coding" {
			t = "coding"
		}
		agents = append(agents, scaffold.AgentConfig{Name: n, Type: t})
	}

	cfg := scaffold.Config{
		Name:        name,
		Description: manifest.Description,
		CodeMode:    "inline",
		Agents:      agents,
		// CLAUDE.md is opt-in only (oz init --claude / oz add claude); never repaired here.
		ClaudeMD: false,
	}

	fmt.Printf("%s %s %s\n\n", styleSubtle.Render("Repairing workspace at"), styleCmd.Render(ws.Root), styleSubtle.Render("..."))

	result, err := scaffold.Repair(ws.Root, cfg)
	if err != nil {
		return fmt.Errorf("repair: %w", err)
	}

	if len(result.Created) == 0 {
		fmt.Println(styleSuccess.Render("ok  all default files present"))
		return nil
	}

	fmt.Printf("%s %d file(s) restored:\n", styleSuccess.Render("✓"), len(result.Created))
	for _, p := range result.Created {
		fmt.Printf("  %s  %s\n", styleSubtle.Render("created"), p)
	}
	return nil
}
