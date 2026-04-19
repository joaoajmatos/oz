package query

import "github.com/oz-tools/oz/internal/graph"

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

// BuildAgentDocs creates one AgentDoc per agent node in the graph.
func BuildAgentDocs(nodes []graph.Node) []AgentDoc {
	var docs []AgentDoc
	for _, n := range nodes {
		if n.Type != graph.NodeTypeAgent {
			continue
		}
		docs = append(docs, agentDocFromNode(n))
	}
	return docs
}

func agentDocFromNode(n graph.Node) AgentDoc {
	return AgentDoc{
		Name:             n.Name,
		Scope:            TokenizePathsMulti(n.Scope),
		Role:             TokenizeMulti(n.Role),
		Responsibilities: TokenizeMulti(n.Responsibilities),
		ReadChain:        TokenizePathsMulti(n.ReadChain),
		OutOfScope:       TokenizeMulti(n.OutOfScope),
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
