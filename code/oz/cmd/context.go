package cmd

import (
	"fmt"
	"os"

	"github.com/oz-tools/oz/internal/context"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage oz workspace context graph",
}

var contextBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the structural context graph (context/graph.json)",
	Long: `Traverse the workspace, parse all oz-convention files, extract cross-references,
and write a deterministic context/graph.json.

Running 'oz context build' twice with no changes produces byte-identical output.`,
	RunE: runContextBuild,
}

var contextQueryCmd = &cobra.Command{
	Use:   "query <text>",
	Short: "Query the context graph (stub — full engine in Sprint 3)",
	Args:  cobra.ExactArgs(1),
	RunE:  runContextQuery,
}

func init() {
	contextCmd.AddCommand(contextBuildCmd)
	contextCmd.AddCommand(contextQueryCmd)
}

func runContextBuild(cmd *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	result, err := context.Build(root)
	if err != nil {
		return fmt.Errorf("build context graph: %w", err)
	}

	if err := context.Serialize(root, result.Graph); err != nil {
		return fmt.Errorf("write graph: %w", err)
	}

	cmd.Printf("context/graph.json written — %d nodes, %d edges (hash: %s)\n",
		result.NodeCount, result.EdgeCount, result.Graph.ContentHash[:12])
	return nil
}

func runContextQuery(cmd *cobra.Command, args []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	// Verify graph exists.
	if _, err := context.LoadGraph(root); err != nil {
		return fmt.Errorf("context graph not found — run 'oz context build' first: %w", err)
	}

	cmd.Printf("oz context query: full engine ships in Sprint 3 (query: %q)\n", args[0])
	return nil
}

// findWorkspaceRoot returns the workspace root. Currently uses the working
// directory; later sprints will add upward-search logic.
func findWorkspaceRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return root, nil
}
