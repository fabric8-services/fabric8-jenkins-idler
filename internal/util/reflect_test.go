package util

import (
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func Test_name_of_method(t *testing.T) {
	callPtr, _, _, _ := runtime.Caller(0)
	name := NameOfFunction(callPtr)

	assert.Equal(t, "Test_name_of_method", name, "Unexpected function name.")
}
