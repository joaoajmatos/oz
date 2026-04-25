package filter

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var lsLongPermRE = regexp.MustCompile(`^[-dlcbps?][-rwxsStT.+]{9}[+@]?\s+`)

type lsFilter struct{}

func (lsFilter) ID() ID { return FilterLs }

func (lsFilter) Match(args []string) bool {
	return len(args) >= 1 && trimArg(0, args) == "ls"
}

func (lsFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	raw := strings.Split(strings.ReplaceAll(stripANSI(stdout), "\r\n", "\n"), "\n")
	dirs, files, links := 0, 0, 0
	names := make([]string, 0)
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if lsLongPermRE.MatchString(line) {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			perm := fields[0]
			switch perm[0] {
			case 'd':
				dirs++
			case 'l':
				links++
			default:
				files++
			}
			name := fields[len(fields)-1]
			if name != "." && name != ".." {
				names = append(names, name)
			}
			continue
		}
		if strings.HasPrefix(line, "total ") {
			continue
		}
		names = append(names, line)
	}
	sort.Strings(names)
	maxNames := ternaryInt(ultraCompact, 15, 30)
	head, omitted := topN(names, maxNames)
	out := []string{
		fmt.Sprintf("ls summary: dirs=%d files=%d links=%d entries=%d", dirs, files, links, len(names)),
	}
	if len(head) > 0 {
		out = append(out, "names:"+formatOmittedSuffix(omitted))
		for _, n := range head {
			out = append(out, "  "+n)
		}
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 24, 40))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
