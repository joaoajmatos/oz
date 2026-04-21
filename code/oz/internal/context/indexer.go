package context

import (
	"path/filepath"
	"strings"

	"github.com/joaoajmatos/oz/internal/convention"
	"github.com/joaoajmatos/oz/internal/graph"
)

// IndexMarkdownFile produces graph nodes for a non-agent markdown file.
// For spec and doc files, each H2 section becomes a separate node.
// For decision, context snapshot, and note files, one node is produced for
// the whole file.
func IndexMarkdownFile(f DiscoveredFile) ([]graph.Node, error) {
	switch f.Kind {
	case KindSpec:
		return indexSections(f, graph.NodeTypeSpecSection, convention.TierSpecs)
	case KindDecision:
		return indexDecision(f)
	case KindDoc:
		return indexSections(f, graph.NodeTypeDoc, convention.TierDocs)
	case KindContextSnapshot:
		return indexContextSnapshot(f)
	case KindNote:
		return indexNote(f)
	default:
		return nil, nil
	}
}

// indexSections creates one node per H2 heading in the markdown file.
func indexSections(f DiscoveredFile, nodeType string, tier convention.Tier) ([]graph.Node, error) {
	sections, err := ParseMarkdownSections(f.AbsPath)
	if err != nil {
		return nil, err
	}

	var nodes []graph.Node
	for _, sec := range sections {
		if sec.Heading == "" {
			continue
		}
		// Skip separator headings (e.g. "---").
		if strings.TrimSpace(sec.Heading) == "---" {
			continue
		}
		id := SectionNodeID(nodeType, f.Path, sec.Heading)
		nodes = append(nodes, graph.Node{
			ID:      id,
			Type:    nodeType,
			File:    f.Path,
			Name:    sec.Heading,
			Tier:    tier,
			Section: sec.Heading,
		})
	}

	// If no sections were found, produce a single file-level node.
	if len(nodes) == 0 {
		id := SectionNodeID(nodeType, f.Path, "")
		nodes = append(nodes, graph.Node{
			ID:   id,
			Type: nodeType,
			File: f.Path,
			Name: filepath.Base(f.Path),
			Tier: tier,
		})
	}

	return nodes, nil
}

// indexDecision creates a single node for a specs/decisions/ file.
func indexDecision(f DiscoveredFile) ([]graph.Node, error) {
	id := DecisionNodeID(f.Path)
	base := filepath.Base(f.Path)
	name := strings.TrimSuffix(base, ".md")

	return []graph.Node{{
		ID:   id,
		Type: graph.NodeTypeDecision,
		File: f.Path,
		Name: name,
		Tier: convention.TierSpecs,
	}}, nil
}

// indexContextSnapshot creates a single node for a context/ file.
func indexContextSnapshot(f DiscoveredFile) ([]graph.Node, error) {
	id := ContextSnapshotNodeID(f.Path)
	return []graph.Node{{
		ID:   id,
		Type: graph.NodeTypeContextSnapshot,
		File: f.Path,
		Name: filepath.Base(f.Path),
		Tier: convention.TierContext,
	}}, nil
}

// indexNote creates a single node for a notes/ file.
func indexNote(f DiscoveredFile) ([]graph.Node, error) {
	id := NoteNodeID(f.Path)
	return []graph.Node{{
		ID:   id,
		Type: graph.NodeTypeNote,
		File: f.Path,
		Name: filepath.Base(f.Path),
		Tier: convention.TierNotes,
	}}, nil
}
