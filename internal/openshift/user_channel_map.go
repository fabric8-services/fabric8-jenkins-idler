package openshift

import (
	"sync"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
)

// UserChannelMap is a type-safe and concurrent map storing channels for model.User keyed against the user namespace.
type UserChannelMap struct {
	sync.RWMutex
	internal map[string]chan model.User
}

// NewUserChannelMap creates a new instance of UserChannelMap.
func NewUserChannelMap() *UserChannelMap {
	return &UserChannelMap{
		internal: make(map[string]chan model.User),
	}
}

// Load gets the channel for model.User from UserChannelMap given its key.
func (rm *UserChannelMap) Load(key string) (chan model.User, bool) {
	rm.RLock()
	result, ok := rm.internal[key]
	rm.RUnlock()
	return result, ok
}

// Delete deletes the channel of model.User from UserChannelMap given its key.
func (rm *UserChannelMap) Delete(key string) {
	rm.Lock()
	delete(rm.internal, key)
	rm.Unlock()
}

// Store stores a channel for given model.User at the given key in UserChannelMap.
func (rm *UserChannelMap) Store(key string, c chan model.User) {
	rm.Lock()
	rm.internal[key] = c
	rm.Unlock()
}
