package lock

import (
	"fmt"
	"sync"
)

type Lock struct {
	mutexes *sync.Map
}

// New instantiate a new Lock
func New() *Lock {
	l := &Lock{
		mutexes: &sync.Map{},
	}
	var _ Locker = (*Lock)(nil)
	return l
}

// Lock locks the underlying mutex by key,
// and returns an unlock function. It returns an
// error if on the map exists a value for the provided
// key and it's not a sync.RWMutex
func (l *Lock) Lock(key string) (func(), error) {
	val, _ := l.mutexes.LoadOrStore(key, &sync.RWMutex{})
	mtx, ok := val.(*sync.RWMutex)
	if !ok {
		return func() {}, fmt.Errorf("the underlying map value for key %q is not a sync.RWMutex", key)
	}

	mtx.Lock()
	return func() { mtx.Unlock() }, nil
}

// RLock locks the underlying mutex for reading by key,
// and returns an unlock function. It returns an
// error if on the map exists a value for the provided
// key and it's not a sync.RWMutex
func (l *Lock) RLock(key string) (func(), error) {
	val, _ := l.mutexes.LoadOrStore(key, &sync.RWMutex{})
	mtx, ok := val.(*sync.RWMutex)
	if !ok {
		return func() {}, fmt.Errorf("the underlying map value for key %q is not a sync.RWMutex", key)
	}

	mtx.RLock()
	return func() { mtx.RUnlock() }, nil
}
