package testws

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// WorkspaceFixture is the YAML schema for a declarative workspace definition.
type WorkspaceFixture struct {
	Agents          []AgentFixture          `yaml:"agents"`
	Specs           []FileFixture           `yaml:"specs"`
	Decisions       []DecisionFixture       `yaml:"decisions"`
	Contexts        []ContextFixture        `yaml:"contexts"`
	Notes           []NoteFixture           `yaml:"notes"`
	SemanticOverlay *SemanticOverlayFixture `yaml:"semantic_overlay,omitempty"`
}

// SemanticOverlayFixture is the YAML schema for a pre-built semantic overlay.
// It is a simplified representation: each concept is listed with its owning agent.
// The fixture loader calls WithSemanticOverlay, which writes a full semantic.json.
type SemanticOverlayFixture struct {
	Concepts []OverlayConcept `yaml:"concepts"`
}

// AgentFixture is the YAML schema for a single agent definition.
type AgentFixture struct {
	Name             string   `yaml:"name"`
	Role             string   `yaml:"role"`
	Scope            []string `yaml:"scope"`
	Responsibilities string   `yaml:"responsibilities"`
	OutOfScope       string   `yaml:"out_of_scope"`
	ReadChain        []string `yaml:"readchain"`
	Rules            []string `yaml:"rules"`
	Skills           []string `yaml:"skills"`
}

// FileFixture is the YAML schema for a spec or doc file with sections.
type FileFixture struct {
	Path     string          `yaml:"path"`
	Sections []SectionFixture `yaml:"sections"`
}

// SectionFixture is a single heading+content pair.
type SectionFixture struct {
	Heading string `yaml:"heading"`
	Content string `yaml:"content"`
}

// DecisionFixture is a decision record under specs/decisions/.
type DecisionFixture struct {
	ID      string `yaml:"id"`
	Content string `yaml:"content"`
}

// ContextFixture is a context snapshot under context/<topic>/.
type ContextFixture struct {
	Topic   string `yaml:"topic"`
	File    string `yaml:"file"`
	Content string `yaml:"content"`
}

// NoteFixture is a file under notes/.
type NoteFixture struct {
	File    string `yaml:"file"`
	Content string `yaml:"content"`
}

// FromFixture loads a workspace.yaml file and returns a Builder configured
// from its contents.
func FromFixture(t *testing.T, path string) *Builder {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("testws: read fixture %s: %v", path, err)
	}

	var fix WorkspaceFixture
	if err := yaml.Unmarshal(data, &fix); err != nil {
		t.Fatalf("testws: parse fixture %s: %v", path, err)
	}

	b := New(t)

	for _, a := range fix.Agents {
		opts := []AgentOption{
			Role(a.Role),
			Scope(a.Scope...),
			Responsibilities(a.Responsibilities),
			OutOfScope(a.OutOfScope),
		}
		if len(a.ReadChain) > 0 {
			opts = append(opts, ReadChain(a.ReadChain...))
		}
		if len(a.Rules) > 0 {
			opts = append(opts, Rules(a.Rules...))
		}
		if len(a.Skills) > 0 {
			opts = append(opts, Skills(a.Skills...))
		}
		b.WithAgent(a.Name, opts...)
	}

	for _, s := range fix.Specs {
		var opts []SpecOption
		for _, sec := range s.Sections {
			opts = append(opts, Section(sec.Heading, sec.Content))
		}
		b.WithSpec(s.Path, opts...)
	}

	for _, d := range fix.Decisions {
		b.WithDecision(d.ID, d.Content)
	}

	for _, c := range fix.Contexts {
		b.WithContextSnapshot(c.Topic, c.File, c.Content)
	}

	for _, n := range fix.Notes {
		b.WithNote(n.File, n.Content)
	}

	if fix.SemanticOverlay != nil {
		b.WithSemanticOverlay(SemanticOverlay{
			Concepts: fix.SemanticOverlay.Concepts,
		})
	}

	return b
}

// GoldenSuite is a complete test suite loaded from a golden directory.
type GoldenSuite struct {
	Name        string
	MinAccuracy float64
	Queries     []QueryCase
	builder     *Builder
}

// Build materializes the workspace for this suite.
func (s *GoldenSuite) Build(t *testing.T) *Workspace {
	t.Helper()
	return s.builder.Build()
}

// QueryCase is a single routing query with expected outcome.
type QueryCase struct {
	Query              string   `yaml:"query"`
	ExpectedAgent      string   `yaml:"expected_agent"`
	MinConfidence      float64  `yaml:"min_confidence"`
	ExpectedCandidates []string `yaml:"expected_candidates"`
	Reason             string   `yaml:"reason"`
}

// QueriesFile is the YAML schema for a queries.yaml file.
type QueriesFile struct {
	MinAccuracy float64     `yaml:"min_accuracy"`
	Queries     []QueryCase `yaml:"queries"`
}

// LoadGoldenSuites loads all golden suites from a directory.
// Each subdirectory of dir containing workspace.yaml and queries.yaml
// is loaded as a suite.
func LoadGoldenSuites(t *testing.T, dir string) ([]*GoldenSuite, error) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read golden dir %s: %w", dir, err)
	}

	var suites []*GoldenSuite
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		suite, err := loadGoldenSuite(t, filepath.Join(dir, e.Name()), e.Name())
		if err != nil {
			return nil, fmt.Errorf("load suite %s: %w", e.Name(), err)
		}
		suites = append(suites, suite)
	}
	return suites, nil
}

func loadGoldenSuite(t *testing.T, dir, name string) (*GoldenSuite, error) {
	t.Helper()

	wsPath := filepath.Join(dir, "workspace.yaml")
	qPath := filepath.Join(dir, "queries.yaml")

	builder := FromFixture(t, wsPath)

	qData, err := os.ReadFile(qPath)
	if err != nil {
		return nil, fmt.Errorf("read queries.yaml: %w", err)
	}

	var qf QueriesFile
	if err := yaml.Unmarshal(qData, &qf); err != nil {
		return nil, fmt.Errorf("parse queries.yaml: %w", err)
	}

	return &GoldenSuite{
		Name:        name,
		MinAccuracy: qf.MinAccuracy,
		Queries:     qf.Queries,
		builder:     builder,
	}, nil
}
