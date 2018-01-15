package util

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func Test_IsURL(t *testing.T) {
	var testURLs = []struct {
		url    string
		errors []string
	}{
		{"http://localhist:9999/api", []string{}},
		{"http://localhist:9999/api/", []string{}},
		{"foo", []string{"Value for FOO needs to be a valid URL."}},
		{"/foo", []string{"Value for FOO needs to be a valid URL."}},
		{"ftp://localhost", []string{"Value for FOO needs to be a valid URL."}},
		{"", []string{"Value for FOO needs to be a valid URL."}},
	}

	for _, testURL := range testURLs {
		err := IsURL(testURL.url, "FOO")
		var errors []string
		if err == nil {
			errors = []string{}
		} else {
			errors = strings.Split(err.Error(), "\n")
		}

		assert.Equal(t, testURL.errors, errors, fmt.Sprintf("Unexpected error for %s", testURL.url))
	}

}
