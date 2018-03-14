package openshift

import (
	"sync"
)

// UnknownUsersMap is a type-safe and concurrent map keeping track of unknown users.
type UnknownUsersMap struct {
	sync.RWMutex
	internal map[string]interface{}
}

// NewUnknownUsersMap creates a new instance of UnknownUsersMap.
func NewUnknownUsersMap() *UnknownUsersMap {
	return &UnknownUsersMap{
		internal: make(map[string]interface{}),
	}
}

// Load returns the value stored under specified user name.
func (m *UnknownUsersMap) Load(user string) (interface{}, bool) {
	m.RLock()
	result, ok := m.internal[user]
	m.RUnlock()
	return result, ok
}

// Delete deletes the specified user from the map.
func (m *UnknownUsersMap) Delete(user string) {
	m.Lock()
	delete(m.internal, user)
	m.Unlock()
}

// Store stores the specified value under the key user.
func (m *UnknownUsersMap) Store(user string, value interface{}) {
	m.Lock()
	m.internal[user] = value
	m.Unlock()
}
