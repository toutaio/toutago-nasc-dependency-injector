// Package registry provides thread-safe storage and retrieval of dependency bindings.
package registry

import (
	"fmt"
	"reflect"
	"sync"
)

// Binding represents a mapping between an interface type and its concrete implementation.
type Binding struct {
	// AbstractType is the interface type being bound (e.g., Logger interface)
	AbstractType reflect.Type

	// ConcreteType is the implementation type (e.g., *ConsoleLogger)
	ConcreteType reflect.Type

	// Lifetime defines how instances are managed (Phase 2 feature)
	// Currently unused in Phase 1 (all bindings are transient)
	Lifetime string

	// Name is an optional identifier for named bindings (Phase 6 feature)
	Name string

	// Tags are optional labels for tagged bindings (Phase 6 feature)
	Tags []string
}

// Registry provides thread-safe storage for bindings.
// It uses a map with reflect.Type keys for O(1) lookup performance.
type Registry struct {
	bindings map[reflect.Type]*Binding
	mu       sync.RWMutex
}

// New creates a new Registry instance.
func New() *Registry {
	return &Registry{
		bindings: make(map[reflect.Type]*Binding),
	}
}

// Register stores a binding in the registry.
// Returns an error if a binding for the same type already exists.
//
// This method is goroutine-safe.
func (r *Registry) Register(binding *Binding) error {
	if binding == nil {
		return fmt.Errorf("binding cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate
	if _, exists := r.bindings[binding.AbstractType]; exists {
		return &BindingAlreadyExistsError{Type: binding.AbstractType}
	}

	r.bindings[binding.AbstractType] = binding
	return nil
}

// Get retrieves a binding by its abstract type.
// Returns the binding and nil error if found.
// Returns nil binding and error if not found.
//
// This method is goroutine-safe.
func (r *Registry) Get(abstractType reflect.Type) (*Binding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	binding, exists := r.bindings[abstractType]
	if !exists {
		return nil, &BindingNotFoundError{Type: abstractType}
	}

	return binding, nil
}

// Has checks if a binding exists for the given type.
// Returns true if the binding exists, false otherwise.
//
// This method is goroutine-safe.
func (r *Registry) Has(abstractType reflect.Type) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.bindings[abstractType]
	return exists
}

// BindingAlreadyExistsError is returned when attempting to register a duplicate binding.
type BindingAlreadyExistsError struct {
	Type reflect.Type
}

func (e *BindingAlreadyExistsError) Error() string {
	return fmt.Sprintf("binding already exists for type %v", e.Type)
}

// BindingNotFoundError is returned when a requested binding does not exist.
type BindingNotFoundError struct {
	Type reflect.Type
}

func (e *BindingNotFoundError) Error() string {
	return fmt.Sprintf("binding not found for type %v", e.Type)
}
