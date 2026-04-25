package filter

import "strings"

// Filter is a specialized shell-output compactor matched by argv.
type Filter interface {
	ID() ID
	// Match returns whether this filter should handle the given command argv.
	Match(args []string) bool
	Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error)
}

var registry []Filter

// Register appends a filter to the global registry (call from init in register order).
func Register(f Filter) {
	if f == nil {
		return
	}
	registry = append(registry, f)
}

func lookup(args []string) Filter {
	for _, f := range registry {
		if f.Match(args) {
			return f
		}
	}
	return nil
}

func trimArg(i int, args []string) string {
	if i < 0 || i >= len(args) {
		return ""
	}
	return strings.TrimSpace(args[i])
}
