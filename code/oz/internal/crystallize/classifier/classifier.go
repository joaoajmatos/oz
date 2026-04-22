// Package classifier classifies notes from the notes/ tier into canonical
// artifact types for promotion by oz crystallize.
//
// Classification order (highest precedence first):
//
//  1. Frontmatter tag override: a note with `crystallize: <type>` in its YAML
//     frontmatter is classified immediately without calling the LLM or heuristic.
//  2. LLM classifier: sends the note to OpenRouter with workspace-aware few-shot
//     context. Result is cached in .oz/crystallize-cache.json by file hash.
//  3. Heuristic fallback: keyword + structure signal scoring. Used when no API
//     key is present, --no-enrich is set, or the LLM call fails.
//
// See ADR-0002 for the rationale behind this ordering.
package classifier

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joaoajmatos/oz/internal/crystallize/cache"
	"github.com/joaoajmatos/oz/internal/crystallize/classifier/heuristic"
)

// ArtifactType is one of the five canonical artifact types oz crystallize can
// promote a note into.
type ArtifactType string

const (
	TypeADR      ArtifactType = "adr"
	TypeSpec     ArtifactType = "spec"
	TypeGuide    ArtifactType = "guide"
	TypeArch     ArtifactType = "arch"
	TypeOpenItem ArtifactType = "open-item"
	TypeUnknown  ArtifactType = "unknown"
)

// Confidence is the classifier's certainty about a classification.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// Classification is the result of classifying a single note.
type Classification struct {
	// Type is the detected artifact type.
	Type ArtifactType
	// Confidence is the classifier's certainty.
	Confidence Confidence
	// Title is a suggested canonical title for the promoted artifact.
	Title string
	// Reason is a one-sentence explanation of the classification.
	Reason string
	// Source indicates which classifier produced this result.
	Source ClassifierSource
}

// ClassifierSource identifies which classifier produced a Classification.
type ClassifierSource string

const (
	SourceFrontmatter ClassifierSource = "frontmatter"
	SourceLLM         ClassifierSource = "llm"
	SourceHeuristic   ClassifierSource = "heuristic"
)

// IsAutoAcceptable reports whether this classification can be auto-accepted by
// --accept-all without interactive review. Only high-confidence results qualify.
func (c Classification) IsAutoAcceptable() bool {
	return c.Confidence == ConfidenceHigh && c.Type != TypeUnknown
}

// Options configures a Classifier.
type Options struct {
	// WorkspaceRoot is the absolute path to the oz workspace root.
	WorkspaceRoot string
	// Model is the OpenRouter model ID. Defaults to openrouter.DefaultModel.
	Model string
	// NoEnrich disables the LLM classifier and forces heuristic-only.
	NoEnrich bool
	// NoCache disables cache reads, forcing a fresh LLM call every time.
	NoCache bool
	// Verbose enables detailed logging to the provided function.
	Verbose func(msg string)
}

// Classifier classifies notes using the LLM (primary) or heuristic (fallback).
type Classifier struct {
	opts      Options
	llm       *llmClassifier
	heuristic *heuristic.Classifier
	cache     *cache.Cache
}

// New creates a Classifier. If the LLM client cannot be initialised (missing
// API key or NoEnrich set), the classifier silently falls back to heuristic.
func New(opts Options) *Classifier {
	c := &Classifier{
		opts:      opts,
		heuristic: heuristic.New(),
	}

	if !opts.NoEnrich {
		llm, err := newLLMClassifier(opts.WorkspaceRoot, opts.Model)
		if err != nil {
			c.logf("LLM classifier unavailable (%v); falling back to heuristic", err)
		} else {
			c.llm = llm
		}
	}

	if !opts.NoCache {
		c.cache = cache.New(filepath.Join(opts.WorkspaceRoot, ".oz", "crystallize-cache.json"))
	}

	return c
}

// Classify classifies the note at the given absolute path.
//
// It reads the file content, checks for a frontmatter override, then tries the
// LLM classifier (using the cache if available), and finally falls back to the
// heuristic classifier.
func (c *Classifier) Classify(path string) (Classification, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Classification{}, fmt.Errorf("read %s: %w", path, err)
	}

	// 1. Frontmatter tag override.
	if cl, ok := classifyFromFrontmatter(content); ok {
		return cl, nil
	}

	// 2. LLM classifier (with cache).
	if c.llm != nil {
		if c.cache != nil && !c.opts.NoCache {
			if entry, ok := c.cache.Get(path, content, c.llm.model); ok {
				return classificationFromEntry(entry), nil
			}
		}

		cl, err := c.llm.classify(path, content)
		if err != nil {
			c.logf("LLM classify %s: %v; falling back to heuristic", filepath.Base(path), err)
		} else {
			if c.cache != nil {
				c.cache.Set(path, content, c.llm.model, entryFromClassification(cl))
				if saveErr := c.cache.Save(); saveErr != nil {
					c.logf("cache save: %v", saveErr)
				}
			}
			return cl, nil
		}
	}

	// 3. Heuristic fallback.
	r := c.heuristic.Classify(path, content)
	return Classification{
		Type:       ArtifactType(r.Type),
		Confidence: Confidence(r.Confidence),
		Title:      r.Title,
		Reason:     r.Reason,
		Source:     SourceHeuristic,
	}, nil
}

func (c *Classifier) logf(format string, args ...any) {
	if c.opts.Verbose != nil {
		c.opts.Verbose(fmt.Sprintf(format, args...))
	}
}
