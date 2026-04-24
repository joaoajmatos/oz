package query

import (
	"strings"

	"github.com/joaoajmatos/oz/internal/graph"
)

// AgentDoc holds the tokenized BM25F fields for one agent.
type AgentDoc struct {
	Name string

	// Tokenized field contents (one token per occurrence, for TF counting).
	Scope            []string // from scope paths
	Role             []string // from role text
	Responsibilities []string // from responsibilities text
	ReadChain        []string // from read-chain paths
	Skills           []string // from skills (backtick path lines)
	Rules            []string // from rules file paths
	ContextTopics    []string // from "Context topics" list items
	OutOfScope       []string // from out-of-scope text (penalty signal)
}

// Field names exposed by AgentDoc.Fields. Kept as constants so the scorer
// and any future consumers agree on the routing-corpus shape.
const (
	AgentFieldScope            = "scope"
	AgentFieldRole             = "role"
	AgentFieldResponsibilities = "responsibilities"
	AgentFieldReadChain        = "readchain"
	AgentFieldSkills           = "skills"
	AgentFieldRules            = "rules"
	AgentFieldContextTopics    = "context_topics"
)

// Fields satisfies the generic FieldDoc interface used by the BM25 core.
// OutOfScope is deliberately excluded — it feeds the penalty term, not TF.
func (d AgentDoc) Fields() map[string][]string {
	return map[string][]string{
		AgentFieldScope:            d.Scope,
		AgentFieldRole:             d.Role,
		AgentFieldResponsibilities: d.Responsibilities,
		AgentFieldReadChain:        d.ReadChain,
		AgentFieldSkills:           d.Skills,
		AgentFieldRules:            d.Rules,
		AgentFieldContextTopics:    d.ContextTopics,
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
	var ctxText string
	switch {
	case n.ContextTopicsBody != "":
		ctxText = n.ContextTopicsBody
	case len(n.ContextTopics) > 0:
		ctxText = strings.Join(n.ContextTopics, "\n")
	}
	var skillsTokens []string
	if n.SkillsBody != "" {
		skillsTokens = TokenizeMulti(n.SkillsBody, useBigrams)
	} else {
		skillsTokens = TokenizePathsMulti(n.Skills, useBigrams)
	}
	return AgentDoc{
		Name:             n.Name,
		Scope:            TokenizePathsMulti(n.Scope, useBigrams),
		Role:             TokenizeMulti(n.Role, useBigrams),
		Responsibilities: TokenizeMulti(n.Responsibilities, useBigrams),
		ReadChain:        TokenizePathsMulti(n.ReadChain, useBigrams),
		Skills:           skillsTokens,
		Rules:            TokenizePathsMulti(n.Rules, useBigrams),
		ContextTopics:    TokenizeMulti(ctxText, useBigrams),
		OutOfScope:       TokenizeMulti(n.OutOfScope, useBigrams),
	}
}

// termMatchesPositiveRoutingFields is true if term appears in any BM25 field used
// for agent routing (excludes out_of_scope, which is penalty-only). Used to avoid
// subtracting the out-of-scope penalty when the same term is a deliberate match
// in role, responsibilities, scope, read-chain, skills, rules, or context topics.
func (d AgentDoc) termMatchesPositiveRoutingFields(term string) bool {
	if containsTerm(term, d.Scope) {
		return true
	}
	if containsTerm(term, d.Role) {
		return true
	}
	if containsTerm(term, d.Responsibilities) {
		return true
	}
	if containsTerm(term, d.ReadChain) {
		return true
	}
	if containsTerm(term, d.Skills) {
		return true
	}
	if containsTerm(term, d.Rules) {
		return true
	}
	if containsTerm(term, d.ContextTopics) {
		return true
	}
	return false
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
