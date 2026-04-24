package context

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joaoajmatos/oz/internal/graph"
)

// ParsedAgent holds all fields extracted from an AGENT.md file.
// All fields are optional — the parser is tolerant and returns what it finds.
type ParsedAgent struct {
	Name              string
	Role              string
	ReadChain         []string
	Rules             []string
	Skills            []string
	Responsibilities  string
	Scope             []string // extracted from Responsibilities bullet items
	OutOfScope        string
	SkillsBody        string // full Skills section (prose + lists; for routing)
	ContextTopics     []string
	ContextTopicsBody string // full Context topics section (for routing)
}

// ParsedSection is a single heading + content block from a markdown file.
type ParsedSection struct {
	Heading string
	Content string
}

// ParseAgentMD reads the AGENT.md at absPath and returns structured fields.
// agentName is the directory name under agents/ (used as the node name).
func ParseAgentMD(absPath, agentName string) (*ParsedAgent, error) {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	sections := splitSections(string(content))

	a := &ParsedAgent{Name: agentName}

	for _, sec := range sections {
		switch normaliseHeading(sec.Heading) {
		case "role":
			a.Role = strings.TrimSpace(sec.Content)

		case "read-chain", "readchain", "read chain":
			a.ReadChain = extractListItems(sec.Content)

		case "rules":
			a.Rules = extractBacktickPaths(sec.Content)

		case "skills":
			a.Skills = extractBacktickPaths(sec.Content)
			a.SkillsBody = strings.TrimSpace(sec.Content)

		case "responsibilities":
			a.Responsibilities, a.Scope = parseResponsibilities(sec.Content)

		case "out of scope", "out-of-scope":
			a.OutOfScope = strings.TrimSpace(sec.Content)

		case "context topics", "context-topics":
			a.ContextTopics = extractListItems(sec.Content)
			a.ContextTopicsBody = strings.TrimSpace(sec.Content)
		}
	}

	return a, nil
}

// ParseMarkdownSections reads a markdown file and returns its H2 sections.
func ParseMarkdownSections(absPath string) ([]ParsedSection, error) {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	return splitSections(string(content)), nil
}

// AgentNode converts a ParsedAgent into a graph.Node.
// file is the path relative to the workspace root.
func AgentNode(a *ParsedAgent, file string) graph.Node {
	id := "agent:" + a.Name
	return graph.Node{
		ID:                id,
		Type:              graph.NodeTypeAgent,
		File:              file,
		Name:              a.Name,
		Role:              a.Role,
		Scope:             nilIfEmpty(a.Scope),
		Responsibilities:  a.Responsibilities,
		OutOfScope:        a.OutOfScope,
		ReadChain:         nilIfEmpty(a.ReadChain),
		Rules:             nilIfEmpty(a.Rules),
		Skills:            nilIfEmpty(a.Skills),
		SkillsBody:        a.SkillsBody,
		ContextTopics:     nilIfEmpty(a.ContextTopics),
		ContextTopicsBody: a.ContextTopicsBody,
	}
}

// SectionNodeID returns the stable ID for a section node.
// nodeType is NodeTypeSpecSection or NodeTypeDoc.
func SectionNodeID(nodeType, file, heading string) string {
	return nodeType + ":" + file + ":" + heading
}

// DecisionNodeID returns the stable ID for a decision node.
func DecisionNodeID(file string) string {
	// Strip directory prefix and .md suffix for the discriminator.
	base := filepath.Base(file)
	base = strings.TrimSuffix(base, ".md")
	return "decision:" + base
}

// ContextSnapshotNodeID returns the stable ID for a context snapshot node.
func ContextSnapshotNodeID(file string) string {
	return "context_snapshot:" + file
}

// NoteNodeID returns the stable ID for a note node.
func NoteNodeID(file string) string {
	return "note:" + file
}

// --- helpers -----------------------------------------------------------------

// splitSections splits markdown content into H2 sections.
// The text before the first H2 is discarded (it's typically the H1 title).
// Horizontal rule lines ("---") that act as section separators are stripped.
func splitSections(content string) []ParsedSection {
	var sections []ParsedSection
	var current *ParsedSection
	var body strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "## ") {
			if current != nil {
				current.Content = strings.TrimSpace(body.String())
				sections = append(sections, *current)
			}
			heading := strings.TrimPrefix(line, "## ")
			heading = strings.TrimSpace(heading)
			current = &ParsedSection{Heading: heading}
			body.Reset()
			continue
		}

		// Skip horizontal rules used as section separators.
		if strings.TrimSpace(line) == "---" {
			continue
		}

		if current != nil {
			body.WriteString(line)
			body.WriteByte('\n')
		}
	}

	if current != nil {
		current.Content = strings.TrimSpace(body.String())
		sections = append(sections, *current)
	}

	return sections
}

// normaliseHeading lowercases a heading and trims surrounding whitespace.
func normaliseHeading(h string) string {
	return strings.ToLower(strings.TrimSpace(h))
}

// extractListItems parses numbered and bulleted list items from markdown content.
// Returns the cleaned item text (backtick wrappers stripped, trailing annotations stripped).
func extractListItems(content string) []string {
	var items []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	// Matches: "1. text", "- text", "* text"
	re := regexp.MustCompile(`^(?:\d+\.\s+|[-*]\s+)(.+)$`)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		item := m[1]
		// Strip leading backtick-wrapped path and trailing annotation.
		// e.g. "`AGENTS.md` — workspace entry point" → "AGENTS.md"
		if strings.HasPrefix(item, "`") {
			end := strings.Index(item[1:], "`")
			if end >= 0 {
				item = item[1 : end+1]
			}
		}
		// Strip trailing " — ..." annotations.
		if idx := strings.Index(item, " — "); idx >= 0 {
			item = item[:idx]
		}
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

// backtickPathRe matches a backtick-wrapped path in markdown.
// Paths must contain at least one / or . to avoid matching inline code words.
var backtickPathRe = regexp.MustCompile("`([^`]+[/.].[^`]*)`")

// extractBacktickPaths extracts file paths wrapped in backticks from content.
func extractBacktickPaths(content string) []string {
	var paths []string
	seen := map[string]bool{}
	for _, m := range backtickPathRe.FindAllStringSubmatch(content, -1) {
		p := strings.TrimSpace(m[1])
		if p != "" && !seen[p] {
			paths = append(paths, p)
			seen[p] = true
		}
	}
	return paths
}

// parseResponsibilities splits the Responsibilities section into the prose
// description and extracted scope paths. Scope paths are backtick-wrapped
// items under a "Scope:" sub-heading or in a bullet list.
func parseResponsibilities(content string) (prose string, scope []string) {
	var proseLines []string
	var inScope bool
	scanner := bufio.NewScanner(strings.NewReader(content))
	scopeRe := regexp.MustCompile("^[-*]\\s+`([^`]+)`")

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.EqualFold(trimmed, "scope:") || strings.EqualFold(trimmed, "**scope:**") {
			inScope = true
			continue
		}

		if inScope {
			// Skip blank lines within the scope block.
			if trimmed == "" {
				continue
			}
			if m := scopeRe.FindStringSubmatch(trimmed); m != nil {
				scope = append(scope, m[1])
				continue
			}
			// Non-blank, non-list line — scope block ended.
			inScope = false
		}

		proseLines = append(proseLines, line)
	}

	prose = strings.TrimSpace(strings.Join(proseLines, "\n"))
	return prose, scope
}

// nilIfEmpty returns nil when the slice has no elements, otherwise returns s.
// This prevents empty JSON arrays in serialized output.
func nilIfEmpty(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}
