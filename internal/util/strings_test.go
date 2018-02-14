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
