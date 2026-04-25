package filter

import (
	"encoding/json"
	"fmt"
	"strings"
)

type jsonFilter struct{}

func (jsonFilter) ID() ID { return FilterJSON }

func (jsonFilter) Match(args []string) bool {
	return len(args) >= 1 && trimArg(0, args) == "jq"
}

func (jsonFilter) Apply(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	return applyJSON(stdout, stderr, exitCode, ultraCompact)
}

func applyJSON(stdout, stderr string, exitCode int, ultraCompact bool) (string, string, error) {
	s := strings.TrimSpace(stdout)
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		out := []string{"json: parse failed; raw head:"}
		head := s
		if len(head) > 200 {
			head = head[:200] + "…"
		}
		out = append(out, head)
		if exitCode != 0 {
			out = appendFailureContext(out, stdout, stderr)
		}
		return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
	}
	maxKeys := ternaryInt(ultraCompact, 12, 24)
	maxArr := ternaryInt(ultraCompact, 5, 10)
	out := summarizeJSONValue(v, 0, maxKeys, maxArr, ultraCompact)
	if exitCode != 0 {
		out = appendFailureContext(out, stdout, stderr)
	}
	out = keepHead(stableUnique(out), ternaryInt(ultraCompact, 30, 60))
	return strings.Join(out, "\n"), strings.Join(keepHead(stableUnique(normalizeLines(stderr)), 12), "\n"), nil
}

func summarizeJSONValue(v any, depth, maxKeys, maxArr int, ultraCompact bool) []string {
	if depth > 3 {
		return []string{"…(max depth)"}
	}
	switch t := v.(type) {
	case map[string]any:
		keys := sortedKeys(t)
		head, omitted := topN(keys, maxKeys)
		out := []string{fmt.Sprintf("json object: keys=%d", len(keys))}
		for _, k := range head {
			out = append(out, fmt.Sprintf("  %s: %s", k, jsonTypeSummary(t[k], depth+1, maxKeys, maxArr, ultraCompact)))
		}
		if omitted > 0 {
			out = append(out, fmt.Sprintf("  …(%d keys omitted)", omitted))
		}
		return out
	case []any:
		head, omitted := topN(t, maxArr)
		out := []string{fmt.Sprintf("json array: len=%d", len(t))}
		for i, el := range head {
			out = append(out, fmt.Sprintf("  [%d]: %s", i, jsonTypeSummary(el, depth+1, maxKeys, maxArr, ultraCompact)))
		}
		if omitted > 0 {
			out = append(out, fmt.Sprintf("  …(%d elements omitted)", omitted))
		}
		return out
	default:
		return []string{fmt.Sprintf("json scalar: %s", jsonScalarString(t))}
	}
}

func jsonTypeSummary(v any, depth, maxKeys, maxArr int, ultraCompact bool) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case map[string]any:
		return fmt.Sprintf("object(%d keys)", len(t))
	case []any:
		return fmt.Sprintf("array(len=%d)", len(t))
	case string:
		s := t
		if len(s) > 40 {
			s = s[:40] + "…"
		}
		return fmt.Sprintf("string(%q)", s)
	case float64:
		return fmt.Sprintf("number(%g)", t)
	case bool:
		return fmt.Sprintf("bool(%v)", t)
	default:
		return fmt.Sprintf("%T", t)
	}
}

func jsonScalarString(v any) string {
	switch t := v.(type) {
	case string:
		if len(t) > 80 {
			return t[:80] + "…"
		}
		return t
	default:
		b, _ := json.Marshal(t)
		s := string(b)
		if len(s) > 80 {
			return s[:80] + "…"
		}
		return s
	}
}
