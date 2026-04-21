package query

import (
	"sort"

	"github.com/joaoajmatos/oz/internal/graph"
)

// RawQueryDebug is the JSON payload for `oz context query --raw` (PRD C-09):
// routing result, per-agent scores, and a query-relevant subgraph of the
// structural graph (not the full workspace graph).
type RawQueryDebug struct {
	Query    string            `json:"query"`
	Result   Result            `json:"result"`
	Agents   []AgentScoreDebug `json:"agents"`
	Subgraph QuerySubgraph     `json:"subgraph"`
}

// AgentScoreDebug carries one agent's BM25F raw score and softmax confidence.
type AgentScoreDebug struct {
	Name         string  `json:"name"`
	RawScore     float64 `json:"raw_score"`
	Confidence   float64 `json:"confidence"`
}

// QuerySubgraph is a filtered slice of graph.json for debugging.
type QuerySubgraph struct {
	Nodes []graph.Node `json:"nodes"`
	Edges []graph.Edge `json:"edges"`
}

// BuildRawQueryDebug runs the same routing pipeline as RunWithOptions and
// assembles a bounded debug view: agent score table plus subgraph nodes/edges.
func BuildRawQueryDebug(workspacePath, queryText string, opts Options) RawQueryDebug {
	st := runRouting(workspacePath, queryText, opts)

	out := RawQueryDebug{
		Query:    queryText,
		Result:   st.Result,
		Subgraph: BuildQuerySubgraph(st.G, st.Result),
	}
	for i, s := range st.Scores {
		row := AgentScoreDebug{Name: s.Agent, RawScore: s.Value}
		if i < len(st.Conf) {
			row.Confidence = st.Conf[i]
		}
		out.Agents = append(out.Agents, row)
	}
	sort.Slice(out.Agents, func(i, j int) bool {
		return out.Agents[i].Name < out.Agents[j].Name
	})
	return out
}

// BuildQuerySubgraph returns all agent nodes, nodes matching the routing
// packet's context blocks, and edges whose endpoints are both in that set.
func BuildQuerySubgraph(g *graph.Graph, result Result) QuerySubgraph {
	if g == nil {
		return QuerySubgraph{}
	}
	ids := make(map[string]bool)
	for _, n := range g.Nodes {
		if n.Type == graph.NodeTypeAgent {
			ids[n.ID] = true
		}
	}
	for _, cb := range result.ContextBlocks {
		for _, n := range g.Nodes {
			if n.File != cb.File {
				continue
			}
			if cb.Section == "" || n.Section == cb.Section {
				ids[n.ID] = true
			}
		}
	}

	var nodes []graph.Node
	for _, n := range g.Nodes {
		if ids[n.ID] {
			nodes = append(nodes, n)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	var edges []graph.Edge
	for _, e := range g.Edges {
		if ids[e.From] && ids[e.To] {
			edges = append(edges, e)
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Type < edges[j].Type
	})

	return QuerySubgraph{Nodes: nodes, Edges: edges}
}
