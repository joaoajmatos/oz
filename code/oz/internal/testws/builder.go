// Package testws provides test infrastructure for building oz-compliant
// workspaces in temporary directories. Tests use the builder to create
// real, on-disk workspaces without depending on a fixture repository.
//
// Usage:
//
//	ws := testws.New(t).
//	    WithAgent("backend",
//	        testws.Scope("code/api/**"),
//	        testws.Role("Builds REST endpoints"),
//	    ).
//	    WithSpec("specs/api.md", testws.Section("overview", "...")).
//	    Build()
//
// The builder accepts any `testing.TB` (`*testing.T` or `*testing.B`) so benchmarks can
// scaffold workspaces. It calls scaffold.Scaffold internally so non-agent files
// (AGENTS.md, OZ.md, rules/, etc.) are always consistent with oz init output.
package testws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/scaffold"
	"github.com/joaoajmatos/oz/internal/semantic"
)

// Builder constructs oz-compliant workspaces for tests.
type Builder struct {
	tb        testing.TB
	agents    []agentDef
	specs     []fileDef
	decisions []fileDef
	contexts  []contextDef
	notes     []fileDef
	overlay   *SemanticOverlay
}

type agentDef struct {
	Name             string
	Scope            []string
	Role             string
	Responsibilities string
	OutOfScope       string
	ReadChain        []string
	Rules            []string
	Skills           []string
}

type fileDef struct {
	Path     string
	Sections []sectionDef
}

type sectionDef struct {
	Heading string
	Content string
}

type contextDef struct {
	Topic   string
	File    string
	Content string
}

// SemanticOverlay represents a pre-built context/semantic.json for testing
// the enriched query path without running oz context enrich.
type SemanticOverlay struct {
	GraphHash string          `json:"graph_hash"`
	Concepts  []OverlayConcept `json:"concepts"`
}

// OverlayConcept is a concept node in the semantic overlay.
type OverlayConcept struct {
	Name    string `json:"name"`
	OwnedBy string `json:"owned_by"`
}

// AgentOption configures an agent definition.
type AgentOption func(*agentDef)

// Scope sets the scope paths declared by the agent.
func Scope(paths ...string) AgentOption {
	return func(a *agentDef) { a.Scope = append(a.Scope, paths...) }
}

// Role sets the agent's role description.
func Role(text string) AgentOption {
	return func(a *agentDef) { a.Role = text }
}

// Responsibilities sets the agent's responsibilities text.
func Responsibilities(text string) AgentOption {
	return func(a *agentDef) { a.Responsibilities = text }
}

// OutOfScope sets the agent's out-of-scope declaration.
func OutOfScope(text string) AgentOption {
	return func(a *agentDef) { a.OutOfScope = text }
}

// ReadChain sets the files in the agent's read-chain.
func ReadChain(files ...string) AgentOption {
	return func(a *agentDef) { a.ReadChain = append(a.ReadChain, files...) }
}

// Rules sets the rule files that govern this agent.
func Rules(files ...string) AgentOption {
	return func(a *agentDef) { a.Rules = append(a.Rules, files...) }
}

// Skills sets the skills this agent is authorized to invoke.
func Skills(names ...string) AgentOption {
	return func(a *agentDef) { a.Skills = append(a.Skills, names...) }
}

// SpecOption configures a spec file.
type SpecOption func(*fileDef)

// Section adds a markdown section to a spec or doc file.
func Section(heading, content string) SpecOption {
	return func(f *fileDef) {
		f.Sections = append(f.Sections, sectionDef{heading, content})
	}
}

// New returns a new Builder bound to tb (*testing.T or *testing.B).
func New(tb testing.TB) *Builder {
	tb.Helper()
	return &Builder{tb: tb}
}

// WithAgent adds an agent to the workspace.
func (b *Builder) WithAgent(name string, opts ...AgentOption) *Builder {
	def := agentDef{Name: name}
	for _, o := range opts {
		o(&def)
	}
	b.agents = append(b.agents, def)
	return b
}

// WithSpec adds a spec file to the workspace.
func (b *Builder) WithSpec(path string, opts ...SpecOption) *Builder {
	def := fileDef{Path: path}
	for _, o := range opts {
		o(&def)
	}
	b.specs = append(b.specs, def)
	return b
}

