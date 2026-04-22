package classifier

import (
	"bufio"
	"bytes"
	"strings"
)

// classifyFromFrontmatter checks for a `crystallize:` key in the note's YAML
// frontmatter. If present, it returns a Classification with SourceFrontmatter
// and the explicit type, bypassing both LLM and heuristic classifiers.
//
// Supported frontmatter keys:
//
//	crystallize: <type>         — required; one of the five ArtifactType values
//	crystallize-title: <title>  — optional; suggested canonical title
func classifyFromFrontmatter(content []byte) (Classification, bool) {
	artifactType, title, ok := parseCrystallizeFrontmatter(content)
	if !ok {
		return Classification{}, false
	}
	t := normaliseType(artifactType)
	if t == TypeUnknown {
		return Classification{}, false
	}
	return Classification{
		Type:       t,
		Confidence: ConfidenceHigh,
		Title:      title,
		Reason:     "explicit crystallize tag in frontmatter",
		Source:     SourceFrontmatter,
	}, true
}

// parseCrystallizeFrontmatter extracts crystallize fields from YAML frontmatter.
// It uses line-by-line scanning (same pattern as workspace.go) rather than a
// full YAML parser to avoid a dependency on the note being well-formed YAML.
func parseCrystallizeFrontmatter(content []byte) (artifactType, title string, ok bool) {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// Frontmatter must start on the first line.
	if !scanner.Scan() {
		return "", "", false
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return "", "", false
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		if v, found := parseField(line, "crystallize"); found {
			artifactType = v
		}
		if v, found := parseField(line, "crystallize-title"); found {
			title = v
		}
	}

	return artifactType, title, artifactType != ""
}

// parseField extracts the value from a `key: value` frontmatter line.
func parseField(line, key string) (string, bool) {
	prefix := key + ":"
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	return strings.TrimSpace(line[len(prefix):]), true
}
