package util

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsURL(t *testing.T) {
	var testURLs = []struct {
		url    string
		errors []string
	}{
		{"http://localhist:9999/api", []string{}},
		{"http://localhist:9999/api/", []string{}},
		{"foo", []string{"value for FOO needs to be a valid URL"}},
		{"/foo", []string{"value for FOO needs to be a valid URL"}},
		{"ftp://localhost", []string{"value for FOO needs to be a valid URL"}},
		{"", []string{"value for FOO needs to be a valid URL"}},
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

func Test_IsBool(t *testing.T) {
	var testBools = []struct {
		value    string
		expected bool
		errors   []string
	}{
		{"true", true, []string{}},
		{"false", false, []string{}},
		{"0", false, []string{}},
		{"1", true, []string{}},
		{"snafu", false, []string{"value for FOO needs to be an bool"}},
		{"", false, []string{"value for FOO needs to be an bool"}},
	}

	for _, testBool := range testBools {
		err := IsBool(testBool.value, "FOO")
		var errors []string
		if err == nil {
			errors = []string{}
		} else {
			errors = strings.Split(err.Error(), "\n")
		}

		assert.Equal(t, testBool.errors, errors, fmt.Sprintf("Unexpected error for %s", testBool.value))
	}
}
