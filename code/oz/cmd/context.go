package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/enrich"
	"github.com/joaoajmatos/oz/internal/mcp"
	"github.com/joaoajmatos/oz/internal/query"
	"github.com/joaoajmatos/oz/internal/review"
	"github.com/joaoajmatos/oz/internal/semantic"
	"github.com/joaoajmatos/oz/internal/termstyle"
	"github.com/joaoajmatos/oz/internal/workspace"
	"github.com/mattn/go-isatty"
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
	enrichModel       string
	enrichQuiet       bool
	enrichForce       bool
	reviewAcceptAll   bool
	reviewNoColor     bool
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
	RunE:         runContextServe,
	SilenceUsage: true,
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
	contextEnrichCmd.Flags().StringVar(&enrichModel, "model", "", "OpenRouter model ID (default: openrouter/free)")
	contextEnrichCmd.Flags().BoolVarP(&enrichQuiet, "quiet", "q", false, "suppress progress and summary output")
	contextEnrichCmd.Flags().BoolVar(&enrichForce, "force", false, "force enrichment even when semantic overlay is already fresh")
	contextReviewCmd.Flags().BoolVar(&reviewAcceptAll, "accept-all", false, "mark all unreviewed items as reviewed without prompting")
	contextReviewCmd.Flags().BoolVar(&reviewNoColor, "no-color", false, "force plain text (no ANSI); also when stdout is not a TTY or NO_COLOR is set")
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
	showProgress := !enrichQuiet
	var (
		stageMu      sync.RWMutex
		currentStage = "starting enrichment"
		requestedAt  time.Time
		waitingHints = []string{
			"dialing OpenRouter",
			"model is thinking",
			"assembling concepts",
			"validating output shape",
			"building semantic candidates",
			"scoring confidence tags",
			"cross-checking graph nodes",
			"mapping concept edges",
			"normalizing identifiers",
			"checking schema constraints",
			"deduplicating concept names",
			"resolving agent ownership",
			"linking specs to concepts",
			"verifying edge directions",
			"cleaning low-signal relations",
			"preparing merge payload",
		}
		stopSpinner func()
	)
	if showProgress {
		stopCh := make(chan struct{})
		doneCh := make(chan struct{})
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		stopSpinner = func() {
			close(stopCh)
			<-doneCh
		}
		go func() {
			defer close(doneCh)
			ticker := time.NewTicker(120 * time.Millisecond)
			defer ticker.Stop()
			i := 0
			for {
				select {
				case <-ticker.C:
					stageMu.RLock()
					stage := currentStage
					waitStarted := requestedAt
					stageMu.RUnlock()
					waitHint := ""
					if !waitStarted.IsZero() {
						elapsed := int(time.Since(waitStarted).Seconds())
						waitHint = waitingHints[(elapsed/3)%len(waitingHints)]
					}
					stageText := termstyle.Muted.Render(stage)
					if waitHint != "" {
						stageText = stageText + " " + termstyle.Subtle.Render("("+waitHint+")")
					}
					fmt.Fprintf(cmd.ErrOrStderr(), "\r\033[2K%s %s",
						termstyle.AccentBold.Render(frames[i%len(frames)]),
						stageText,
					)
					i++
				case <-stopCh:
					fmt.Fprintf(cmd.ErrOrStderr(), "\r\033[2K")
					return
				}
			}
		}()
		defer stopSpinner()
	}

	progress := func(format string, args ...interface{}) {
		if !showProgress {
			return
		}
		nextStage := fmt.Sprintf(format, args...)
		stageMu.Lock()
		changed := currentStage != nextStage
		currentStage = nextStage
		if nextStage == "requesting model response" {
			requestedAt = time.Now()
		} else {
			requestedAt = time.Time{}
		}
		stageMu.Unlock()
		if changed {
			fmt.Fprintf(cmd.ErrOrStderr(), "\r\033[2K%s %s\n",
				termstyle.Subtle.Render("•"),
				termstyle.Subtle.Render(nextStage),
			)
		}
	}

	// Load (or build) the structural graph.
	progress("loading context graph")
	g, loadErr := ozcontext.LoadGraph(root)
	if loadErr != nil {
		progress("context graph not found; building it first")
		result, buildErr := ozcontext.Build(root)
		if buildErr != nil {
			return fmt.Errorf("build context graph: %w", buildErr)
		}
		if err := ozcontext.Serialize(root, result.Graph); err != nil {
			return fmt.Errorf("write graph: %w", err)
		}
		progress("context/graph.json written — %d nodes, %d edges", result.NodeCount, result.EdgeCount)
		g = result.Graph
	}
	existingOverlay, err := semantic.Load(root)
	if err != nil {
		return fmt.Errorf("load existing overlay: %w", err)
	}
	if shouldSkipEnrich(existingOverlay, g.ContentHash, enrichModel, enrichForce) {
		if showProgress {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s %s\n",
				termstyle.OK.Render("✓"),
				termstyle.Muted.Render("semantic overlay is already up to date; skipping enrichment"),
			)
			cmd.Printf("%s\n", termstyle.Brand.Render("context enrich skipped"))
			cmd.Printf("  %s %s\n", termstyle.Subtle.Render("reason   "), termstyle.AccentBold.Render("semantic overlay is fresh"))
			cmd.Printf("  %s %s\n", termstyle.Subtle.Render("hint     "), termstyle.AccentBold.Render("use --force to re-enrich anyway"))
		}
		return nil
	}

	// Pre-flight: warn when a prior enrichment exists but produced a sparse concept
	// graph — implements edges will have little to map to. Does not block fresh
	// workspaces (existingOverlay.Concepts == nil). Use --force to suppress.
	const minConceptsForCodeMapping = 10
	if !enrichForce && existingOverlay != nil && len(existingOverlay.Concepts) > 0 &&
		len(existingOverlay.Concepts) < minConceptsForCodeMapping {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"warning: concept graph is sparse (%d concepts) — implements edges may not map well.\n"+
				"  Run 'oz context enrich' on specs/docs first, or use --force to skip this check.\n",
			len(existingOverlay.Concepts),
		)
	}

	res, err := enrich.Run(root, g, enrich.Options{
		Model: enrichModel,
		Progress: func(stage string) {
			progress("%s", stage)
		},
	})
	if err != nil {
		return err
	}

	if enrichQuiet {
		return nil
	}
	if showProgress {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s %s\n",
			termstyle.OK.Render("✓"),
			termstyle.Muted.Render("enrichment completed"),
		)
	}

	cmd.Printf("%s\n", termstyle.Brand.Render("context enrich complete"))
	cmd.Printf("  %s %s\n", termstyle.Subtle.Render("output   "), termstyle.AccentBold.Render("context/semantic.json"))
	cmd.Printf("  %s %s\n", termstyle.Subtle.Render("model    "), termstyle.AccentBold.Render(res.Model))
	cmd.Printf("  %s %s\n", termstyle.Subtle.Render("concepts "), termstyle.AccentBold.Render(fmt.Sprintf("%d extracted", res.ConceptsAdded)))
	cmd.Printf("  %s %s\n", termstyle.Subtle.Render("edges    "), termstyle.AccentBold.Render(fmt.Sprintf("%d added", res.EdgesAdded)))
	if res.Cost > 0 {
		cmd.Printf("  %s %s\n", termstyle.Subtle.Render("cost     "), termstyle.AccentBold.Render(fmt.Sprintf("$%.4f", res.Cost)))
	}
	if len(res.Skipped) > 0 {
		cmd.Printf("  %s %s\n", termstyle.Subtle.Render("skipped  "), termstyle.AccentBold.Render(fmt.Sprintf("%d items", len(res.Skipped))))
		for _, s := range res.Skipped {
			cmd.Printf("    - %s\n", termstyle.Subtle.Render(s))
		}
	}
	return nil
}

