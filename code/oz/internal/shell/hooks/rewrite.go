package hooks

import "strings"

type Decision struct {
	Allowed   bool
	Mode      Mode
	Suggested string
	Rewritten string
	Reason    string
}

func Decide(command string, cfg Config) Decision {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return Decision{Allowed: false, Mode: cfg.Mode, Reason: "empty command"}
	}
	if !cfg.Enabled {
		return Decision{Allowed: false, Mode: cfg.Mode, Reason: "hooks disabled"}
	}
	base := firstToken(trimmed)
	for _, excluded := range cfg.ExcludeCommands {
		if strings.EqualFold(strings.TrimSpace(excluded), base) {
			return Decision{Allowed: false, Mode: cfg.Mode, Reason: "command excluded"}
		}
	}
	if isAlreadyWrapped(trimmed) {
		return Decision{Allowed: false, Mode: cfg.Mode, Reason: "already wrapped"}
	}

	wrapped := rewriteCompound(trimmed, cfg)
	if wrapped == trimmed {
		return Decision{Allowed: false, Mode: cfg.Mode, Reason: "command excluded"}
	}
	if cfg.Mode == ModeRewrite {
		return Decision{
			Allowed:   true,
			Mode:      cfg.Mode,
			Rewritten: wrapped,
			Reason:    "rewrite opt-in enabled",
		}
	}
	return Decision{
		Allowed:   true,
		Mode:      ModeSuggest,
		Suggested: wrapped,
		Reason:    "suggest mode default",
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
	if len(fields) < 4 {
		return false
	}
	return fields[0] == "oz" && fields[1] == "shell" && fields[2] == "run" && fields[3] == "--"
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
	return leading + "oz shell run -- " + trimmed + trailing
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
