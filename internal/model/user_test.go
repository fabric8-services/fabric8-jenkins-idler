package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_String(t *testing.T) {
	user := User{ID: "42"}
	userAsString := user.String()

	assert.Equal(t, userAsString, "HasBuilds:false HasActiveBuilds:false JenkinsLastUpdate:01 Jan 01 00:00 UTC", "Unexpected string format")
}
