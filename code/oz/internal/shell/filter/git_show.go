package filter

import "strings"

type gitShowFilter struct{}

func (gitShowFilter) ID() ID { return FilterGitShow }

func (gitShowFilter) Match(args []string) bool {
	return len(args) >= 2 && trimArg(0, args) == "git" && trimArg(1, args) == "show"
}

func (gitShowFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	meta, patch := splitGitShow(stdout)
	metaLines := normalizeLines(meta)
	metaOut := keepHead(metaLines, ternaryInt(ultraCompact, 12, 24))
	patchCompact, patchErr, err := applyGitDiff(patch, stderr, exitCode, ultraCompact)
	if err != nil {
		return "", "", err
	}
	if patchErr != "" {
		patchCompact = patchCompact + "\n\nstderr:\n" + patchErr
	}
	out := []string{"git show metadata:"}
	for _, l := range metaOut {
		out = append(out, "  "+l)
	}
	out = append(out, "git show diff:")
	for _, l := range strings.Split(patchCompact, "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		out = append(out, l)
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	return strings.Join(stableUnique(out), "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}

func splitGitShow(stdout string) (meta, patch string) {
	lines := strings.Split(strings.ReplaceAll(stdout, "\r\n", "\n"), "\n")
	idx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			idx = i
			break
		}
	}
	if idx < 0 {
		return strings.TrimSpace(stdout), ""
	}
	return strings.Join(lines[:idx], "\n"), strings.Join(lines[idx:], "\n")
}
