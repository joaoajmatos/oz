package filter

import "strings"

type ID string

const (
	FilterNone      ID = "none"
	FilterGeneric   ID = "generic"
	FilterGitStatus ID = "git.status"
	FilterGitDiff   ID = "git.diff"
	FilterRG        ID = "rg"
	FilterGoTest    ID = "go.test"
)

func Classify(args []string) ID {
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
