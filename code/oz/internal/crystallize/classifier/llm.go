package classifier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaoajmatos/oz/internal/openrouter"
)

// llmClassifier classifies notes using OpenRouter with workspace-aware context.
type llmClassifier struct {
	client  *openrouter.Client
	model   string
	context string // pre-built workspace context block, reused across calls
}

// newLLMClassifier creates an llmClassifier. Returns an error if the OpenRouter
// client cannot be initialised (e.g. missing API key).
func newLLMClassifier(workspaceRoot, model string) (*llmClassifier, error) {
	client, err := openrouter.New(model)
	if err != nil {
		return nil, err
	}
	ctx, err := buildWorkspaceContext(workspaceRoot)
	if err != nil {
		// Non-fatal: proceed with minimal context.
		ctx = minimalContext
	}
	return &llmClassifier{
		client:  client,
		model:   client.Model,
		context: ctx,
	}, nil
}

// classify sends the note content to the LLM and returns a Classification.
// It retries once on JSON parse failure before returning an error.
func (l *llmClassifier) classify(path string, content []byte) (Classification, error) {
	prompt := buildClassifyPrompt(l.context, filepath.Base(path), string(content))
	msgs := []openrouter.Message{{Role: "user", Content: prompt}}

	var lastErr error
	for attempt := range 2 {
		resp, err := l.client.Complete(msgs)
		if err != nil {
			return Classification{}, fmt.Errorf("openrouter: %w", err)
		}
		raw := resp.Choices[0].Message.Content
		cl, err := parseClassifyResponse(raw)
		if err == nil {
			return cl, nil
		}
		lastErr = err
		// On first failure, add a correction message and retry.
		if attempt == 0 {
			msgs = append(msgs,
				openrouter.Message{Role: "assistant", Content: raw},
				openrouter.Message{Role: "user", Content: retryPrompt},
			)
		}
	}
	return Classification{}, fmt.Errorf("parse LLM response: %w", lastErr)
}

// llmResponse is the JSON structure the LLM is asked to return.
type llmResponse struct {
	Type       string `json:"type"`
	Confidence string `json:"confidence"`
	Title      string `json:"title"`
	Reason     string `json:"reason"`
}

func parseClassifyResponse(raw string) (Classification, error) {
	raw = strings.TrimSpace(raw)
	// Strip markdown code fences if the LLM wrapped its JSON.
	if strings.HasPrefix(raw, "```") {
		if idx := strings.Index(raw, "\n"); idx != -1 {
			raw = raw[idx+1:]
		}
		if idx := strings.LastIndex(raw, "```"); idx != -1 {
			raw = raw[:idx]
		}
		raw = strings.TrimSpace(raw)
	}

	var r llmResponse
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return Classification{}, fmt.Errorf("JSON unmarshal: %w", err)
	}

	cl := Classification{
		Type:       normaliseType(r.Type),
		Confidence: normaliseConfidence(r.Confidence),
		Title:      strings.TrimSpace(r.Title),
		Reason:     strings.TrimSpace(r.Reason),
		Source:     SourceLLM,
	}
	if cl.Title == "" {
		return Classification{}, fmt.Errorf("missing title in LLM response")
	}
	return cl, nil
}

func normaliseType(s string) ArtifactType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "adr":
		return TypeADR
	case "spec":
		return TypeSpec
	case "guide":
		return TypeGuide
	case "arch":
		return TypeArch
	case "open-item", "open_item", "openitem":
		return TypeOpenItem
	default:
		return TypeUnknown
	}
}

func normaliseConfidence(s string) Confidence {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "high":
		return ConfidenceHigh
	case "medium":
		return ConfidenceMedium
	default:
		return ConfidenceLow
	}
}

// buildWorkspaceContext assembles the few-shot context block by sampling the
// workspace's own existing artifacts (one ADR, one spec section, one guide).
func buildWorkspaceContext(root string) (string, error) {
	var sb strings.Builder
	sb.WriteString(workspaceContextHeader)

	examples := []struct {
		label string
		glob  string
	}{
		{"ADR example", filepath.Join(root, "specs", "decisions", "0001-*.md")},
		{"Spec example", filepath.Join(root, "specs", "oz-project-specification.md")},
		{"Guide example", filepath.Join(root, "docs", "guides", "*.md")},
	}

	for _, ex := range examples {
		matches, _ := filepath.Glob(ex.glob)
		if len(matches) == 0 {
			continue
		}
		content, err := os.ReadFile(matches[0])
		if err != nil {
			continue
		}
		lines := strings.SplitN(string(content), "\n", 32)
		if len(lines) > 30 {
			lines = lines[:30]
		}
		fmt.Fprintf(&sb, "\n### %s (%s)\n```\n%s\n```\n", ex.label, filepath.Base(matches[0]), strings.Join(lines, "\n"))
	}

	return sb.String(), nil
}

func buildClassifyPrompt(wsContext, filename, content string) string {
	// Truncate very large notes to avoid token waste; 200 lines is ample.
	lines := strings.SplitN(content, "\n", 202)
	if len(lines) > 200 {
		lines = append(lines[:200], "... [truncated]")
	}
	truncated := strings.Join(lines, "\n")

	return fmt.Sprintf(`%s

---

## Task

Classify the following note into exactly one of these artifact types:

| type | target path | when |
|------|-------------|------|
| adr | specs/decisions/NNNN-<slug>.md | captures a decision and its rationale |
| spec | specs/<slug>.md | normative convention text (MUST/SHOULD language) |
| guide | docs/guides/<slug>.md | step-by-step how-to with numbered steps or shell commands |
| arch | docs/<slug>.md | system or component design explanation |
| open-item | appended to docs/open-items.md | known issue, open question, or TODO list |
| unknown | — | cannot determine type with confidence |

Return ONLY a valid JSON object — no markdown fences, no explanation outside the JSON.

Schema:
{
  "type": "adr | spec | guide | arch | open-item | unknown",
  "confidence": "high | medium | low",
  "title": "<suggested canonical title for the promoted artifact>",
  "reason": "<one sentence explaining the classification>"
}

Rules:
- confidence "high": clear, unambiguous match to one type
- confidence "medium": probable match but some signals point elsewhere
- confidence "low": weak signals, type is a best guess
- If the note is too short or too vague to classify, use type "unknown" with confidence "low"
- title must not include the type name (e.g. not "ADR: foo", just "foo")

## Note to classify

Filename: %s

%s`, wsContext, filename, truncated)
}

const workspaceContextHeader = `You are classifying notes in an oz workspace.

An oz workspace enforces a source-of-truth hierarchy:
  specs/ (highest trust) > docs/ > context/ > notes/ (lowest trust)

The goal of oz crystallize is to promote raw notes from notes/ into canonical
artifacts at the correct location. Below are examples of existing artifacts in
this workspace so you understand its conventions.`

const minimalContext = workspaceContextHeader

const retryPrompt = `Your response could not be parsed as valid JSON. ` +
	`Return ONLY the JSON object with keys "type", "confidence", "title", and "reason". ` +
	`No markdown fences, no extra text.`
