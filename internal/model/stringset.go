package model

import (
	cmap "github.com/orcaman/concurrent-map"
)

// StringSet is a type-safe and concurrent map keeping track of disabled v for idler.
type StringSet struct {
	cmap.ConcurrentMap
}

// NewStringSet creates a new instance of StringSet map.
func NewStringSet() *StringSet {
	return &StringSet{cmap.New()}
}

// Add stores the specified value under the key user.
func (m *StringSet) Add(vs []string) {
	for _, v := range vs {
		m.ConcurrentMap.Set(v, true)
	}
}

// Remove deletes the specified disabled user from the map.
func (m *StringSet) Remove(vs []string) {
	for _, v := range vs {
		m.ConcurrentMap.Remove(v)
	}
}
