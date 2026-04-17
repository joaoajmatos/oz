package convention

// Version is the current oz standard version.
const Version = "v0.1"

// Hierarchy defines the source of truth order, highest trust first.
var Hierarchy = []string{"specs", "docs", "context", "notes"}

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
