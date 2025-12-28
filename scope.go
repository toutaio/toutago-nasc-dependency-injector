package nasc

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/toutaio/toutago-nasc-dependency-injector/registry"
)

// Disposable represents a service that requires cleanup.
// Services implementing this interface will have Dispose called
// when their scope is disposed.
//
// Example:
//
//	type DatabaseConnection struct {}
//	func (d *DatabaseConnection) Dispose() error {
//	    return d.connection.Close()
//	}
type Disposable interface {
	Dispose() error
}

// Initializable represents a service that requires initialization.
// Services implementing this interface will have Initialize called
// after being created.
//
// Example:
//
//	type Service struct {}
//	func (s *Service) Initialize() error {
//	    return s.setup()
//	}
type Initializable interface {
	Initialize() error
}

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
	parent         *Nasc
	instances      map[reflect.Type]interface{}
	creationOrder  []interface{} // Track order for reverse disposal
	children       []*Scope
	disposed       bool
	mu             sync.RWMutex
}

// newScope creates a new scope with the given parent container.
func newScope(parent *Nasc) *Scope {
	return &Scope{
		parent:        parent,
		instances:     make(map[reflect.Type]interface{}),
		creationOrder: make([]interface{}, 0),
		children:      make([]*Scope, 0),
		disposed:      false,
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

	s.mu.RLock()
	if s.disposed {
		s.mu.RUnlock()
		panic("cannot resolve from disposed scope")
	}
	s.mu.RUnlock()

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
			instance = s.createInstance(binding, abstractT)
			s.instances[abstractT] = instance
			s.creationOrder = append(s.creationOrder, instance)
		}
		s.mu.Unlock()

		// Initialize if implements Initializable
		if initializable, ok := instance.(Initializable); ok {
			if err := initializable.Initialize(); err != nil {
				panic(fmt.Sprintf("failed to initialize instance of type %v: %v", abstractT, err))
			}
		}

		return instance

	case LifetimeSingleton:
		// Delegate to parent for singleton
		return s.parent.Make(abstractType)

	case LifetimeFactory:
		// Delegate to parent for factory
		return s.parent.Make(abstractType)

	case LifetimeTransient:
		// Create new instance (don't cache)
		instance := s.createInstance(binding, abstractT)
		
		// Initialize if implements Initializable
		if initializable, ok := instance.(Initializable); ok {
			if err := initializable.Initialize(); err != nil {
				panic(fmt.Sprintf("failed to initialize instance of type %v: %v", abstractT, err))
			}
		}
		
		return instance

	default:
		panic(fmt.Sprintf("unknown lifetime: %v", binding.Lifetime))
	}
}

// createInstance creates a new instance from a binding
func (s *Scope) createInstance(binding *registry.Binding, abstractT reflect.Type) interface{} {
	if binding.Constructor != nil {
		info := binding.Constructor.(*constructorInfo)
		instance, err := s.parent.invokeConstructor(info)
		if err != nil {
			panic(fmt.Sprintf("failed to invoke constructor for type %v: %v", abstractT, err))
		}
		return instance
	}
	instance := reflect.New(binding.ConcreteType.Elem())
	return instance.Interface()
}

// CreateChildScope creates a child scope that inherits parent registrations.
// Child scopes are automatically disposed when the parent is disposed.
//
// Example:
//
//	parentScope := container.CreateScope()
//	defer parentScope.Dispose()
//	
//	childScope := parentScope.CreateChildScope()
//	// Child will be disposed with parent
func (s *Scope) CreateChildScope() *Scope {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.disposed {
		panic("cannot create child scope from disposed scope")
	}

	child := newScope(s.parent)
	s.children = append(s.children, child)
	return child
}

// Dispose releases resources held by this scope.
// Calls Dispose() on all instances implementing Disposable interface
// in reverse creation order (dependencies disposed before dependents).
// Also disposes all child scopes first.
//
// Example:
//
//	scope := container.CreateScope()
//	defer scope.Dispose()
func (s *Scope) Dispose() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.disposed {
		return nil // Already disposed
	}

	var errors []error

	// First, dispose all child scopes
	for _, child := range s.children {
		if err := child.Dispose(); err != nil {
			errors = append(errors, fmt.Errorf("child scope disposal error: %w", err))
		}
	}
	s.children = nil

	// Dispose instances in reverse creation order
	for i := len(s.creationOrder) - 1; i >= 0; i-- {
		instance := s.creationOrder[i]
		if disposable, ok := instance.(Disposable); ok {
			if err := disposable.Dispose(); err != nil {
				errors = append(errors, fmt.Errorf("disposal error for %T: %w", instance, err))
			}
		}
	}

	// Clear instance cache and creation order
	s.instances = make(map[reflect.Type]interface{})
	s.creationOrder = nil
	s.disposed = true

	if len(errors) > 0 {
		return fmt.Errorf("scope disposal encountered %d error(s): %v", len(errors), errors)
	}

	return nil
}
