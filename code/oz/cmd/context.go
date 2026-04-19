package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/query"
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

var (
	queryRaw          bool
	queryIncludeNotes bool
)

var contextQueryCmd = &cobra.Command{
	Use:   "query <text>",
	Short: "Query the context graph for agent routing",
	Args:  cobra.ExactArgs(1),
	RunE:  runContextQuery,
}

func init() {
	contextCmd.AddCommand(contextBuildCmd)
	contextCmd.AddCommand(contextQueryCmd)
	contextQueryCmd.Flags().BoolVar(&queryRaw, "raw", false, "output full subgraph JSON instead of routing packet")
	contextQueryCmd.Flags().BoolVar(&queryIncludeNotes, "include-notes", false, "include notes/ in context blocks")
}

func runContextBuild(cmd *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	result, err := ozcontext.Build(root)
	if err != nil {
		return fmt.Errorf("build context graph: %w", err)
	}

	if err := ozcontext.Serialize(root, result.Graph); err != nil {
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

	queryText := args[0]
	opts := query.Options{
		IncludeNotes: queryIncludeNotes,
		RawMode:      queryRaw,
	}

	result := query.RunWithOptions(root, queryText, opts)

	if queryRaw {
		// Raw mode: emit full graph for debugging.
		g, loadErr := ozcontext.LoadGraph(root)
		if loadErr != nil {
			return fmt.Errorf("load graph for raw output: %w", loadErr)
		}
		raw := struct {
			Query  string       `json:"query"`
			Result query.Result `json:"result"`
			Graph  interface{}  `json:"graph"`
		}{queryText, result, g}
		return printJSON(cmd, raw)
	}

	return printJSON(cmd, result)
}

func printJSON(cmd *cobra.Command, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(data))
	return nil
}

// findWorkspaceRoot returns the workspace root from the working directory.
func findWorkspaceRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return root, nil
}
