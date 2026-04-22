package heuristic_test

import (
	"testing"

	"github.com/joaoajmatos/oz/internal/crystallize/classifier/heuristic"
)

func TestClassify(t *testing.T) {
	c := heuristic.New()

	cases := []struct {
		name           string
		path           string
		content        string
		wantType       string
		wantConfidence string
	}{
		// ADR positives
		{
			name:           "adr: classic we-decided language",
			path:           "notes/sprint-3.md",
			content:        "We decided to use BM25F for query routing because it handles multi-field scoring naturally.\n\n## Alternatives considered\n\nWe rejected TF-IDF due to lack of IDF floor support.",
			wantType:       "adr",
			wantConfidence: "high",
		},
		{
			name:           "adr: status accepted frontmatter",
			path:           "notes/auth.md",
			content:        "---\nstatus: accepted\n---\n\n## Context\nWe chose JWT over session cookies.\n\n## Decision\nUse JWT.\n\n## Consequences\nStateless auth.",
			wantType:       "adr",
			wantConfidence: "high",
		},
		{
			name:           "adr: rejected alternatives",
			path:           "notes/db.md",
			content:        "We rejected Postgres in favour of SQLite for local-first storage. The tradeoff is acceptable.",
			wantType:       "adr",
			wantConfidence: "high",
		},
		// ADR negatives
		{
			name:           "adr: pure guide is not adr",
			path:           "notes/setup.md",
			content:        "## Prerequisites\n\nStep 1: Install Go\n\n```sh\nbrew install go\n```\n\nStep 2: Clone the repo.",
			wantType:       "guide",
			wantConfidence: "high",
		},

		// Spec positives
		{
			name:           "spec: RFC 2119 language",
			path:           "notes/convention.md",
			content:        "All agents MUST read the AGENT.md file before starting. The workspace MUST contain an AGENTS.md root file. Agents SHOULD follow the read-chain.",
			wantType:       "spec",
			wantConfidence: "high",
		},
		{
			name:           "spec: requirements section",
			path:           "notes/req.md",
			content:        "## Requirements\n\n1. The workspace must include a specs/ directory.\n2. Every agent shall have a defined read-chain.\n3. Convention files must be valid YAML.",
			wantType:       "spec",
			wantConfidence: "high",
		},
		{
			name:           "spec: convention keyword",
			path:           "notes/conv.md",
			content:        "convention: agents must declare their scope. Required: AGENTS.md present at root. All workspaces must be valid according to oz validate.",
			wantType:       "spec",
			wantConfidence: "high",
		},
		// Spec negatives
		{
			name:           "spec: guide not spec",
			path:           "notes/howto.md",
			content:        "How to set up your workspace:\n\nStep 1: Run oz init.\nStep 2: Configure agents.\n\n```sh\noz init\n```",
			wantType:       "guide",
			wantConfidence: "high",
		},

		// Guide positives
		{
			name:           "guide: numbered steps with shell",
			path:           "notes/setup-cursor.md",
			content:        "# Setup Cursor Integration\n\n## Prerequisites\n\nInstall oz first.\n\n## Steps\n\n1. Run `oz add cursor`\n2. Open Cursor\n3. Configure the MCP server\n\n```sh\noz add cursor\n```",
			wantType:       "guide",
			wantConfidence: "high",
		},
		{
			name:           "guide: how to prefix",
			path:           "notes/guide.md",
			content:        "How to use oz crystallize:\n\nStep 1: Run oz crystallize --dry-run to preview.\nStep 2: Review the classifications.\nStep 3: Accept or skip each file.",
			wantType:       "guide",
			wantConfidence: "high",
		},
		// Guide dominant: has guide signals but also one "we decided" — guide wins
		// at medium confidence (score 4, below the high threshold of 6).
		{
			name:           "guide dominant with weak adr signal",
			path:           "notes/mixed.md",
			content:        "We decided to try this. Step 1: do the thing. Step 2: check it worked.",
			wantType:       "guide",
			wantConfidence: "medium",
		},

		// Arch positives
		{
			name:           "arch: architecture heading",
			path:           "notes/graph.md",
			content:        "# Graph Architecture\n\n## Overview\n\nThe context system has three layers: convention, structural graph, and semantic overlay.\n\n## Components\n\n- graph.json: the structural graph\n- semantic.json: the LLM overlay",
			wantType:       "arch",
			wantConfidence: "high",
		},
		{
			name:           "arch: mermaid diagram",
			path:           "notes/flow.md",
			content:        "## Data Flow\n\n```mermaid\ngraph TD\n  A[notes/] --> B[classifier]\n  B --> C[promote]\n```\n\nThe pipeline routes each note through the classifier component.",
			wantType:       "arch",
			wantConfidence: "high",
		},

		// Open-item positives
		{
			name:           "open-item: TODO list",
			path:           "notes/todos.md",
			content:        "## Open Questions\n\n- [ ] Should we support tree-sitter in V1?\n- [ ] TBD: how do we handle open-items.md becoming too large?\n- [ ] Known issue: cache not invalidated on model change.",
			wantType:       "open-item",
			wantConfidence: "high",
		},
		{
			name:           "open-item: short TODO file",
			path:           "notes/issues.md",
			content:        "TODO: fix the ADR numbering collision edge case.\nFIXME: crystallize-cache.json not written on SIGKILL.\nTBD: should --accept-all require --force for medium confidence?",
			wantType:       "open-item",
			wantConfidence: "high",
		},

		// Unknown
		{
			name:           "unknown: too short",
			path:           "notes/scratch.md",
			content:        "quick note: look into this later",
			wantType:       "unknown",
			wantConfidence: "low",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.Classify(tc.path, []byte(tc.content))
			if got.Type != tc.wantType {
				t.Errorf("type = %q, want %q (scores: %v)", got.Type, tc.wantType, got.Scores)
			}
			if got.Confidence != tc.wantConfidence {
				t.Errorf("confidence = %q, want %q", got.Confidence, tc.wantConfidence)
			}
		})
	}
}

func TestTitleFromClassify(t *testing.T) {
	c := heuristic.New()
	r := c.Classify("notes/auth-redesign.md", []byte("We decided to rewrite auth. Status: accepted."))
	if r.Title != "Auth Redesign" {
		t.Errorf("title = %q, want %q", r.Title, "Auth Redesign")
	}
}
