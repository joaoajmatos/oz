package filter

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var diffSummaryRE = regexp.MustCompile(`(\d+)\s+files?\s+changed(?:,\s+(\d+)\s+insertions?\(\+\))?(?:,\s+(\d+)\s+deletions?\(-\))?`)

func applyGitDiff(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := normalizeLines(stdout)
	filesChanged := 0
	insertions := 0
	deletions := 0
	changedFiles := make(map[string]struct{})

	for _, line := range lines {
		if m := diffSummaryRE.FindStringSubmatch(line); len(m) > 0 {
			filesChanged = parseInt(m[1], filesChanged)
			insertions = parseInt(m[2], insertions)
			deletions = parseInt(m[3], deletions)
			continue
		}
		if strings.HasPrefix(line, "+++ b/") || strings.HasPrefix(line, "--- a/") {
			path := strings.TrimPrefix(strings.TrimPrefix(line, "+++ b/"), "--- a/")
			if path != "/dev/null" && path != "" {
				changedFiles[path] = struct{}{}
			}
		}
		if strings.HasPrefix(line, "diff --git a/") {
			parts := strings.Split(line, " ")
			if len(parts) >= 3 {
				path := strings.TrimPrefix(parts[2], "a/")
				if path != "" {
					changedFiles[path] = struct{}{}
				}
			}
		}
	}

	out := []string{
		fmt.Sprintf("git diff summary: files=%d insertions=%d deletions=%d", max(filesChanged, len(changedFiles)), insertions, deletions),
	}
	keys := sortedKeys(changedFiles)
	sort.Strings(keys)
	for _, path := range keepHead(keys, ternaryInt(ultraCompact, 5, 10)) {
		out = append(out, "file: "+path)
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	return strings.Join(stableUnique(out), "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}

func parseInt(v string, fallback int) int {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type gitDiffFilter struct{}

func (gitDiffFilter) ID() ID { return FilterGitDiff }

func (gitDiffFilter) Match(args []string) bool {
	return len(args) >= 2 && trimArg(0, args) == "git" && trimArg(1, args) == "diff"
}

func (gitDiffFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	return applyGitDiff(stdout, stderr, exitCode, ultraCompact)
}
