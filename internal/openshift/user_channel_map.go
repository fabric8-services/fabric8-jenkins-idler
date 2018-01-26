package openshift

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"sync"
)

type UserChannelMap struct {
	sync.RWMutex
	internal map[string]chan model.User
}

// A type-safe and concurrent map storing channels for model.User keyed against the user namespace.
func NewUserChannelMap() *UserChannelMap {
	return &UserChannelMap{
		internal: make(map[string]chan model.User),
	}
}

func (rm *UserChannelMap) Load(key string) (chan model.User, bool) {
	rm.RLock()
	result, ok := rm.internal[key]
	rm.RUnlock()
	return result, ok
}

func (rm *UserChannelMap) Delete(key string) {
	rm.Lock()
	delete(rm.internal, key)
	rm.Unlock()
}

func (rm *UserChannelMap) Store(key string, c chan model.User) {
	rm.Lock()
	rm.internal[key] = c
	rm.Unlock()
}
