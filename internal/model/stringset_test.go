package model

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringSet_new(t *testing.T) {
	s := NewStringSet()
	assert.NotNil(t, s, "must return a new object")
}

func TestStringSet_count(t *testing.T) {
	s := NewStringSet()
	assert.Equal(t, 0, s.Count(), "must have 0 items")
}

func TestStringSet_add(t *testing.T) {
	s := NewStringSet()
	s.Add([]string{"foo", "bar"})
	assert.Equal(t, 2, s.Count())
}

func TestStringSet_has(t *testing.T) {
	s := NewStringSet()
	assert.False(t, s.Has("foo"), "must be empty")
	s.Add([]string{"foo", "bar"})

	assert.True(t, s.Has("foo"), "must be present after adding")
	assert.True(t, s.Has("bar"), "must be present after adding")
	assert.False(t, s.Has("foobar"), "non existent key seems to be found")
}

func TestStringSet_remove(t *testing.T) {
	s := NewStringSet()
	assert.Equal(t, 0, s.Count())

	s.Add([]string{"foo", "bar"})
	assert.Equal(t, 2, s.Count())

	s.Remove([]string{"bar"})
	assert.Equal(t, 1, s.Count())

	assert.False(t, s.Has("bar"), "key that got removed still exists")
}

func TestStringSet_keys(t *testing.T) {
	s := NewStringSet()

	keys := []string{"foo", "bar"}

	s.Add(keys)

	values := s.Keys()
	sort.Strings(keys)
	sort.Strings(values)

	assert.Equal(t, keys, values)
}
