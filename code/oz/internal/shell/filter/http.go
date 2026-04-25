package filter

import (
	"fmt"
	"strings"
)

type httpFilter struct{}

func (httpFilter) ID() ID { return FilterHTTP }

func (httpFilter) Match(args []string) bool {
	if len(args) < 1 {
		return false
	}
	switch trimArg(0, args) {
	case "curl", "wget", "http", "https":
		return true
	default:
		return false
	}
}

func (httpFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	text := strings.TrimSpace(stripANSI(stdout))
	if text == "" {
		text = strings.TrimSpace(stripANSI(stderr))
	} else if strings.TrimSpace(stripANSI(stderr)) != "" {
		text = text + "\n" + strings.TrimSpace(stripANSI(stderr))
	}
	parts := strings.SplitN(text, "\n\n", 2)
	head := parts[0]
	var body string
	if len(parts) == 2 {
		body = strings.TrimSpace(parts[1])
	}
	out := make([]string, 0)
	for _, line := range normalizeLines(head) {
		if strings.HasPrefix(line, "HTTP/") {
			out = append(out, "status: "+line)
			continue
		}
		name, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		val = strings.TrimSpace(val)
		lower := strings.ToLower(name)
		switch lower {
		case "content-type", "content-length", "location", "cache-control":
			out = append(out, fmt.Sprintf("%s: %s", name, truncateRunes(val, 120)))
		case "authorization", "cookie", "set-cookie", "x-api-key", "x-auth-token":
			out = append(out, fmt.Sprintf("%s: <redacted>", name))
		}
	}
	if body != "" {
		if looksLikeJSON(body) {
			jsOut, jsErr, err := applyJSON(body, "", exitCode, ultraCompact)
			if err != nil {
				return "", "", err
			}
			out = append(out, "body(json):")
			for _, l := range strings.Split(jsOut, "\n") {
				if strings.TrimSpace(l) != "" {
					out = append(out, "  "+strings.TrimSpace(l))
				}
			}
			if strings.TrimSpace(jsErr) != "" {
				out = append(out, "json stderr: "+jsErr)
			}
		} else {
			out = append(out, "body: "+truncateRunes(body, ternaryInt(ultraCompact, 200, 400)))
		}
	}
	if len(out) == 0 {
		out = append(out, "http summary: (no structured parse)")
		out = append(out, "raw_head: "+truncateRunes(text, ternaryInt(ultraCompact, 200, 400)))
	}
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 30, 60))
	return strings.Join(out, "\n"), "", nil
}
