package query

import "github.com/joaoajmatos/oz/internal/graph"

// AgentDoc holds the tokenized BM25F fields for one agent.
type AgentDoc struct {
	Name string

	// Tokenized field contents (one token per occurrence, for TF counting).
	Scope            []string // from scope paths
	Role             []string // from role text
	Responsibilities []string // from responsibilities text
	ReadChain        []string // from read-chain paths
	OutOfScope       []string // from out-of-scope text (penalty signal)
}

// Field names exposed by AgentDoc.Fields. Kept as constants so the scorer
// and any future consumers agree on the routing-corpus shape.
const (
	AgentFieldScope            = "scope"
	AgentFieldRole             = "role"
	AgentFieldResponsibilities = "responsibilities"
	AgentFieldReadChain        = "readchain"
)

// Fields satisfies the generic FieldDoc interface used by the BM25 core.
// OutOfScope is deliberately excluded — it feeds the penalty term, not TF.
func (d AgentDoc) Fields() map[string][]string {
	return map[string][]string{
		AgentFieldScope:            d.Scope,
		AgentFieldRole:             d.Role,
		AgentFieldResponsibilities: d.Responsibilities,
		AgentFieldReadChain:        d.ReadChain,
	}
}

// BuildAgentDocs creates one AgentDoc per agent node in the graph.
func BuildAgentDocs(nodes []graph.Node, cfg ScoringConfig) []AgentDoc {
	var docs []AgentDoc
	for _, n := range nodes {
		if n.Type != graph.NodeTypeAgent {
			continue
		}
		docs = append(docs, agentDocFromNode(n, cfg.UseBigrams))
	}
	return docs
}

func agentDocFromNode(n graph.Node, useBigrams bool) AgentDoc {
	return AgentDoc{
		Name:             n.Name,
		Scope:            TokenizePathsMulti(n.Scope, useBigrams),
		Role:             TokenizeMulti(n.Role, useBigrams),
		Responsibilities: TokenizeMulti(n.Responsibilities, useBigrams),
		ReadChain:        TokenizePathsMulti(n.ReadChain, useBigrams),
		OutOfScope:       TokenizeMulti(n.OutOfScope, useBigrams),
	}
}

// termFreq returns the frequency of term in tokens.
func termFreq(term string, tokens []string) int {
	count := 0
	for _, t := range tokens {
		if t == term {
			count++
		}
	}
	return count
}

// containsTerm reports whether term appears in tokens.
func containsTerm(term string, tokens []string) bool {
	for _, t := range tokens {
		if t == term {
			return true
		}
	}
	return false
}
