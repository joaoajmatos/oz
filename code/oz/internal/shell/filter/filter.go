package filter

import (
	"fmt"
	"os"
)

func Apply(args []string, stdout, stderr string, exitCode int, ultraCompact bool) (compactStdout, compactStderr string, matched ID, err error) {
	matched = Classify(args)
	if matched != FilterGeneric && os.Getenv("OZ_SHELL_TEST_FORCE_FILTER_ERROR") == "1" &&
		hasArg(args, "__OZ_FORCE_SPECIALIZED_FILTER_ERROR__") {
		return "", "", matched, fmt.Errorf("forced specialized filter error")
	}
	switch matched {
	case FilterGitStatus:
		compactStdout, compactStderr, err = applyGitStatus(stdout, stderr, exitCode, ultraCompact)
	case FilterGitDiff:
		compactStdout, compactStderr, err = applyGitDiff(stdout, stderr, exitCode, ultraCompact)
	case FilterRG:
		compactStdout, compactStderr, err = applyRG(stdout, stderr, exitCode, ultraCompact)
	case FilterGoTest:
		compactStdout, compactStderr, err = applyGoTest(stdout, stderr, exitCode, ultraCompact)
	default:
		matched = FilterGeneric
		compactStdout, compactStderr, err = fallbackGeneric(stdout, stderr, ultraCompact)
	}
	if err != nil {
		return "", "", matched, fmt.Errorf("apply %s filter: %w", matched, err)
	}
	return compactStdout, compactStderr, matched, nil
}

func hasArg(args []string, needle string) bool {
	for _, arg := range args {
		if arg == needle {
			return true
		}
	}
	return false
}
