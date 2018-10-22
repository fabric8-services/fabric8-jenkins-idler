package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_String(t *testing.T) {
	user := User{ID: "42"}
	userAsString := user.StateDump()

	assert.Equal(t, userAsString, "HasBuilds:false HasActiveBuilds:false JenkinsLastUpdate:01 Jan 01 00:00 UTC", "Unexpected string format")
}

func TestNewIdleStatus(t *testing.T) {
	idleError := fmt.Errorf("things are messed up")
	tests := []struct {
		name       string
		inputError error
		success    bool
		reason     string
	}{
		{
			name:       "test output with an error",
			inputError: idleError,
			success:    false,
			reason:     fmt.Sprintf("Failed to idle with error: %v", idleError),
		},
		{
			name:       "test output without an error",
			inputError: nil,
			success:    true,
			reason:     "Successfully idled",
		},
	}

	for _, test := range tests {
		output := NewIdleStatus(test.inputError)
		if output.Success != test.success {
			t.Errorf("Expected success to be %v, got %v", test.success, output.Success)
		}
		if output.Reason != test.reason {
			t.Errorf("Expected reason to be %v, got %v", test.reason, output.Reason)
		}
	}
}

func TestNewUnidleStatus(t *testing.T) {
	unidleError := fmt.Errorf("things are messed up")
	tests := []struct {
		name       string
		inputError error
		success    bool
		reason     string
	}{
		{
			name:       "test output with an error",
			inputError: unidleError,
			success:    false,
			reason:     fmt.Sprintf("Failed to un-idle with error: %v", unidleError),
		},
		{
			name:       "test output without an error",
			inputError: nil,
			success:    true,
			reason:     "Successfully un-idled",
		},
	}

	for _, test := range tests {
		output := NewUnidleStatus(test.inputError)
		if output.Success != test.success {
			t.Errorf("Expected success to be %v, got %v", test.success, output.Success)
		}
		if output.Reason != test.reason {
			t.Errorf("Expected reason to be %v, got %v", test.reason, output.Reason)
		}
	}
}
