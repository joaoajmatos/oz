package langs

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/joaoajmatos/oz/internal/shell/readfilter"
)

type jsonReader struct{}

func (jsonReader) Name() string { return "json" }
func (jsonReader) Extensions() []string {
	return []string{".json"}
}

func (jsonReader) Filter(content string, opts readfilter.Options) (string, error) {
	trimmed := strings.TrimSpace(content)
	if !looksLikeJSON(trimmed) {
		return content, nil
	}
	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		return content, nil
	}
	maxKeys := 20
	maxArray := 10
	if opts.UltraCompact {
		maxKeys = 10
		maxArray = 5
	}
	lines := summarizeJSON(v, 0, maxKeys, maxArray)
	return strings.Join(lines, "\n"), nil
}

func looksLikeJSON(s string) bool {
	if s == "" {
		return false
	}
	if !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "[") {
		return false
	}
	return json.Valid([]byte(s))
}

func summarizeJSON(v any, depth, maxKeys, maxArray int) []string {
	if depth > 3 {
		return []string{"... (max depth)"}
	}
	switch typed := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		head := keys
		if len(head) > maxKeys {
			head = head[:maxKeys]
		}
		out := []string{fmt.Sprintf("json object: keys=%d", len(keys))}
		for _, key := range head {
			out = append(out, fmt.Sprintf("  %s: %s", key, jsonType(typed[key])))
		}
		if len(keys) > len(head) {
			out = append(out, fmt.Sprintf("  ... (%d keys omitted)", len(keys)-len(head)))
		}
		return out
	case []any:
		head := typed
		if len(head) > maxArray {
			head = head[:maxArray]
		}
		out := []string{fmt.Sprintf("json array: len=%d", len(typed))}
		for i, val := range head {
			out = append(out, fmt.Sprintf("  [%d]: %s", i, jsonType(val)))
		}
		if len(typed) > len(head) {
			out = append(out, fmt.Sprintf("  ... (%d elements omitted)", len(typed)-len(head)))
		}
		return out
	default:
		return []string{fmt.Sprintf("json scalar: %s", jsonType(typed))}
	}
}

func jsonType(v any) string {
	switch typed := v.(type) {
	case nil:
		return "null"
	case map[string]any:
		return fmt.Sprintf("object(%d keys)", len(typed))
	case []any:
		return fmt.Sprintf("array(len=%d)", len(typed))
	case string:
		if len(typed) > 40 {
			return fmt.Sprintf("string(%q)", typed[:40]+"...")
		}
		return fmt.Sprintf("string(%q)", typed)
	case float64:
		return fmt.Sprintf("number(%g)", typed)
	case bool:
		return fmt.Sprintf("bool(%v)", typed)
	default:
		return fmt.Sprintf("%T", typed)
	}
}

func init() {
	readfilter.Register(jsonReader{})
}
