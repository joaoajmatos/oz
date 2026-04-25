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
	if strings.HasPrefix(trimmed, "oz shell run --") {
		return Decision{Allowed: false, Mode: cfg.Mode, Reason: "already wrapped"}
	}

	wrapped := "oz shell run -- " + trimmed
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
	return fields[0]
}
