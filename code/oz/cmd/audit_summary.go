package cmd

import (
	"fmt"

	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/spf13/cobra"
)

var auditSummaryCmd = &cobra.Command{
	Use:   "graph-summary",
	Short: "Print a node/edge summary of context/graph.json (V0 output)",
	Long: `Load context/graph.json and print a summary of nodes and edges.

This is the original V0 audit output, preserved as a subcommand.
Run 'oz context build' first to generate graph.json.`,
	RunE: runAuditSummary,
}

func runAuditSummary(cmd *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	g, err := ozcontext.LoadGraph(root)
	if err != nil {
		return fmt.Errorf("load context graph: %w\n\nHint: run 'oz context build' first", err)
	}

	typeCounts := make(map[string]int)
	for _, n := range g.Nodes {
		typeCounts[n.Type]++
	}

	edgeCounts := make(map[string]int)
	for _, e := range g.Edges {
		edgeCounts[e.Type]++
	}

	cmd.Printf("oz audit — workspace graph summary\n")
	cmd.Printf("  schema version : %s\n", g.SchemaVersion)
	cmd.Printf("  content hash   : %s\n", g.ContentHash[:12])
	cmd.Printf("\n  nodes (%d total):\n", len(g.Nodes))
	for _, nodeType := range []string{
		graph.NodeTypeAgent,
		graph.NodeTypeSpecSection,
		graph.NodeTypeDecision,
		graph.NodeTypeDoc,
		graph.NodeTypeContextSnapshot,
		graph.NodeTypeNote,
	} {
		if c := typeCounts[nodeType]; c > 0 {
			cmd.Printf("    %-20s %d\n", nodeType, c)
		}
	}

	cmd.Printf("\n  edges (%d total):\n", len(g.Edges))
	for _, edgeType := range []string{
		graph.EdgeTypeReads,
		graph.EdgeTypeOwns,
		graph.EdgeTypeReferences,
		graph.EdgeTypeSupports,
		graph.EdgeTypeCrystallizedFrom,
	} {
		if c := edgeCounts[edgeType]; c > 0 {
			cmd.Printf("    %-20s %d\n", edgeType, c)
		}
	}

	return nil
}
