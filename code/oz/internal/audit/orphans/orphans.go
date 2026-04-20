// Package orphans implements the oz audit orphans check.
// It reports spec files, decisions, doc files, and context snapshots that
// are not referenced by any agent or document.
package orphans

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oz-tools/oz/internal/audit"
	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/graph"
)

// Check implements audit.Check for orphan detection.
type Check struct{}

// Name returns the check name.
func (c *Check) Name() string { return "orphans" }

// Codes returns the finding codes this check may produce.
func (c *Check) Codes() []string { return []string{"ORPH001", "ORPH002", "ORPH003"} }

// Run loads the graph from root and returns orphan findings.
func (c *Check) Run(root string, _ audit.Options) ([]audit.Finding, error) {
	g, err := ozcontext.LoadGraph(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("graph.json not found — run 'oz context build' first")
		}
		return nil, fmt.Errorf("orphans: load graph: %w", err)
	}
	return runCheck(g), nil
}

// edgeCounts tracks inbound edge counts per node.
type edgeCounts struct {
	// specEdges counts reads + references + supports (used for ORPH001).
	specEdges int
	// anyEdge counts all inbound edges (used for ORPH002, ORPH003).
	anyEdge int
}

// runCheck runs orphan detection against a pre-loaded graph.
// Separated from Run for testability.
func runCheck(g *graph.Graph) []audit.Finding {
	inbound := make(map[string]*edgeCounts, len(g.Nodes))
	for _, n := range g.Nodes {
		inbound[n.ID] = &edgeCounts{}
	}
	for _, e := range g.Edges {
		c := inbound[e.To]
		if c == nil {
			continue
		}
		switch e.Type {
		case graph.EdgeTypeReads, graph.EdgeTypeReferences, graph.EdgeTypeSupports:
			c.specEdges++
		}
		c.anyEdge++
	}

	agentTopics := collectAgentTopics(g)

	var findings []audit.Finding
	findings = append(findings, checkSpecFiles(g, inbound)...)
	findings = append(findings, checkDecisions(g, inbound)...)
	findings = append(findings, checkDocFiles(g, inbound)...)
	findings = append(findings, checkContextSnapshots(g, inbound, agentTopics)...)
	return findings
}

// collectAgentTopics returns a set of lowercase context topic names declared by all agents.
func collectAgentTopics(g *graph.Graph) map[string]bool {
	topics := make(map[string]bool)
	for _, n := range g.Nodes {
		if n.Type != graph.NodeTypeAgent {
			continue
		}
		for _, t := range n.ContextTopics {
			topics[strings.ToLower(strings.TrimSpace(t))] = true
		}
	}
	return topics
}

// checkSpecFiles emits ORPH001 for each spec file where no section has any
// inbound reads/references/supports edge.
//
// Spec files produce one spec_section node per H2 heading. A reads edge from
// an agent's read-chain connects to the first section node for that file (via
// the file→node index). Checking at the file level avoids false positives for
// secondary sections that are part of a referenced file.
func checkSpecFiles(g *graph.Graph, inbound map[string]*edgeCounts) []audit.Finding {
	type fileState struct {
		file    string
		nodeIDs []string
		maxIn   int
	}
	byFile := make(map[string]*fileState)

	for _, n := range g.Nodes {
		if n.Type != graph.NodeTypeSpecSection {
			continue
		}
		fs := byFile[n.File]
		if fs == nil {
			fs = &fileState{file: n.File}
			byFile[n.File] = fs
		}
		fs.nodeIDs = append(fs.nodeIDs, n.ID)
		if c := inbound[n.ID]; c != nil && c.specEdges > fs.maxIn {
			fs.maxIn = c.specEdges
		}
	}

	var findings []audit.Finding
	for _, fs := range byFile {
		if fs.maxIn > 0 {
			continue
		}
		findings = append(findings, audit.Finding{
			Check:    "orphans",
			Code:     "ORPH001",
			Severity: audit.SeverityError,
			Message:  fmt.Sprintf("spec file %s is not referenced by any agent or document", fs.file),
			File:     fs.file,
			Hint:     "add this spec to an agent's read-chain, or reference it from another document",
			Refs:     fs.nodeIDs,
		})
	}
	return findings
}

// checkDecisions emits ORPH001 for each decision node with no inbound
// reads/references/supports edges.
func checkDecisions(g *graph.Graph, inbound map[string]*edgeCounts) []audit.Finding {
	var findings []audit.Finding
	for _, n := range g.Nodes {
		if n.Type != graph.NodeTypeDecision {
			continue
		}
		c := inbound[n.ID]
		if c != nil && c.specEdges > 0 {
			continue
		}
		findings = append(findings, audit.Finding{
			Check:    "orphans",
			Code:     "ORPH001",
			Severity: audit.SeverityError,
			Message:  fmt.Sprintf("decision %s is not referenced by any agent or document", n.Name),
			File:     n.File,
			Hint:     "reference this decision from a spec or an agent's read-chain",
			Refs:     []string{n.ID},
		})
	}
	return findings
}

// checkDocFiles emits ORPH002 for each doc file where no section has any
// inbound edge of any type.
func checkDocFiles(g *graph.Graph, inbound map[string]*edgeCounts) []audit.Finding {
	type fileState struct {
		file  string
		maxIn int
	}
	byFile := make(map[string]*fileState)

	for _, n := range g.Nodes {
		if n.Type != graph.NodeTypeDoc {
			continue
		}
		fs := byFile[n.File]
		if fs == nil {
			fs = &fileState{file: n.File}
			byFile[n.File] = fs
		}
		if c := inbound[n.ID]; c != nil && c.anyEdge > fs.maxIn {
			fs.maxIn = c.anyEdge
		}
	}

	var findings []audit.Finding
	for _, fs := range byFile {
		if fs.maxIn > 0 {
			continue
		}
		findings = append(findings, audit.Finding{
			Check:    "orphans",
			Code:     "ORPH002",
			Severity: audit.SeverityWarn,
			Message:  fmt.Sprintf("doc file %s has no inbound references", fs.file),
			File:     fs.file,
			Hint:     "reference this document from a spec, another doc, or an agent's read-chain",
		})
	}
	return findings
}

// checkContextSnapshots emits ORPH003 for each context snapshot that has
// no inbound edges and whose topic is not declared in any agent's context_topics.
func checkContextSnapshots(g *graph.Graph, inbound map[string]*edgeCounts, agentTopics map[string]bool) []audit.Finding {
	var findings []audit.Finding
	for _, n := range g.Nodes {
		if n.Type != graph.NodeTypeContextSnapshot {
			continue
		}
		c := inbound[n.ID]
		if c != nil && c.anyEdge > 0 {
			continue
		}
		topic := contextTopic(n.File)
		if agentTopics[strings.ToLower(topic)] {
			continue
		}
		findings = append(findings, audit.Finding{
			Check:    "orphans",
			Code:     "ORPH003",
			Severity: audit.SeverityInfo,
			Message:  fmt.Sprintf("context snapshot %s is not referenced by any agent", n.File),
			File:     n.File,
			Hint:     "add this topic to an agent's context_topics section, or remove the snapshot",
			Refs:     []string{n.ID},
		})
	}
	return findings
}

// contextTopic extracts the topic directory name from a context snapshot path.
// "context/auth/snapshot.md" → "auth"
func contextTopic(file string) string {
	parts := strings.Split(filepath.ToSlash(file), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
