package filter

import "strings"

const (
	FilterGeneric   = "generic"
	FilterGitStatus = "git.status"
	FilterGitDiff   = "git.diff"
	FilterRG        = "rg"
	FilterGoTest    = "go.test"
)

func Classify(args []string) string {
	if len(args) < 1 {
		return FilterGeneric
	}
	cmd := strings.TrimSpace(args[0])
	switch cmd {
	case "git":
		if len(args) < 2 {
			return FilterGeneric
		}
		switch strings.TrimSpace(args[1]) {
		case "status":
			return FilterGitStatus
		case "diff":
			return FilterGitDiff
		default:
			return FilterGeneric
		}
	case "rg", "grep":
		return FilterRG
	case "go":
		if len(args) >= 2 && strings.TrimSpace(args[1]) == "test" {
			return FilterGoTest
		}
		return FilterGeneric
	default:
		return FilterGeneric
	}
}
