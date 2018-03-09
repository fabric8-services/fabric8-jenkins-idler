package openshift

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_retrieve_idler_for_unknown_namespace_returns_nil(t *testing.T) {
	m := NewUserIdlerMap()
	idler, ok := m.Load("foo")
	assert.False(t, ok, "There should be no entry mapped")
	assert.Nil(t, idler, "No reference should be returned")
}

func Test_retrieve_idler_for_empty_namespace_returns_nil(t *testing.T) {
	m := NewUserIdlerMap()
	idler, ok := m.Load("")
	assert.False(t, ok, "There should be no entry mapped")
	assert.Nil(t, idler, "No reference should be returned")
}
