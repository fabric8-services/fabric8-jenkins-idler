package openshift

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/idler"
	cmap "github.com/orcaman/concurrent-map"
)

// UserIdlerMap is a type-safe and concurrent map storing UserIdler instances keyed against the user namespace.
type UserIdlerMap struct {
	internal cmap.ConcurrentMap
}

// NewUserIdlerMap creates a new instance of UnknownUsersMap.
func NewUserIdlerMap() *UserIdlerMap {
	return &UserIdlerMap{
		internal: cmap.New(),
	}
}

// Load returns a pointer to UnknownUsersMap keyed against the specified key.
func (m *UserIdlerMap) Load(namespace string) (*idler.UserIdler, bool) {
	v, ok := m.internal.Get(namespace)
	if !ok {
		return nil, ok
	}
	result, ok := v.(*idler.UserIdler)
	return result, ok
}

// Delete deletes the entry for the specified namespace from the map.
func (m *UserIdlerMap) Delete(namespace string) {
	m.internal.Remove(namespace)
}

// Len returns number of items in map
func (m *UserIdlerMap) Len() int {
	return m.internal.Count()
}

// Store stores the user user idler with the given key.
func (m *UserIdlerMap) Store(namespace string, i *idler.UserIdler) {
	m.internal.Set(namespace, i)
}
