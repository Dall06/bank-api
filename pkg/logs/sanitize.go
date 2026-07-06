package logs

import "strings"

// isBlocked checks if the key contains any of the blocked substrings.
func isBlocked(key string, blockedFields []string) bool {
	lowerKey := strings.ToLower(key)
	for _, blocked := range blockedFields {
		if strings.Contains(lowerKey, strings.ToLower(blocked)) {
			return true
		}
	}
	return false
}
