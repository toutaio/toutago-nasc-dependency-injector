package nasc

import (
	"reflect"
	"sync"
)

// singletonInstance holds a singleton value and ensures it's created only once.
type singletonInstance struct {
	value interface{}
	err   error
	once  sync.Once
}

// singletonCache manages singleton instances with thread-safe lazy initialization.
type singletonCache struct {
	instances map[reflect.Type]*singletonInstance
	mu        sync.RWMutex
}

// newSingletonCache creates a new singleton cache.
func newSingletonCache() *singletonCache {
	return &singletonCache{
		instances: make(map[reflect.Type]*singletonInstance),
	}
}

// getOrCreate retrieves an existing singleton or creates it using the provided factory.
// The factory is called exactly once per type, even under concurrent access.
//
// This method is goroutine-safe.
func (sc *singletonCache) getOrCreate(abstractType reflect.Type, factory func() (interface{}, error)) (interface{}, error) {
	// Fast path: check if instance exists (read lock)
	sc.mu.RLock()
	instance, exists := sc.instances[abstractType]
	sc.mu.RUnlock()

	if !exists {
		// Slow path: create instance holder (write lock)
		sc.mu.Lock()
		// Double-check after acquiring write lock (another goroutine might have created it)
		instance, exists = sc.instances[abstractType]
		if !exists {
			instance = &singletonInstance{}
			sc.instances[abstractType] = instance
		}
		sc.mu.Unlock()
	}

	// Use sync.Once to ensure factory is called exactly once
	instance.once.Do(func() {
		instance.value, instance.err = factory()
	})

	return instance.value, instance.err
}
