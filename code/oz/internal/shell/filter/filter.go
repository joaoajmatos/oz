package filter

import (
	"fmt"
	"os"
)

func Apply(args []string, stdout, stderr string, exitCode int, ultraCompact bool) (compactStdout, compactStderr string, matched ID, err error) {
	f := lookup(args)
	matched = FilterGeneric
	if f != nil {
		matched = f.ID()
	}
	if matched != FilterGeneric && os.Getenv("OZ_SHELL_TEST_FORCE_FILTER_ERROR") == "1" &&
		hasArg(args, "__OZ_FORCE_SPECIALIZED_FILTER_ERROR__") {
		return "", "", matched, fmt.Errorf("forced specialized filter error")
	}
	if f != nil {
		compactStdout, compactStderr, err = f.Apply(stdout, stderr, exitCode, ultraCompact)
	} else if looksLikeJSON(stdout) {
		matched = FilterJSON
		compactStdout, compactStderr, err = applyJSON(stdout, stderr, exitCode, ultraCompact)
	} else {
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
