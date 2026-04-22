// Package signals defines the keyword and structure signal tables used by the
// heuristic classifier. Each artifact type has four signal lists:
//
//   - Strong:     very likely indicators (+3 each)
//   - Structure:  heading or formatting patterns (+2 each)
//   - Supporting: weak corroborating signals (+1 each)
//   - Anti:       signals that make this type less likely (-2 each)
package signals

// Table holds the four signal lists for one artifact type.
type Table struct {
	Strong     []string
	Structure  []string
	Supporting []string
	Anti       []string
}

// Types is the ordered list of all artifact types the heuristic recognises.
var Types = []string{"adr", "spec", "guide", "arch", "open-item"}

// For returns the signal Table for the given artifact type.
// Returns an empty Table for unknown types.
func For(artifactType string) Table {
	t, ok := tables[artifactType]
	if !ok {
		return Table{}
	}
	return t
}

// TopReason returns a short human-readable reason string for the strongest
// signal hit for the given type in text. Used for verbose output.
func TopReason(artifactType, text string) string {
	reasons := map[string]string{
		"adr":       "contains decision-record language",
		"spec":      "contains normative requirement language",
		"guide":     "contains step-by-step instructions",
		"arch":      "contains architectural design language",
		"open-item": "contains open questions or TODO items",
	}
	if r, ok := reasons[artifactType]; ok {
		return r
	}
	return "heuristic classification"
}

var tables = map[string]Table{
	"adr": {
		Strong: []string{
			"we decided",
			"we chose",
			"decision:",
			"rejected",
			"status: accepted",
			"status: proposed",
			"status: deprecated",
			"alternatives considered",
			"## decision",
			"# adr",
			"adr-",
		},
		Structure: []string{
			"## context",
			"## consequences",
			"## alternatives",
			"## status",
			"pros and cons",
			"tradeoff",
			"trade-off",
		},
		Supporting: []string{
			"rationale",
			"because",
			"instead of",
			"over ",
			"we rejected",
			"we considered",
		},
		Anti: []string{
			"## prerequisites",
			"step 1",
			"$ ",
			"```sh",
			"```bash",
		},
	},

	"spec": {
		Strong: []string{
			" must ",
			" must not ",
			" shall ",
			" shall not ",
			" should ",
			" should not ",
			" may ",
			"convention:",
			"required:",
			"normative",
			"## requirements",
			"## convention",
			"## rules",
		},
		Structure: []string{
			"## constraints",
			"## definitions",
			"## overview",
			"/^[0-9]+\\. .+(must|shall|should)/",
		},
		Supporting: []string{
			"compliance",
			"compliant",
			"enforce",
			"workspace must",
			"all agents",
		},
		Anti: []string{
			"step 1",
			"how to",
			"$ ",
			"```sh",
			"```bash",
			"## prerequisites",
		},
	},

	"guide": {
		Strong: []string{
			"how to",
			"step 1",
			"step 2",
			"## steps",
			"## prerequisites",
			"getting started",
			"walkthrough",
		},
		Structure: []string{
			"```sh",
			"```bash",
			"$ ",
			"## example",
			"## usage",
			"/^[0-9]+\\. (run|create|open|add|edit|install|configure|clone|copy|move|delete)/",
		},
		Supporting: []string{
			"run the following",
			"e.g.",
			"for example",
			"before you begin",
			"then run",
		},
		Anti: []string{
			" must ",
			" shall ",
			"we decided",
			"## decision",
		},
	},

	"arch": {
		Strong: []string{
			"architecture",
			"## architecture",
			"## design",
			"data flow",
			"system design",
			"component",
			"how .+ works",
		},
		Structure: []string{
			"```mermaid",
			"graph td",
			"sequencediagram",
			"+---",
			"## layers",
			"## components",
			"## overview",
			"## data flow",
			"## integration",
		},
		Supporting: []string{
			"layer",
			"interface",
			"pipeline",
			"subsystem",
			"module",
			"dependency",
		},
		Anti: []string{
			"step 1",
			"## steps",
			"how to",
			"we decided",
			"TODO",
			"open question",
		},
	},

	"open-item": {
		Strong: []string{
			"todo",
			"fixme",
			"open question",
			"tbd",
			"to be determined",
			"known issue",
			"not yet decided",
			"needs resolution",
			"to be investigated",
		},
		Structure: []string{
			"- [ ]",
			"## open",
			"## questions",
			"## issues",
			"## todos",
		},
		Supporting: []string{
			"?",
			"unclear",
			"unsure",
			"to be",
			"pending",
		},
		Anti: []string{
			"we decided",
			"## decision",
			" must ",
			"architecture",
			"step 1",
		},
	},
}
