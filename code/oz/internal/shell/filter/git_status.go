package filter

import (
	"fmt"
	"regexp"
	"strings"
)

var porcelainStatusRE = regexp.MustCompile(`^[ MARCUD?!][ MARCUD?!] `)

func applyGitStatus(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := normalizeLines(stdout)
	staged := 0
	unstaged := 0
	untracked := 0
	conflicts := 0
	other := 0

	for _, line := range lines {
		if porcelainStatusRE.MatchString(line) {
			if strings.HasPrefix(line, "?? ") {
				untracked++
				continue
			}
			x := line[0]
			y := line[1]
			if x != ' ' && x != '?' {
				staged++
			}
			if y != ' ' && y != '?' {
				unstaged++
			}
			if x == 'U' || y == 'U' {
				conflicts++
			}
			continue
		}
		switch {
		case strings.Contains(line, "Changes to be committed"):
			staged++
		case strings.Contains(line, "Changes not staged for commit"):
			unstaged++
		case strings.Contains(line, "Untracked files"):
			untracked++
		case strings.Contains(line, "both modified"):
			conflicts++
		default:
			other++
		}
	}

	out := []string{
		fmt.Sprintf("git status summary: staged=%d unstaged=%d untracked=%d conflicts=%d", staged, unstaged, untracked, conflicts),
	}
	if other > 0 {
		out = append(out, fmt.Sprintf("context_lines=%d", other))
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 8, 16))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
