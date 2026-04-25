package filter

type diffFilter struct{}

func (diffFilter) ID() ID { return FilterDiff }

func (diffFilter) Match(args []string) bool {
	if len(args) < 1 {
		return false
	}
	switch trimArg(0, args) {
	case "diff", "patch":
		return true
	default:
		return false
	}
}

func (diffFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	return applyGitDiff(stdout, stderr, exitCode, ultraCompact)
}
