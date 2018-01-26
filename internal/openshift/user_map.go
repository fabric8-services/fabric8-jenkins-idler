package openshift

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"sync"
)

// A type-safe and concurrent map storing instances of model.User keyed against the user namespace.
type UserMap struct {
	sync.RWMutex
	internal map[string]model.User
}

func NewUserMap() *UserMap {
	return &UserMap{
		internal: make(map[string]model.User),
	}
}

func (rm *UserMap) Load(key string) (model.User, bool) {
	rm.RLock()
	result, ok := rm.internal[key]
	rm.RUnlock()
	return result, ok
}

func (rm *UserMap) Delete(key string) {
	rm.Lock()
	delete(rm.internal, key)
	rm.Unlock()
}

func (rm *UserMap) Store(key string, user model.User) {
	rm.Lock()
	rm.internal[key] = user
	rm.Unlock()
}
