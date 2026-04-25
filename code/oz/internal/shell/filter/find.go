package filter

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const findMaxPaths = 50

type findFilter struct{}

func (findFilter) ID() ID { return FilterFind }

func (findFilter) Match(args []string) bool {
	return len(args) >= 1 && trimArg(0, args) == "find"
}

func (findFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	lines := normalizeLines(stripANSI(stdout))
	seen := make(map[string]struct{})
	paths := make([]string, 0)
	for _, line := range lines {
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		paths = append(paths, line)
	}
	sort.Strings(paths)
	extCount := make(map[string]int)
	for _, p := range paths {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(p), "."))
		if ext == "" {
			ext = "(noext)"
		}
		extCount[ext]++
	}
	extKeys := sortedKeys(extCount)
	maxExt := ternaryInt(ultraCompact, 8, 16)
	extKeys = keepHead(extKeys, maxExt)
	maxPaths := ternaryInt(ultraCompact, 25, findMaxPaths)
	head, omitted := topN(paths, maxPaths)
	out := []string{
		fmt.Sprintf("find summary: unique_paths=%d", len(paths)),
	}
	var extParts []string
	for _, k := range extKeys {
		extParts = append(extParts, fmt.Sprintf("%s=%d", k, extCount[k]))
	}
	if len(extParts) > 0 {
		out = append(out, "by_ext: "+strings.Join(extParts, ", "))
	}
	out = append(out, "sample:"+formatOmittedSuffix(omitted))
	for _, p := range head {
		out = append(out, "  "+p)
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 40, 80))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}
