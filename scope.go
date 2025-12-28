package nasc

import (
	"fmt"
	"reflect"
	"sync"
)

// Scope represents an isolated dependency resolution context.
// Scoped bindings create one instance per scope, allowing for request-scoped
// or transaction-scoped dependencies.
//
// Example:
//
//	scope := container.CreateScope()
//	defer scope.Dispose()
//	
//	// Scoped instances are unique to this scope
//	uow := scope.Make((*UnitOfWork)(nil)).(UnitOfWork)
type Scope struct {
	parent    *Nasc
	instances map[reflect.Type]interface{}
	mu        sync.RWMutex
}

// newScope creates a new scope with the given parent container.
func newScope(parent *Nasc) *Scope {
	return &Scope{
		parent:    parent,
		instances: make(map[reflect.Type]interface{}),
	}
}

// Make resolves an instance within this scope.
// Scoped bindings are cached in the scope, while singleton and factory bindings
// are delegated to the parent container.
//
// Example:
//
//	service := scope.Make((*Service)(nil)).(Service)
func (s *Scope) Make(abstractType interface{}) interface{} {
	if abstractType == nil {
		panic("cannot resolve nil type")
	}

	// Extract reflect.Type
	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	// Get binding from parent
	binding, err := s.parent.registry.Get(abstractT)
	if err != nil {
		panic(fmt.Sprintf("binding not found for type %v: %v", abstractT, err))
	}

	// Handle based on lifetime
	switch Lifetime(binding.Lifetime) {
	case LifetimeScoped:
		// Check if instance exists in scope cache
		s.mu.RLock()
		instance, exists := s.instances[abstractT]
		s.mu.RUnlock()

		if exists {
			return instance
		}

		// Create new instance for this scope
		s.mu.Lock()
		// Double-check after acquiring write lock
		instance, exists = s.instances[abstractT]
		if !exists {
			newInstance := reflect.New(binding.ConcreteType.Elem())
			instance = newInstance.Interface()
			s.instances[abstractT] = instance
		}
		s.mu.Unlock()

		return instance

	case LifetimeSingleton:
		// Delegate to parent for singleton
		return s.parent.Make(abstractType)

	case LifetimeFactory:
		// Delegate to parent for factory
		return s.parent.Make(abstractType)

	case LifetimeTransient:
		// Create new instance (don't cache)
		instance := reflect.New(binding.ConcreteType.Elem())
		return instance.Interface()

	default:
		panic(fmt.Sprintf("unknown lifetime: %v", binding.Lifetime))
	}
}

// Dispose releases resources held by this scope.
// Phase 2: This is a placeholder for Phase 8 (disposal implementation).
func (s *Scope) Dispose() error {
	// TODO: Implement disposal in Phase 8
	// For now, just clear the instance cache
	s.mu.Lock()
	s.instances = make(map[reflect.Type]interface{})
	s.mu.Unlock()
	return nil
}
