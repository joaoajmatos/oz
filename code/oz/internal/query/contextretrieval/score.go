package contextretrieval

import (
	"sort"

	"github.com/joaoajmatos/oz/internal/query/bm25"
)

// Block is one retrieval candidate for context_blocks ranking.
// TokenFields are already tokenized/stemmed by the caller.
type Block struct {
	File    string
	Section string
	Trust   string // "high", "medium", "low"
	Tier    string // "specs", "docs", "context", "notes" (for trust_boost lookup)

	TokenFields map[string][]string

	// ConnectedAgents identifies agents connected to this block via reads/owns/etc.
	ConnectedAgents map[string]bool
}

// Fields adapts Block to query.FieldDoc for shared BM25 helpers.
func (b Block) Fields() map[string][]string {
	return b.TokenFields
}

// RetrievalConfig configures BM25 and multiplicative boosts for retrieval scoring.
type RetrievalConfig struct {
	K1     float64
	Fields []bm25.BM25Field

	TrustBoost         map[string]float64
	AgentAffinityBoost float64
}

// ScoredBlock is a ranked candidate with scoring breakdown.
type ScoredBlock struct {
	Block Block

	BM25          float64
	TrustBoost    float64
	AffinityBoost float64
	Relevance     float64
}

// Score ranks retrieval blocks by:
//  1) relevance DESC
//  2) trust DESC
//  3) file ASC
//  4) section ASC
//
// Relevance is BM25(query, block_fields) * trust_boost * affinity_boost.
func Score(queryTokens []string, blocks []Block, cfg RetrievalConfig, winningAgent string) []ScoredBlock {
	if len(queryTokens) == 0 || len(blocks) == 0 {
		return nil
	}

	docs := make([]bm25.FieldDoc, len(blocks))
	for i := range blocks {
		docs[i] = blocks[i]
	}
	avgLen := bm25.AvgFieldLengths(docs, cfg.Fields)
	df := bm25.ComputeDF(docs)

	scored := make([]ScoredBlock, len(blocks))
	for i, b := range blocks {
		bm25 := bm25.BM25Score(
			queryTokens,
			b.Fields(),
			cfg.Fields,
			cfg.K1,
			avgLen,
			df,
			len(blocks),
		)
		trustBoost := 1.0
		if v, ok := cfg.TrustBoost[b.Tier]; ok {
			trustBoost = v
		} else if v, ok := cfg.TrustBoost[b.Trust]; ok {
			trustBoost = v
		}
		affinityBoost := 1.0
		if winningAgent != "" && b.ConnectedAgents[winningAgent] && cfg.AgentAffinityBoost > 0 {
			affinityBoost = cfg.AgentAffinityBoost
		}
		scored[i] = ScoredBlock{
			Block:         b,
			BM25:          bm25,
			TrustBoost:    trustBoost,
			AffinityBoost: affinityBoost,
			Relevance:     bm25 * trustBoost * affinityBoost,
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Relevance != scored[j].Relevance {
			return scored[i].Relevance > scored[j].Relevance
		}
		ti := trustRank(scored[i].Block.Trust)
		tj := trustRank(scored[j].Block.Trust)
		if ti != tj {
			return ti > tj
		}
		if scored[i].Block.File != scored[j].Block.File {
			return scored[i].Block.File < scored[j].Block.File
		}
		return scored[i].Block.Section < scored[j].Block.Section
	})

	return scored
}

func trustRank(trust string) int {
	switch trust {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

