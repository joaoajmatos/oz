package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/enrich"
	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/query"
	"github.com/joaoajmatos/oz/internal/semantic"
	"github.com/joaoajmatos/oz/internal/termstyle"
	"github.com/spf13/cobra"
)

var (
	conceptAddName        string
	conceptAddSeed        string
	conceptAddFrom        []string
	conceptAddNoRetrieval bool
	conceptAddPrint       bool
	conceptAddModel       string
	conceptAddRetrievalK  int
)

var contextConceptCmd = &cobra.Command{
	Use:   "concept",
	Short: "Manage semantic concepts in context/semantic.json",
}

var contextConceptAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Propose a new concept and add it to context/semantic.json",
	Long: `Propose one new semantic concept using retrieval-grounded LLM output.

Unlike 'oz context enrich' (which re-extracts all concepts from the full graph),
'oz context concept add' proposes a single, focused concept by name. It uses the
same retrieval engine as 'oz context query' — but without agent routing — to
ground the model in the workspace's existing context.

The proposed concept is written to context/semantic.json with reviewed: false.
Run 'oz context review' to accept or reject it.

Requires OPENROUTER_API_KEY to be set in the environment.

Examples:

  oz context concept add --name "Query Routing"
  oz context concept add --name "Semantic Overlay" --seed "how concepts are extracted and merged"
  oz context concept add --name "BM25F Scoring" --from specs/routing-packet.md --retrieval-k 8
  oz context concept add --name "Audit Pipeline" --no-retrieval --print`,
	RunE: runContextConceptAdd,
}

func init() {
	contextConceptCmd.AddCommand(contextConceptAddCmd)
	contextConceptAddCmd.Flags().StringVar(&conceptAddName, "name", "", "concept name (required)")
	contextConceptAddCmd.Flags().StringVar(&conceptAddSeed, "seed", "", "optional seed description to anchor the proposal")
	contextConceptAddCmd.Flags().StringArrayVar(&conceptAddFrom, "from", nil, "file paths to anchor the proposal (repeatable)")
	contextConceptAddCmd.Flags().BoolVar(&conceptAddNoRetrieval, "no-retrieval", false, "skip retrieval; send only the allowlist and name to the model")
	contextConceptAddCmd.Flags().BoolVar(&conceptAddPrint, "print", false, "print prompt + proposed JSON to stdout instead of writing to semantic.json")
	contextConceptAddCmd.Flags().StringVar(&conceptAddModel, "model", "", "OpenRouter model ID (default: openrouter/free)")
	contextConceptAddCmd.Flags().IntVar(&conceptAddRetrievalK, "retrieval-k", 5, "max context blocks to include in the prompt")
	if err := contextConceptAddCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
}

func runContextConceptAdd(cmd *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	g, err := loadOrBuildGraph(root, cmd)
	if err != nil {
		return err
	}

	blocks := buildRetrievalBlocks(root)

	if conceptAddPrint {
		return runConceptAddDryRun(root, g, blocks)
	}

	res, err := enrich.ProposeConcept(root, g, enrich.ProposeOptions{
		Name:      conceptAddName,
		Seed:      conceptAddSeed,
		FromFiles: conceptAddFrom,
		Blocks:    blocks,
		Model:     conceptAddModel,
		Progress: func(stage string) {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s %s\n",
				termstyle.Subtle.Render("•"),
				termstyle.Subtle.Render(stage),
			)
		},
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "%s %s\n",
		termstyle.OK.Render("✓"),
		termstyle.Muted.Render("proposal written"),
	)
	cmd.Printf("%s\n", termstyle.Brand.Render("context concept add complete"))
	cmd.Printf("  %s %s\n", termstyle.Subtle.Render("output   "), termstyle.AccentBold.Render("context/semantic.json"))
	cmd.Printf("  %s %s\n", termstyle.Subtle.Render("concept  "), termstyle.AccentBold.Render(res.Concept.Name))
	cmd.Printf("  %s %s\n", termstyle.Subtle.Render("id       "), termstyle.AccentBold.Render(res.Concept.ID))
	cmd.Printf("  %s %s\n", termstyle.Subtle.Render("edges    "), termstyle.AccentBold.Render(fmt.Sprintf("%d added", len(res.Edges))))
	cmd.Printf("  %s %s\n", termstyle.Subtle.Render("model    "), termstyle.AccentBold.Render(res.Model))
	if res.Cost > 0 {
		cmd.Printf("  %s %s\n", termstyle.Subtle.Render("cost     "), termstyle.AccentBold.Render(fmt.Sprintf("$%.4f", res.Cost)))
	}
	if len(res.Skipped) > 0 {
		cmd.Printf("  %s %s\n", termstyle.Subtle.Render("skipped  "), termstyle.AccentBold.Render(fmt.Sprintf("%d items", len(res.Skipped))))
		for _, s := range res.Skipped {
			cmd.Printf("    - %s\n", termstyle.Subtle.Render(s))
		}
	}
	cmd.Printf("  %s %s\n", termstyle.Subtle.Render("next     "), termstyle.AccentBold.Render("run 'oz context review' to accept or reject"))
	return nil
}