func runContextReview(_ *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	_, err = review.Run(root, review.Options{
		AcceptAll: reviewAcceptAll,
		NoColor:   reviewNoColor,
	})
	return err
}

func runContextServe(cmd *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	printContextServeBanner(cmd.ErrOrStderr(), root)
	srv := mcp.New(root)
	return srv.Serve(os.Stdin)
}

// printContextServeBanner explains that the process is an MCP stdio server and
// how to stop. It must use stderr only — stdout is reserved for JSON-RPC.
func printContextServeBanner(w io.Writer, root string) {
	if shouldUseContextServeTTY(w) {
		fmt.Fprintf(w, "%s\n", termstyle.Brand.Render("context serve — MCP stdio server"))
		fmt.Fprintf(w, "  %s %s\n", termstyle.Subtle.Render("workspace"), termstyle.AccentBold.Render(root))
		fmt.Fprintf(w, "  %s %s\n", termstyle.Subtle.Render("protocol "), termstyle.Muted.Render("JSON-RPC on stdin/stdout"))
		fmt.Fprintf(w, "  %s %s\n", termstyle.Subtle.Render("stop     "), termstyle.AccentBold.Render("Ctrl+C, or close stdin (EOF)"))
		return
	}
	fmt.Fprintln(w, "context serve — MCP stdio server")
	fmt.Fprintf(w, "  workspace: %s\n", root)
	fmt.Fprintln(w, "  protocol: JSON-RPC on stdin/stdout")
	fmt.Fprintln(w, "  stop: Ctrl+C, or close stdin (EOF)")
}

func shouldUseContextServeTTY(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
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

func shouldSkipEnrich(existing *semantic.Overlay, graphHash, requestedModel string, force bool) bool {
	if force || existing == nil || semantic.IsStale(existing, graphHash) {
		return false
	}
	if requestedModel != "" && existing.Model != requestedModel {
		return false
	}
	return true
}
