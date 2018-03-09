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

// NewUserIdlerMap creates a new instance of UserIdlerMap.
func NewUserIdlerMap() *UserIdlerMap {
	return &UserIdlerMap{
		internal: make(map[string]*idler.UserIdler),
	}
}

// Load returns a pointer to UserIdlerMap keyed against the specified key.
func (m *UserIdlerMap) Load(namespace string) (*idler.UserIdler, bool) {
	m.RLock()
	result, ok := m.internal[namespace]
	m.RUnlock()
	return result, ok
}

// Delete deletes the channel of model.User from UserChannelMap given its key.
func (m *UserIdlerMap) Delete(namespace string) {
	m.Lock()
	delete(m.internal, namespace)
	m.Unlock()
}

// Store stores a channel for given model.User at the given key in UserChannelMap.
func (m *UserIdlerMap) Store(key string, i *idler.UserIdler) {
	m.Lock()
	m.internal[key] = i
	m.Unlock()
}
