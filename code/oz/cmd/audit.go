package cmd

import (
	"fmt"

	"github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/graph"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit workspace health using the context graph (stub)",
	Long: `Load context/graph.json and print a summary of nodes and edges.

The workspace root is found by walking up from the current directory until
AGENTS.md and OZ.md exist, so this works from any subdirectory inside the workspace.

Full audit logic (drift detection, orphan detection, coverage checks) ships
in a later sprint. This stub validates the graph contract is sound.`,
	RunE: runAudit,
}

func runAudit(cmd *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	g, err := context.LoadGraph(root)
	if err != nil {
		return fmt.Errorf("load context graph: %w\n\nHint: run 'oz context build' first", err)
	}

	// Count by node type.
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

	cmd.Printf("\nfull audit logic ships in a later sprint.\n")
	return nil
}

func init() {
	// auditCmd is registered in root.go
	_ = auditCmd
}
