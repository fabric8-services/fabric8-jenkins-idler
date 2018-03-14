package openshift

import (
	"sync"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/idler"
)

// UserIdlerMap is a type-safe and concurrent map storing UserIdler instances keyed against the user namespace.
type UserIdlerMap struct {
	sync.RWMutex
	internal map[string]*idler.UserIdler
}

// NewUserIdlerMap creates a new instance of UnknownUsersMap.
func NewUserIdlerMap() *UserIdlerMap {
	return &UserIdlerMap{
		internal: make(map[string]*idler.UserIdler),
	}
}

// Load returns a pointer to UnknownUsersMap keyed against the specified key.
func (m *UserIdlerMap) Load(namespace string) (*idler.UserIdler, bool) {
	m.RLock()
	result, ok := m.internal[namespace]
	m.RUnlock()
	return result, ok
}

// Delete deletes the entry for the specified namespace from the map.
func (m *UserIdlerMap) Delete(namespace string) {
	m.Lock()
	delete(m.internal, namespace)
	m.Unlock()
}

// Store stores the user user idler with the given key.
func (m *UserIdlerMap) Store(key string, i *idler.UserIdler) {
	m.Lock()
	m.internal[key] = i
	m.Unlock()
}
