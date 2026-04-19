package convention

// Version is the current oz standard version.
const Version = "v0.1"

// Tier is a source-of-truth band in the workspace hierarchy.
type Tier string

const (
	TierSpecs   Tier = "specs"
	TierDocs    Tier = "docs"
	TierContext Tier = "context"
	TierNotes   Tier = "notes"
)

// SourceOfTruthOrder defines the source-of-truth order, highest trust first.
var SourceOfTruthOrder = []Tier{TierSpecs, TierDocs, TierContext, TierNotes}

// TrustRank returns 0 for highest-trust tiers, then increasing values.
// Unknown or empty tiers sort after all known tiers.
func (t Tier) TrustRank() int {
	if t == "" {
		return len(SourceOfTruthOrder)
	}
	for i, x := range SourceOfTruthOrder {
		if t == x {
			return i
		}
	}
	return len(SourceOfTruthOrder)
}

// LessTrustTier reports whether tier a should sort before tier b when ordering
// by descending trust (more trusted material first).
func LessTrustTier(a, b Tier) bool {
	ra, rb := a.TrustRank(), b.TrustRank()
	if ra != rb {
		return ra < rb
	}
	return string(a) < string(b)
}

// Directories defines required/recommended/optional workspace directories.
var Directories = map[string]string{
	"agents":          "required",
	"specs/decisions": "required",
	"docs":            "required",
	"context":         "required",
	"skills":          "required",
	"rules":           "required",
	"notes":           "required",
	"code":            "recommended",
	"tools":           "optional",
	"scripts":         "optional",
	".oz":             "optional",
}

// RootFiles defines required files at the workspace root.
var RootFiles = map[string]string{
	"AGENTS.md": "required",
	"OZ.md":     "required",
	"README.md":  "recommended",
}

// DefaultAgents are the agent names created when using the default set.
var DefaultAgents = []string{"coding", "maintainer", "onboarding"}
