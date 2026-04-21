package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/enrich"
	"github.com/joaoajmatos/oz/internal/mcp"
	"github.com/joaoajmatos/oz/internal/query"
	"github.com/joaoajmatos/oz/internal/review"
	"github.com/joaoajmatos/oz/internal/semantic"
	"github.com/joaoajmatos/oz/internal/workspace"
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
	reviewAcceptAll   bool
	quietBuild        bool
)

var contextQueryCmd = &cobra.Command{
	Use:   "query [text]",
	Short: "Query the context graph for agent routing",
	Args:  cobra.MaximumNArgs(1),
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

var contextReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review unreviewed concepts and edges in context/semantic.json",
	Long: `Present new and changed concepts and edges from context/semantic.json in a
human-readable table, then prompt to accept or reject each item.

Accepted items are marked reviewed: true. Rejected items are removed.
Use --accept-all to skip interactive prompts (suitable for CI).`,
	RunE: runContextReview,
}

var contextServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start an MCP stdio server exposing oz context tools",
	Long: `Start a Model Context Protocol (MCP) server on stdin/stdout.

The server exposes four tools:
  query_graph    — route a task to the best-matching agent (full routing packet)
  get_node       — retrieve a structural graph node by ID
  get_neighbors  — list nodes adjacent to a given node
  agent_for_task — shorthand routing: task → agent + confidence only

Wire it into Claude Code or Cursor with:
  {"mcpServers":{"oz":{"command":"oz","args":["context","serve"]}}}`,
	RunE: runContextServe,
}

func init() {
	contextCmd.AddCommand(contextBuildCmd)
	contextCmd.AddCommand(contextQueryCmd)
	contextCmd.AddCommand(contextEnrichCmd)
	contextCmd.AddCommand(contextReviewCmd)
	contextCmd.AddCommand(contextServeCmd)
	contextBuildCmd.Flags().BoolVarP(&quietBuild, "quiet", "q", false, "suppress output")
	contextQueryCmd.Flags().BoolVar(&queryRaw, "raw", false, "output routing debug JSON (scores + query-relevant subgraph) instead of routing packet")
	contextQueryCmd.Flags().BoolVar(&queryIncludeNotes, "include-notes", false, "include notes/ in context blocks")
	contextEnrichCmd.Flags().StringVar(&enrichModel, "model", "", "OpenRouter model ID (default: anthropic/claude-haiku-4)")
	contextReviewCmd.Flags().BoolVar(&reviewAcceptAll, "accept-all", false, "mark all unreviewed items as reviewed without prompting")
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

	if !quietBuild {
		cmd.Printf("context/graph.json written — %d nodes, %d edges (hash: %s)\n",
			result.NodeCount, result.EdgeCount, result.Graph.ContentHash[:12])
	}
	return nil
}

func runContextQuery(cmd *cobra.Command, args []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	// No query text: return workspace overview (all agents listed).
	if len(args) == 0 {
		return printJSON(cmd, query.ListAgents(root))
	}

	// S6-01: warn if semantic overlay is stale.
	if g, loadErr := ozcontext.LoadGraph(root); loadErr == nil {
		if o, semErr := semantic.Load(root); semErr == nil && semantic.IsStale(o, g.ContentHash) {
			fmt.Fprintln(os.Stderr, "warning: semantic overlay may be stale — run 'oz context enrich' to update")
		}
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

func runContextReview(_ *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	_, err = review.Run(root, review.Options{AcceptAll: reviewAcceptAll})
	return err
}

func runContextServe(_ *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	srv := mcp.New(root)
	return srv.Serve(os.Stdin)
}

func printJSON(_ *cobra.Command, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(data))
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
