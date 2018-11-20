package openshift

import (
	"strconv"
	"sync"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/idler"
	"github.com/stretchr/testify/assert"
)

func Test_retrieve_idler_for_unknown_namespace_returns_nil(t *testing.T) {
	m := NewUserIdlerMap()
	idler, ok := m.Load("foo")
	assert.False(t, ok, "There should be no entry mapped")
	assert.Nil(t, idler, "No reference should be returned")
}

func Test_retrieve_idler_for_empty_namespace_returns_nil(t *testing.T) {
	m := NewUserIdlerMap()
	idler, ok := m.Load("")
	assert.False(t, ok, "There should be no entry mapped")
	assert.Nil(t, idler, "No reference should be returned")
}

func Test_len(t *testing.T) {
	m := NewUserIdlerMap()
	assert.Equal(t, 0, m.Len(), "Empty map should return 0 len")
	m.Store("foo", &idler.UserIdler{})
	assert.Equal(t, 1, m.Len(), "Len should return number of items")
}

func Test_len_mutli(t *testing.T) {
	m := NewUserIdlerMap()
	assert.Equal(t, 0, m.Len(), "Empty map should return 0 len")

	const n = 50
	wg := &sync.WaitGroup{}

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(x int) {
			defer wg.Done()

			key := strconv.Itoa(x)
			m.Store(key, &idler.UserIdler{})
		}(i)
	}

	wg.Wait()
	assert.Equal(t, n, m.Len(), "Len should return number of items")
}

// Checking store and delete
func TestUserIdlerMap_StoreDelete(t *testing.T) {
	m := NewUserIdlerMap()
	assert.Equal(t, 0, m.Len(), "Empty map should return 0 len")
	uidlerA := &idler.UserIdler{}
	uidlerB := &idler.UserIdler{}
	uidlerC := &idler.UserIdler{}

	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() {
		m.Store("foo_a", uidlerA)
		result, ok := m.Load("foo_a")
		assert.True(t, ok, "There should be an entry mapped")
		assert.Equal(t, uidlerA, result, "result retrieved should be same")
		m.Delete("foo_a")
		idler, ok := m.Load("foo_a")
		assert.False(t, ok, "There should be no entry mapped")
		assert.Nil(t, idler, "No reference should be returned")
		wg.Done()

	}()
	go func() {
		m.Store("foo_b", uidlerB)
		result, ok := m.Load("foo_b")
		assert.True(t, ok, "There should be an entry mapped")
		assert.Equal(t, uidlerB, result, "result retrieved should be same")
		wg.Done()
	}()
	go func() {
		m.Store("foo_c", uidlerC)
		result, ok := m.Load("foo_c")
		assert.True(t, ok, "There should be an entry mapped")
		assert.Equal(t, uidlerC, result, "result retrieved should be same")
		wg.Done()
	}()
	wg.Wait()
}
