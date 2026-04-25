package filter

import (
	"regexp"
	"strings"
)

var secretKeyRE = regexp.MustCompile(`(?i)(token|secret|password|passwd|pwd|bearer|credential|api[_-]?key|auth|private[_-]?key)`)

func isSecretKeyName(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	return secretKeyRE.MatchString(key)
}

func isSensitiveHTTPHeader(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case "authorization", "cookie", "set-cookie", "x-api-key", "x-auth-token":
		return true
	default:
		return false
	}
}