// WithDecision adds a decision file under specs/decisions/.
func (b *Builder) WithDecision(id, content string) *Builder {
	b.decisions = append(b.decisions, fileDef{
		Path:     fmt.Sprintf("specs/decisions/%s.md", id),
		Sections: []sectionDef{{"Decision", content}},
	})
	return b
}

// WithContextSnapshot adds a context snapshot file under context/<topic>/.
func (b *Builder) WithContextSnapshot(topic, file, content string) *Builder {
	b.contexts = append(b.contexts, contextDef{topic, file, content})
	return b
}

// WithNote adds a note file under notes/.
func (b *Builder) WithNote(file, content string) *Builder {
	b.notes = append(b.notes, fileDef{
		Path:     fmt.Sprintf("notes/%s", file),
		Sections: []sectionDef{{"", content}},
	})
	return b
}

// WithSemanticOverlay sets a pre-built semantic overlay for the workspace.
func (b *Builder) WithSemanticOverlay(overlay SemanticOverlay) *Builder {
	b.overlay = &overlay
	return b
}

// Build materializes the workspace in tb.TempDir() and returns a handle to it.
// The directory is automatically removed when the test or benchmark ends.
func (b *Builder) Build() *Workspace {
	b.tb.Helper()

	dir := b.tb.TempDir()

	// Build agent configs for scaffold.
	agentCfgs := make([]scaffold.AgentConfig, len(b.agents))
	for i, a := range b.agents {
		agentCfgs[i] = scaffold.AgentConfig{
			Name:        a.Name,
			Description: a.Role,
			Type:        "coding", // use coding type so read-chain stubs are sensible
		}
	}

	name := "test-workspace"
	if len(b.agents) > 0 {
		name = b.agents[0].Name + "-workspace"
	}

	cfg := scaffold.Config{
		Name:        name,
		Description: "oz test workspace",
		CodeMode:    "inline",
		Agents:      agentCfgs,
	}

	if err := scaffold.Scaffold(dir, cfg); err != nil {
		b.tb.Fatalf("testws: scaffold failed: %v", err)
	}

	// Overwrite each AGENT.md with detailed content from the builder.
	for _, a := range b.agents {
		path := filepath.Join(dir, "agents", a.Name, "AGENT.md")
		if err := os.WriteFile(path, []byte(renderAgentMD(a)), 0644); err != nil {
			b.tb.Fatalf("testws: write AGENT.md for %s: %v", a.Name, err)
		}
	}

	// Write spec files.
	for _, s := range b.specs {
		if err := b.writeMarkdown(dir, s); err != nil {
			b.tb.Fatalf("testws: write spec %s: %v", s.Path, err)
		}
	}

	// Write decision files.
	for _, d := range b.decisions {
		if err := b.writeMarkdown(dir, d); err != nil {
			b.tb.Fatalf("testws: write decision %s: %v", d.Path, err)
		}
	}

	// Write context snapshots.
	for _, c := range b.contexts {
		path := filepath.Join(dir, "context", c.Topic, c.File)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			b.tb.Fatalf("testws: mkdir for context %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(c.Content), 0644); err != nil {
			b.tb.Fatalf("testws: write context %s: %v", path, err)
		}
	}

	// Write notes.
	for _, n := range b.notes {
		if err := b.writeMarkdown(dir, n); err != nil {
			b.tb.Fatalf("testws: write note %s: %v", n.Path, err)
		}
	}

	// Write semantic overlay if provided.
	if b.overlay != nil {
		if err := b.writeOverlay(dir, b.overlay); err != nil {
			b.tb.Fatalf("testws: write semantic overlay: %v", err)
		}
	}

	return &Workspace{tb: b.tb, path: dir}
}

// writeOverlay converts the simplified test overlay to the full semantic.json
// format and writes it to context/semantic.json.
func (b *Builder) writeOverlay(root string, o *SemanticOverlay) error {
	full := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		GraphHash:     o.GraphHash,
	}
	for _, c := range o.Concepts {
		conceptID := "concept:" + slugify(c.Name)
		agentNodeID := "agent:" + c.OwnedBy
		full.Concepts = append(full.Concepts, semantic.Concept{
			ID:         conceptID,
			Name:       c.Name,
			Tag:        semantic.TagExtracted,
			Confidence: 1.0,
		})
		full.Edges = append(full.Edges, semantic.ConceptEdge{
			From:       conceptID,
			To:         agentNodeID,
			Type:       semantic.EdgeTypeAgentOwnsConcept,
			Tag:        semantic.TagExtracted,
			Confidence: 1.0,
		})
	}
	return semantic.Write(root, full)
}

