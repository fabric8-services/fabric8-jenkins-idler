package util

import (
	"math/rand"
	"strconv"
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

// Contains returns true if the specified slice contains the string s. False otherwise.
func Contains(list []string, s string) bool {
	for _, elem := range list {
		if elem == s {
			return true
		}
	}
	return false
}

// RandomString returns random string of length len
func RandomString(len int) string {
	return strconv.Itoa(rand.Intn(1000))
}
