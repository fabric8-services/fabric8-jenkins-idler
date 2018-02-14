package util

import (
	"strings"
)

// EnsureSuffix ensures that hte given strings ends with the specified suffix.
// If the string s already ends with the suffix suffix, then it is returned unmodified, otherwise
// string s with the suffix appended is returned.
func EnsureSuffix(s string, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		return s
	}
	return s + suffix
}