// slugify converts a concept name to a URL-friendly slug for use in concept IDs.
func slugify(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

// writeMarkdown writes a fileDef as a markdown file, creating parent dirs.
func (b *Builder) writeMarkdown(root string, f fileDef) error {
	path := filepath.Join(root, f.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(renderMarkdown(f)), 0644)
}

// renderAgentMD produces a complete, convention-compliant AGENT.md from an agentDef.
func renderAgentMD(a agentDef) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s Agent\n\n", a.Name)

	// Role
	role := a.Role
	if role == "" {
		role = "<!-- Describe what this agent is responsible for. -->"
	}
	fmt.Fprintf(&b, "## Role\n\n%s\n\n---\n\n", role)

	// Read-chain
	b.WriteString("## Read-chain\n\nRead these files in order before starting any task:\n\n")
	chain := a.ReadChain
	if len(chain) == 0 {
		chain = []string{"AGENTS.md — workspace entry point", "OZ.md — workspace manifest"}
	}
	for i, f := range chain {
		fmt.Fprintf(&b, "%d. `%s`\n", i+1, f)
	}
	b.WriteString("\n---\n\n")

	// Rules
	b.WriteString("## Rules\n\nThese files govern your behavior. Read them and follow them without exception:\n\n")
	rules := a.Rules
	if len(rules) == 0 {
		rules = []string{"rules/coding-guidelines.md — hard constraints for all code"}
	}
	for _, r := range rules {
		fmt.Fprintf(&b, "- `%s`\n", r)
	}
	b.WriteString("\n---\n\n")

	// Skills
	b.WriteString("## Skills\n\nYou are authorized to invoke these skills:\n\n")
	if len(a.Skills) == 0 {
		b.WriteString("<!-- No skills defined. -->\n")
	} else {
		for _, s := range a.Skills {
			fmt.Fprintf(&b, "- `%s`\n", s)
		}
	}
	b.WriteString("\n---\n\n")

	// Responsibilities — include scope paths as declarations
	b.WriteString("## Responsibilities\n\n")
	if a.Responsibilities != "" {
		fmt.Fprintf(&b, "%s\n\n", a.Responsibilities)
	}
	if len(a.Scope) > 0 {
		b.WriteString("Scope:\n\n")
		for _, s := range a.Scope {
			fmt.Fprintf(&b, "- `%s`\n", s)
		}
		b.WriteString("\n")
	}
	b.WriteString("---\n\n")

	// Out of scope
	b.WriteString("## Out of scope\n\n")
	if a.OutOfScope != "" {
		fmt.Fprintf(&b, "%s\n\n", a.OutOfScope)
	} else {
		b.WriteString("<!-- Nothing explicitly excluded. -->\n\n")
	}
	b.WriteString("---\n\n")

	// Context topics
	b.WriteString("## Context topics\n\n<!-- List context/<topic> entries this agent should read. -->\n")

	return b.String()
}

// renderMarkdown produces markdown content from a fileDef.
func renderMarkdown(f fileDef) string {
	var b strings.Builder
	for _, s := range f.Sections {
		if s.Heading != "" {
			fmt.Fprintf(&b, "## %s\n\n", s.Heading)
		}
		if s.Content != "" {
			fmt.Fprintf(&b, "%s\n\n", s.Content)
		}
	}
	return b.String()
}

// Workspace is a handle to a materialized test workspace.
type Workspace struct {
	tb   testing.TB
	path string
}

// Path returns the absolute path to the workspace root.
func (w *Workspace) Path() string { return w.path }

// Cleanup removes the workspace directory. Calling this is optional — the
// directory is automatically removed when the test ends via t.TempDir().
func (w *Workspace) Cleanup() {
	_ = os.RemoveAll(w.path)
}

// ReadFile reads a file relative to the workspace root.
func (w *Workspace) ReadFile(rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(w.path, rel))
}
