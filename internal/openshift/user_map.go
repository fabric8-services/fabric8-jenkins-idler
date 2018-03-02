package openshift

import (
	"sync"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
)

// UserMap is a type-safe and concurrent map storing instances of model.User keyed against the user namespace.
type UserMap struct {
	sync.RWMutex
	internal map[string]model.User
}

// NewUserMap creates a new instance of the UserMap type.
func NewUserMap() *UserMap {
	return &UserMap{
		internal: make(map[string]model.User),
	}
}

// Load gets a model.User from UserMap given its key.
func (rm *UserMap) Load(key string) (model.User, bool) {
	rm.RLock()
	result, ok := rm.internal[key]
	rm.RUnlock()
	return result, ok
}

// Delete deletes a model.User from UserMap given its key.
func (rm *UserMap) Delete(key string) {
	rm.Lock()
	delete(rm.internal, key)
	rm.Unlock()
}

// Store stores given model.User at the given key in UserMap.
func (rm *UserMap) Store(key string, user model.User) {
	rm.Lock()
	rm.internal[key] = user
	rm.Unlock()
}
