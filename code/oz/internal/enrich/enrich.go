// Package enrich implements the oz context enrich pipeline.
//
// It sends the structural graph to an LLM via OpenRouter and writes
// context/semantic.json with extracted concept nodes and typed relationships.
// The overlay is merged with any existing semantic.json, preserving items
// that a human has already reviewed (reviewed: true).
package enrich

import (
	"fmt"
	"time"

	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/openrouter"
	"github.com/joaoajmatos/oz/internal/semantic"
)

// Options configures an enrichment run.
type Options struct {
	// Model is the OpenRouter model ID. Defaults to openrouter.DefaultModel.
	Model string
	// Progress receives stage updates during enrichment when non-nil.
	Progress func(stage string)
}

// Result reports the outcome of an enrichment run.
type Result struct {
	Model         string
	ConceptsAdded int
	EdgesAdded    int
	Skipped       []string
	Cost          float64 // USD estimate from OpenRouter; 0 if unavailable
}

// Run executes the full enrichment pipeline:
//
//  1. Build prompt from graph
//  2. Send to OpenRouter LLM
//  3. Parse and validate response
//  4. Merge with existing semantic.json (preserve reviewed items)
//  5. Write context/semantic.json
func Run(workspacePath string, g *graph.Graph, opts Options) (*Result, error) {
	report := func(stage string) {
		if opts.Progress != nil {
			opts.Progress(stage)
		}
	}

	report("initializing OpenRouter client")
	client, err := openrouter.New(opts.Model)
	if err != nil {
		return nil, err
	}

	report("building enrichment prompt")
	prompt, err := BuildPrompt(g)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	report("requesting model response")
	resp, err := client.Complete([]openrouter.Message{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}

	// Build a set of graph node IDs for edge validation in the parser.
	nodeIDs := make(map[string]struct{}, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeIDs[n.ID] = struct{}{}
	}

	report("parsing and validating response")
	concepts, edges, skipped := ParseResponse(resp.Choices[0].Message.Content, nodeIDs)

	report("loading existing semantic overlay")
	existing, err := semantic.Load(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("load existing overlay: %w", err)
	}

	incoming := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		GraphHash:     g.ContentHash,
		Model:         client.Model,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Concepts:      concepts,
		Edges:         edges,
	}

	report("merging and writing semantic overlay")
	merged := semantic.Merge(existing, incoming)

	if err := semantic.Write(workspacePath, merged); err != nil {
		return nil, fmt.Errorf("write semantic.json: %w", err)
	}

	res := &Result{
		Model:         client.Model,
		ConceptsAdded: len(concepts),
		EdgesAdded:    len(edges),
		Skipped:       skipped,
	}
	if resp.Usage != nil {
		res.Cost = resp.Usage.Cost
	}
	return res, nil
}
