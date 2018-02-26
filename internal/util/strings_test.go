package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Ensure_Suffix(t *testing.T) {
	var tests = []struct {
		s              string
		suffix         string
		expectedString string
	}{
		{"http://example.com", "/", "http://example.com/"},
		{"http://example.com/", "/", "http://example.com/"},
	}

	for _, test := range tests {
		actualString := EnsureSuffix(test.s, test.suffix)
		assert.Equal(t, test.expectedString, actualString, "Unexpected suffix string")
	}
}

func Test_Contains(t *testing.T) {
	var tests = []struct {
		s         string
		list      []string
		contained bool
	}{
		{"42", []string{""}, false},
		{"42", nil, false},
		{"42", []string{"421"}, false},
		{"42", []string{"42"}, true},
		{"42", []string{"100", "42"}, true},
	}

	for _, test := range tests {
		contained := Contains(test.list, test.s)
		assert.Equal(t, test.contained, contained, "Unexpected return from Contains.")
	}
}
