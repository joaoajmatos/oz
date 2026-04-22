// Package heuristic implements a keyword + structure signal scoring classifier
// for oz crystallize. It is the fallback when the LLM classifier is unavailable
// (no API key, --no-enrich, or LLM call failure).
//
// Scoring model:
//
//	score(type) = Σ strong_hits×3 + Σ structure_hits×2 + Σ supporting_hits×1 − Σ anti_hits×2
//
// Thresholds:
//
//	score < 4              → TypeUnknown (too weak to classify)
//	top − second_best < 2  → ambiguous, confidence Medium
//	otherwise              → confidence High (score ≥ 6) or Medium (score 4–5)
package heuristic

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joaoajmatos/oz/internal/crystallize/classifier/signals"
)

// Result is the heuristic classifier's output for a single note.
type Result struct {
	Type       string
	Confidence string
	Title      string
	Reason     string
	Scores     map[string]int // exposed for verbose output
}

// Classifier scores notes against signal tables for all five artifact types.
type Classifier struct{}

// New returns a Classifier.
func New() *Classifier { return &Classifier{} }

// Classify scores the note content and returns a Result.
func (c *Classifier) Classify(path string, content []byte) Result {
	text := strings.ToLower(string(content))
	scores := scoreAll(text)

	best, secondScore := topTwo(scores)
	bestScore := scores[best]

	title := titleFromPath(path)

	switch {
	case bestScore < 4:
		return Result{
			Type:       "unknown",
			Confidence: "low",
			Title:      title,
			Reason:     "no strong signals found",
			Scores:     scores,
		}
	case bestScore-secondScore < 2:
		return Result{
			Type:       best,
			Confidence: "medium",
			Title:      title,
			Reason:     "ambiguous — signals match multiple types",
			Scores:     scores,
		}
	case bestScore >= 6:
		return Result{
			Type:       best,
			Confidence: "high",
			Title:      title,
			Reason:     signals.TopReason(best, text),
			Scores:     scores,
		}
	default:
		return Result{
			Type:       best,
			Confidence: "medium",
			Title:      title,
			Reason:     signals.TopReason(best, text),
			Scores:     scores,
		}
	}
}

// scoreAll scores the text against every artifact type's signal table.
func scoreAll(text string) map[string]int {
	out := make(map[string]int, len(signals.Types))
	for _, t := range signals.Types {
		out[t] = scoreType(text, t)
	}
	return out
}

// scoreType computes the score for a single artifact type.
func scoreType(text, artifactType string) int {
	table := signals.For(artifactType)
	score := 0
	for _, p := range table.Strong {
		if matchesPattern(text, p) {
			score += 3
		}
	}
	for _, p := range table.Structure {
		if matchesPattern(text, p) {
			score += 2
		}
	}
	for _, p := range table.Supporting {
		if matchesPattern(text, p) {
			score += 1
		}
	}
	for _, p := range table.Anti {
		if matchesPattern(text, p) {
			score -= 2
		}
	}
	return score
}

// matchesPattern reports whether the (lowercased) text matches the pattern.
// Patterns starting and ending with "/" are treated as regular expressions.
func matchesPattern(text, pattern string) bool {
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") {
		re, err := regexp.Compile(pattern[1 : len(pattern)-1])
		if err != nil {
			return false
		}
		return re.MatchString(text)
	}
	return strings.Contains(text, pattern)
}

// topTwo returns the best type name and the score of the second-best type.
func topTwo(scores map[string]int) (best string, secondScore int) {
	bestScore := -999
	secondScore = -999
	for t, s := range scores {
		switch {
		case s > bestScore:
			secondScore = bestScore
			bestScore = s
			best = t
		case s > secondScore:
			secondScore = s
		}
	}
	return best, secondScore
}

// titleFromPath converts a filename into a human-readable title suggestion.
// e.g. "auth-redesign.md" → "Auth Redesign"
func titleFromPath(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	parts := strings.FieldsFunc(base, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
