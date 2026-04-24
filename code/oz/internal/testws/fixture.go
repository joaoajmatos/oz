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
	CodePackages    []CodePackageFixture    `yaml:"code_packages,omitempty"`
	SemanticOverlay *SemanticOverlayFixture `yaml:"semantic_overlay,omitempty"`
}

// SemanticOverlayFixture is the YAML schema for a pre-built semantic overlay.
// It is a simplified representation: each concept is listed with its owning agent.
// The fixture loader calls WithSemanticOverlay, which writes a full semantic.json.
type SemanticOverlayFixture struct {
	Concepts   []OverlayConcept          `yaml:"concepts"`
	Implements []OverlayImplementsFixture `yaml:"implements,omitempty"`
}

type OverlayImplementsFixture struct {
	Concept  string `yaml:"concept"`
	Package  string `yaml:"package"`
	Reviewed *bool  `yaml:"reviewed,omitempty"`
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

type CodePackageFixture struct {
	Path    string               `yaml:"path"`
	Import  string               `yaml:"import"`
	Doc     string               `yaml:"doc"`
	Symbols []CodeSymbolFixture  `yaml:"symbols"`
}

type CodeSymbolFixture struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	File string `yaml:"file"`
	Line int    `yaml:"line"`
	Doc  string `yaml:"doc"`
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

	for _, pkg := range fix.CodePackages {
		symbols := make([]codeSymbolDef, 0, len(pkg.Symbols))
		for _, s := range pkg.Symbols {
			symbols = append(symbols, codeSymbolDef{
				Name: s.Name,
				Kind: s.Kind,
				File: s.File,
				Line: s.Line,
				Doc:  s.Doc,
			})
		}
		b.WithCodePackage(pkg.Path, pkg.Import, pkg.Doc, symbols)
	}

	if fix.SemanticOverlay != nil {
		implements := make([]OverlayImplements, 0, len(fix.SemanticOverlay.Implements))
		for _, e := range fix.SemanticOverlay.Implements {
			implements = append(implements, OverlayImplements{
				Concept:  e.Concept,
				Package:  e.Package,
				Reviewed: e.Reviewed,
			})
		}
		b.WithSemanticOverlay(SemanticOverlay{
			Concepts: fix.SemanticOverlay.Concepts,
			Implements: implements,
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

// QueryCase is a single query with its expected routing and retrieval
// outcomes. Routing fields (ExpectedAgent, MinConfidence, ExpectedCandidates,
// Reason) are enforced by TestRoutingAccuracy / QueryCase.Matches. The
// Expect* retrieval fields are enforced by the retrieval matchers in
// assertions.go — they are only read once the corresponding R-requirement
// from the retrieval PRD has shipped.
type QueryCase struct {
	Query              string   `yaml:"query"`
	ExpectedAgent      string   `yaml:"expected_agent"`
	MinConfidence      float64  `yaml:"min_confidence"`
	ExpectedCandidates []string `yaml:"expected_candidates"`
	Reason             string   `yaml:"reason"`

	// Retrieval expectations (drafted in Sprint 1; activated Sprints 2–4).
	ExpectBlocksInTopK          []BlockExpectation          `yaml:"expect_blocks_in_topk,omitempty"`
	ExpectCodeEntryPointsInTopK []CodeEntryPointExpectation `yaml:"expect_code_entry_points_in_topk,omitempty"`
	ExpectPackagesInTopK        []PackageExpectation        `yaml:"expect_packages_in_topk,omitempty"`
	ExpectRelevanceDescending   bool                        `yaml:"expect_relevance_descending,omitempty"`
	ExpectNoRelevantContext     bool                        `yaml:"expect_no_relevant_context,omitempty"`
	ExpectTrustBeats            *TrustBeatsExpectation      `yaml:"expect_trust_beats,omitempty"`
}

// BlockExpectation asserts that a context_block matching (File, Section) is
// present in the top K returned blocks. An empty Section matches any section
// under File.
type BlockExpectation struct {
	File    string `yaml:"file"`
	Section string `yaml:"section"`
	K       int    `yaml:"k"`
}

// CodeEntryPointExpectation asserts that a code_entry_point with the given
// symbol name is present in the top K. Symbol may be bare ("Run") or
// qualified ("drift.Run").
type CodeEntryPointExpectation struct {
	Symbol string `yaml:"symbol"`
	K      int    `yaml:"k"`
}

// PackageExpectation asserts that the given package path is in the top K
// implementing_packages.
type PackageExpectation struct {
	Package string `yaml:"package"`
	K       int    `yaml:"k"`
}

// TrustBeatsExpectation asserts that at least one block at WinnerTier appears
// at a higher rank than every block at LoserTier — used for cross-tier
// queries where specs must outrank notes on ties.
type TrustBeatsExpectation struct {
	WinnerTier string `yaml:"winner_tier"`
	LoserTier  string `yaml:"loser_tier"`
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
