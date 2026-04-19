package cmd

import (
	"encoding/json"
	"fmt"

	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/enrich"
	"github.com/oz-tools/oz/internal/query"
	"github.com/oz-tools/oz/internal/workspace"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage oz workspace context graph",
	Long: `Context commands discover the workspace by walking up from the current
directory until AGENTS.md and OZ.md are found, so they work from any subdirectory
inside an oz workspace.`,
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
	enrichModel       string
)

var contextQueryCmd = &cobra.Command{
	Use:   "query <text>",
	Short: "Query the context graph for agent routing",
	Args:  cobra.ExactArgs(1),
	RunE:  runContextQuery,
}

var contextEnrichCmd = &cobra.Command{
	Use:   "enrich",
	Short: "Enrich the context graph with LLM-extracted concepts (context/semantic.json)",
	Long: `Send the structural graph to an LLM via OpenRouter and write context/semantic.json
with extracted concept nodes and typed relationships.

Requires OPENROUTER_API_KEY to be set in the environment.

Existing reviewed items (reviewed: true) are preserved across re-enrichment runs.
Run 'oz context review' to review and accept extracted concepts.`,
	RunE: runContextEnrich,
}

func init() {
	contextCmd.AddCommand(contextBuildCmd)
	contextCmd.AddCommand(contextQueryCmd)
	contextCmd.AddCommand(contextEnrichCmd)
	contextQueryCmd.Flags().BoolVar(&queryRaw, "raw", false, "output routing debug JSON (scores + query-relevant subgraph) instead of routing packet")
	contextQueryCmd.Flags().BoolVar(&queryIncludeNotes, "include-notes", false, "include notes/ in context blocks")
	contextEnrichCmd.Flags().StringVar(&enrichModel, "model", "", "OpenRouter model ID (default: anthropic/claude-haiku-4)")
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

	if queryRaw {
		debug := query.BuildRawQueryDebug(root, queryText, opts)
		return printJSON(cmd, debug)
	}
	return printJSON(cmd, query.RunWithOptions(root, queryText, opts))
}

func runContextEnrich(cmd *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	// Load (or build) the structural graph.
	g, loadErr := ozcontext.LoadGraph(root)
	if loadErr != nil {
		result, buildErr := ozcontext.Build(root)
		if buildErr != nil {
			return fmt.Errorf("build context graph: %w", buildErr)
		}
		if err := ozcontext.Serialize(root, result.Graph); err != nil {
			return fmt.Errorf("write graph: %w", err)
		}
		g = result.Graph
	}

	res, err := enrich.Run(root, g, enrich.Options{Model: enrichModel})
	if err != nil {
		return err
	}

	cmd.Printf("context/semantic.json written\n")
	cmd.Printf("  model:     %s\n", res.Model)
	cmd.Printf("  concepts:  %d extracted\n", res.ConceptsAdded)
	cmd.Printf("  edges:     %d added\n", res.EdgesAdded)
	if res.Cost > 0 {
		cmd.Printf("  cost:      $%.4f\n", res.Cost)
	}
	if len(res.Skipped) > 0 {
		cmd.Printf("  skipped:   %d items\n", len(res.Skipped))
		for _, s := range res.Skipped {
			cmd.Printf("    - %s\n", s)
		}
	}
	return nil
}

func printJSON(cmd *cobra.Command, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(data))
	return nil
}

// findWorkspaceRoot returns the workspace root by walking up from the
// current working directory (same discovery rules as workspace.New with path ".").
func findWorkspaceRoot() (string, error) {
	ws, err := workspace.New(".")
	if err != nil {
		return "", fmt.Errorf("loading workspace: %w", err)
	}
	if !ws.Valid() {
		return "", fmt.Errorf("%s is not an oz workspace (missing AGENTS.md or OZ.md)", ws.Root)
	}
	return ws.Root, nil
}
