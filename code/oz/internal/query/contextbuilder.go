package query

import (
	"sort"

	"github.com/oz-tools/oz/internal/convention"
	"github.com/oz-tools/oz/internal/graph"
)

// BuildContextBlocks returns the ordered context blocks for the winning agent.
//
// Selection strategy:
//  1. Nodes the agent explicitly reads (reads edges from agent node).
//  2. All spec_section and decision nodes (always relevant).
//  3. context_snapshot nodes connected to the agent.
//  4. doc nodes reachable via the agent's reads edges.
//
// Blocks are sorted by trust tier (specs > docs > context > notes).
// Notes are excluded unless cfg.IncludeNotes is true.
// Notes are always listed in Excluded when not included.
func BuildContextBlocks(g *graph.Graph, agentName string, cfg ScoringConfig) (blocks []ContextBlock, excluded []string) {
	agentID := "agent:" + agentName

	// Build file→node lookup for quick resolution.
	fileToNode := buildFileNodeMap(g)

	// Collect node IDs that the agent reads (via reads edges).
	readsSet := make(map[string]bool)
	for _, e := range g.Edges {
		if e.From == agentID && e.Type == graph.EdgeTypeReads {
			readsSet[e.To] = true
		}
	}

	// Collect all nodes that should appear in context blocks.
	selected := make(map[string]graph.Node)

	for _, n := range g.Nodes {
		switch n.Type {
		case graph.NodeTypeSpecSection, graph.NodeTypeDecision:
			// Always include spec material.
			selected[n.ID] = n
		case graph.NodeTypeDoc:
			// Include docs that the agent reads.
			if readsSet[n.ID] || readsSet[fileToNode[n.File]] {
				selected[n.ID] = n
			}
		case graph.NodeTypeContextSnapshot:
			if readsSet[n.ID] {
				selected[n.ID] = n
			}
		case graph.NodeTypeNote:
			if cfg.IncludeNotes {
				selected[n.ID] = n
			}
		}
	}

	// Convert to ContextBlock slice and sort by trust tier.
	for _, n := range selected {
		blocks = append(blocks, ContextBlock{
			File:    n.File,
			Section: n.Section,
			Trust:   tierToTrust(n.Tier),
		})
	}

	sortContextBlocks(blocks)

	// Excluded: note paths (when not included).
	if !cfg.IncludeNotes {
		hasNotes := false
		for _, n := range g.Nodes {
			if n.Type == graph.NodeTypeNote {
				hasNotes = true
				break
			}
		}
		if hasNotes {
			excluded = []string{"notes/"}
		}
	}

	return blocks, excluded
}

// BuildScopeForAgent returns the scope paths for the winning agent node.
func BuildScopeForAgent(g *graph.Graph, agentName string) []string {
	for _, n := range g.Nodes {
		if n.Type == graph.NodeTypeAgent && n.Name == agentName {
			return n.Scope
		}
	}
	return nil
}

// ---- helpers ----------------------------------------------------------------

// tierToTrust maps graph tier strings to context block trust strings.
func tierToTrust(tier convention.Tier) string {
	switch tier {
	case convention.TierSpecs:
		return "high"
	case convention.TierDocs, convention.TierContext:
		return "medium"
	case convention.TierNotes:
		return "low"
	default:
		return "medium"
	}
}

// sortContextBlocks sorts blocks: high trust first, then medium, then low.
// Within the same trust level, sort by file path for stability.
func sortContextBlocks(blocks []ContextBlock) {
	trustRank := map[string]int{"high": 0, "medium": 1, "low": 2}
	sort.SliceStable(blocks, func(i, j int) bool {
		ri := trustRank[blocks[i].Trust]
		rj := trustRank[blocks[j].Trust]
		if ri != rj {
			return ri < rj
		}
		if blocks[i].File != blocks[j].File {
			return blocks[i].File < blocks[j].File
		}
		return blocks[i].Section < blocks[j].Section
	})
}

// buildFileNodeMap returns file → first node ID for that file.
func buildFileNodeMap(g *graph.Graph) map[string]string {
	m := make(map[string]string)
	for _, n := range g.Nodes {
		if n.File == "" {
			continue
		}
		if _, ok := m[n.File]; !ok {
			m[n.File] = n.ID
		}
	}
	return m
}
