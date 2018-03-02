package toggles

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_default_values(t *testing.T) {
	testUuids := []string{"42", "1001"}
	toggle, err := NewFixedUUIDToggle(testUuids)
	assert.NoError(t, err, "Creating toggle failed unexpectedly.")
	assert.NotNil(t, toggle, "No nil object expected")

	var toggleTests = []struct {
		uuid    string
		enabled bool
	}{
		{"42", true},
		{"1001", true},
		{"", false},
		{"1", false},
		{"abc", false},
	}

	for _, toggleTest := range toggleTests {
		result, err := toggle.IsIdlerEnabled(toggleTest.uuid)
		assert.NoError(t, err, "IsIdlerEnabled call failed unexpectedly.")
		assert.Equal(t, toggleTest.enabled, result, "Unexpected result for IsIdlerEnabled")
	}
}