// buildRetrievalBlocks runs RetrievalForProposal and converts the top-k blocks
// to enrich.RetrievedBlock. Returns nil when --no-retrieval is set.
func buildRetrievalBlocks(root string) []enrich.RetrievedBlock {
	if conceptAddNoRetrieval {
		return nil
	}
	queryText := conceptAddName
	if conceptAddSeed != "" {
		queryText += " " + conceptAddSeed
	}
	r, err := query.RetrievalForProposal(root, queryText)
	if err != nil {
		return nil
	}
	k := conceptAddRetrievalK
	if k < 1 {
		k = 1
	}
	var blocks []enrich.RetrievedBlock
	for i, b := range r.ContextBlocks {
		if i >= k {
			break
		}
		blocks = append(blocks, enrich.RetrievedBlock{
			File:    b.File,
			Section: b.Section,
			Trust:   b.Trust,
		})
	}
	return blocks
}

// runConceptAddDryRun builds the proposal prompt without calling OpenRouter and
// prints it as JSON for inspection. No semantic.json is written.
func runConceptAddDryRun(root string, g *graph.Graph, blocks []enrich.RetrievedBlock) error {
	existing, _ := semantic.Load(root)
	allowlist := enrich.BuildAllowlist(g, existing)

	var existingNames []string
	if existing != nil {
		for _, c := range existing.Concepts {
			if c.Reviewed {
				existingNames = append(existingNames, c.Name)
			}
		}
	}

	prompt, err := enrich.BuildProposalPrompt(enrich.ProposeOptions{
		Name:      conceptAddName,
		Seed:      conceptAddSeed,
		FromFiles: conceptAddFrom,
		Blocks:    blocks,
	}, allowlist, existingNames)
	if err != nil {
		return fmt.Errorf("build prompt: %w", err)
	}

	out := struct {
		PromptChars     int    `json:"prompt_chars"`
		PromptEstTokens int    `json:"prompt_est_tokens"`
		RetrievalBlocks int    `json:"retrieval_blocks"`
		AllowlistCount  int    `json:"allowlist_count"`
		Prompt          string `json:"prompt"`
	}{
		PromptChars:     len(prompt),
		PromptEstTokens: len(prompt) / 4,
		RetrievalBlocks: len(blocks),
		AllowlistCount:  len(allowlist),
		Prompt:          prompt,
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

// loadOrBuildGraph loads context/graph.json, building it first if absent.
func loadOrBuildGraph(root string, cmd *cobra.Command) (*graph.Graph, error) {
	g, err := ozcontext.LoadGraph(root)
	if err == nil {
		return g, nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "%s %s\n",
		termstyle.Subtle.Render("•"),
		termstyle.Subtle.Render("context graph not found; building it first"),
	)
	result, buildErr := ozcontext.Build(root)
	if buildErr != nil {
		return nil, fmt.Errorf("build context graph: %w", buildErr)
	}
	if serErr := ozcontext.Serialize(root, result.Graph); serErr != nil {
		return nil, fmt.Errorf("write graph: %w", serErr)
	}
	return result.Graph, nil
}
