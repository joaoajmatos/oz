package filter

import (
	"fmt"
	"regexp"
	"strings"
)

var gitLogCommitRE = regexp.MustCompile(`^commit ([0-9a-f]{7,40})`)

type gitLogCommit struct {
	hash, subject, author, date string
}

type gitLogFilter struct{}

func (gitLogFilter) ID() ID { return FilterGitLog }

func (gitLogFilter) Match(args []string) bool {
	return len(args) >= 2 && trimArg(0, args) == "git" && trimArg(1, args) == "log"
}

func (gitLogFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	commits := parseGitLogCommits(stdout)
	maxCommits := ternaryInt(ultraCompact, 10, 20)
	shown, omitted := topN(commits, maxCommits)
	out := []string{fmt.Sprintf("git log summary: total=%d shown=%d", len(commits), len(shown))}
	for _, c := range shown {
		sub := c.subject
		if len(sub) > 80 {
			sub = sub[:80] + "…"
		}
		out = append(out, fmt.Sprintf("%s · %s · %s · %s", shortHash(c.hash), sub, c.author, c.date))
	}
	if omitted > 0 {
		out = append(out, fmt.Sprintf("…(%d commits omitted)", omitted))
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	return strings.Join(stableUnique(out), "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}

func shortHash(h string) string {
	h = strings.TrimSpace(h)
	if len(h) <= 12 {
		return h
	}
	return h[:12]
}

func parseGitLogCommits(stdout string) []gitLogCommit {
	raw := strings.Split(strings.ReplaceAll(stdout, "\r\n", "\n"), "\n")
	var commits []gitLogCommit
	var cur *gitLogCommit
	flush := func() {
		if cur == nil || cur.hash == "" {
			return
		}
		commits = append(commits, *cur)
		cur = nil
	}
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if m := gitLogCommitRE.FindStringSubmatch(line); len(m) == 2 {
			flush()
			cur = &gitLogCommit{hash: m[1]}
			continue
		}
		if cur == nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "Author:"):
			cur.author = strings.TrimSpace(strings.TrimPrefix(line, "Author:"))
		case strings.HasPrefix(line, "Date:"):
			cur.date = strings.TrimSpace(strings.TrimPrefix(line, "Date:"))
		case strings.HasPrefix(line, "Merge:"):
			// skip
		default:
			if cur.subject == "" && line != "" {
				cur.subject = line
			}
		}
	}
	flush()
	return commits
}
