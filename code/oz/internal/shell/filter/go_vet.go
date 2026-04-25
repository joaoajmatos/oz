package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// go vet: file.go:line: message
// staticcheck: file.go:line:col: message
var vetLineRE = regexp.MustCompile(`^(.+\.go):(\d+):(?:\d+:)?\s*(.+)$`)

type goVetFilter struct{}

func (goVetFilter) ID() ID { return FilterGoVet }

func (goVetFilter) Match(args []string) bool {
	return len(args) >= 2 && trimArg(0, args) == "go" && trimArg(1, args) == "vet"
}

func (goVetFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := append(normalizeLines(stderr), normalizeLines(stdout)...)
	byFile := make(map[string][]string)
	seen := make(map[string]struct{})
	for _, line := range lines {
		m := vetLineRE.FindStringSubmatch(line)
		if len(m) != 4 {
			continue
		}
		file, msg := m[1], strings.TrimSpace(m[3])
		key := file + "|" + msg
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		byFile[file] = append(byFile[file], fmt.Sprintf("%s:%s: %s", file, m[2], msg))
	}
	files := sortedKeys(byFile)
	out := []string{fmt.Sprintf("go vet summary: files=%d findings=%d", len(files), len(seen))}
	maxFiles := ternaryInt(ultraCompact, 8, 16)
	files = keepHead(files, maxFiles)
	for _, f := range files {
		for _, l := range keepHead(byFile[f], ternaryInt(ultraCompact, 4, 8)) {
			out = append(out, l)
		}
	}
	if len(out) == 1 {
		out = append(out, "go vet summary: (no file:line findings parsed)")
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	return strings.Join(stableUnique(out), "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}

type staticcheckFilter struct{}

func (staticcheckFilter) ID() ID { return FilterGoVet }

func (staticcheckFilter) Match(args []string) bool {
	return len(args) >= 1 && trimArg(0, args) == "staticcheck"
}

func (f staticcheckFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	return (goVetFilter{}).Apply(stdout, stderr, exitCode, ultraCompact)
}
