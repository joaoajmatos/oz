package hooks

import (
	"strconv"
	"strings"
)

type Decision struct {
	Allowed   bool
	Mode      Mode
	Suggested string
	Rewritten string
	Reason    Reason
}

type Reason string

const (
	ReasonEmptyCommand       Reason = "empty command"
	ReasonHooksDisabled      Reason = "hooks disabled"
	ReasonCommandExcluded    Reason = "command excluded"
	ReasonAlreadyWrapped     Reason = "already wrapped"
	ReasonRewriteOptIn       Reason = "rewrite opt-in enabled"
	ReasonSuggestModeDefault Reason = "suggest mode default"
)

func Decide(command string, cfg Config) Decision {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return Decision{Allowed: false, Mode: cfg.Mode, Reason: ReasonEmptyCommand}
	}
	if !cfg.Enabled {
		return Decision{Allowed: false, Mode: cfg.Mode, Reason: ReasonHooksDisabled}
	}
	base := firstToken(trimmed)
	for _, excluded := range cfg.ExcludeCommands {
		if strings.EqualFold(strings.TrimSpace(excluded), base) {
			return Decision{Allowed: false, Mode: cfg.Mode, Reason: ReasonCommandExcluded}
		}
	}
	if isAlreadyWrapped(trimmed) {
		return Decision{Allowed: false, Mode: cfg.Mode, Reason: ReasonAlreadyWrapped}
	}

	wrapped := rewriteCompound(trimmed, cfg)
	if wrapped == trimmed {
		return Decision{Allowed: false, Mode: cfg.Mode, Reason: ReasonCommandExcluded}
	}
	if cfg.Mode == ModeRewrite {
		return Decision{
			Allowed:   true,
			Mode:      cfg.Mode,
			Rewritten: wrapped,
			Reason:    ReasonRewriteOptIn,
		}
	}
	return Decision{
		Allowed:   true,
		Mode:      ModeSuggest,
		Suggested: wrapped,
		Reason:    ReasonSuggestModeDefault,
	}
}

func firstToken(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], `"'`)
}

func isAlreadyWrapped(command string) bool {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(command)))
	if len(fields) < 3 || fields[0] != "oz" || fields[1] != "shell" {
		return false
	}
	switch fields[2] {
	case "read":
		return true
	case "run":
		return len(fields) >= 4 && fields[3] == "--"
	}
	return false
}

func rewriteCompound(command string, cfg Config) string {
	var out strings.Builder
	i := 0
	for i < len(command) {
		seg, op, consumed := nextSegment(command[i:])
		rewritten := rewriteSegment(seg, cfg)
		out.WriteString(rewritten)
		if op != "" {
			out.WriteString(op)
		}
		i += consumed
	}
	return out.String()
}

func rewriteSegment(segment string, cfg Config) string {
	trimmed := strings.TrimSpace(segment)
	if trimmed == "" {
		return segment
	}
	if isAlreadyWrapped(trimmed) {
		return segment
	}
	base := firstToken(trimmed)
	for _, excluded := range cfg.ExcludeCommands {
		if strings.EqualFold(strings.TrimSpace(excluded), base) {
			return segment
		}
	}

	leading := segment[:len(segment)-len(strings.TrimLeft(segment, " \t"))]
	trailing := segment[len(strings.TrimRight(segment, " \t")):]

	if rewritten, ok := tryRewriteFileRead(trimmed); ok {
		return leading + rewritten + trailing
	}
	return leading + "oz shell run -- " + trimmed + trailing
}

// tryRewriteFileRead rewrites cat/head/tail file reads to oz shell read so the
// language-aware readfilter path is used instead of the generic run filter.
// Returns the rewritten command and true if applicable, "" and false otherwise.
func tryRewriteFileRead(command string) (string, bool) {
	// Bail on redirections — we can't safely rewrite those.
	if strings.ContainsAny(command, "<>") {
		return "", false
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "", false
	}
	cmd := strings.ToLower(strings.Trim(fields[0], `"'`))
	switch cmd {
	case "cat":
		return rewriteCatRead(fields)
	case "head":
		return rewriteWindowedRead(fields, "--max-lines", 10)
	case "tail":
		return rewriteWindowedRead(fields, "--tail-lines", 10)
	}
	return "", false
}

func rewriteCatRead(fields []string) (string, bool) {
	args := fields[1:]
	if len(args) == 0 {
		return "", false // bare cat reads stdin interactively; skip
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") && a != "-" {
			return "", false // unsupported flag
		}
	}
	return "oz shell read " + strings.Join(args, " "), true
}

// rewriteWindowedRead handles head/tail with optional -n N or -N line count.
// Unsupported flags (e.g. tail -f) fall through to oz shell run.
func rewriteWindowedRead(fields []string, flag string, defaultLines int) (string, bool) {
	args := fields[1:]
	lines := 0
	var files []string
	for i := 0; i < len(args); {
		a := args[i]
		switch {
		case a == "-n" || a == "--lines":
			if i+1 >= len(args) {
				return "", false
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n <= 0 {
				return "", false
			}
			lines = n
			i += 2
		case len(a) > 1 && a[0] == '-' && isAllDigits(a[1:]):
			n, err := strconv.Atoi(a[1:])
			if err != nil || n <= 0 {
				return "", false
			}
			lines = n
			i++
		case strings.HasPrefix(a, "-"):
			return "", false // unsupported flag
		default:
			files = append(files, a)
			i++
		}
	}
	if len(files) == 0 {
		return "", false
	}
	if lines == 0 {
		lines = defaultLines
	}
	return "oz shell read " + flag + " " + strconv.Itoa(lines) + " " + strings.Join(files, " "), true
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func nextSegment(input string) (segment, op string, consumed int) {
	opIdx, foundOp := findSplitOp(input)
	pipeIdx := strings.Index(input, "|")
	switch {
	case pipeIdx >= 0 && (opIdx == -1 || pipeIdx < opIdx):
		return input[:pipeIdx], input[pipeIdx:], len(input)
	case opIdx >= 0:
		return input[:opIdx], foundOp, opIdx + len(foundOp)
	default:
		return input, "", len(input)
	}
}

func findSplitOp(input string) (int, string) {
	bestIdx := -1
	bestOp := ""
	ops := []string{"&&", "||", ";"}
	for _, op := range ops {
		if idx := strings.Index(input, op); idx >= 0 && (bestIdx == -1 || idx < bestIdx) {
			bestIdx = idx
			bestOp = op
		}
	}
	return bestIdx, bestOp
}
