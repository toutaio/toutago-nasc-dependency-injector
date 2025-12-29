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
	// For factory bindings, this may be nil
	ConcreteType reflect.Type

	// Lifetime defines how instances are managed
	// Values: "transient", "singleton", "scoped", "factory"
	Lifetime string

	// Factory is the custom creation function for factory bindings
	// Only used when Lifetime is "factory"
	Factory interface{} // stores FactoryFunc

	// Constructor holds constructor function metadata
	// Phase 4 feature - stores *constructorInfo
	Constructor interface{}

	// AutoWireEnabled indicates whether to auto-wire instances after creation
	// Phase 3 feature
	AutoWireEnabled bool

	// Name is an optional identifier for named bindings (Phase 6 feature)
	Name string

	// Tags are optional labels for tagged bindings (Phase 6 feature)
	Tags []string
}

// Registry provides thread-safe storage for bindings.
// It uses a map with reflect.Type keys for O(1) lookup performance.
type Registry struct {
	mu            sync.RWMutex
	bindings      map[reflect.Type]*Binding
	namedBindings map[reflect.Type]map[string]*Binding
}

// New creates a new Registry instance.
func New() *Registry {
	return &Registry{
		bindings:      make(map[reflect.Type]*Binding),
		namedBindings: make(map[reflect.Type]map[string]*Binding),
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

// RegisterNamed stores a named binding in the registry.
// Multiple bindings of the same type can exist with different names.
//
// This method is goroutine-safe.
func (r *Registry) RegisterNamed(binding *Binding) error {
	if binding == nil {
		return fmt.Errorf("binding cannot be nil")
	}
	if binding.Name == "" {
		return fmt.Errorf("named binding must have a name")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize nested map if needed
	if r.namedBindings[binding.AbstractType] == nil {
		r.namedBindings[binding.AbstractType] = make(map[string]*Binding)
	}

	// Check for duplicate name
	if _, exists := r.namedBindings[binding.AbstractType][binding.Name]; exists {
		return fmt.Errorf("named binding '%s' for type %v already exists", binding.Name, binding.AbstractType)
	}

	r.namedBindings[binding.AbstractType][binding.Name] = binding
	return nil
}

// GetNamed retrieves a binding by type and name.
// Returns the binding and nil error if found.
// Returns nil binding and error if not found.
//
// This method is goroutine-safe.
func (r *Registry) GetNamed(abstractType reflect.Type, name string) (*Binding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	typeBindings, exists := r.namedBindings[abstractType]
	if !exists {
		return nil, &BindingNotFoundError{Type: abstractType}
	}

	binding, exists := typeBindings[name]
	if !exists {
		return nil, fmt.Errorf("named binding '%s' for type %v not found", name, abstractType)
	}

	return binding, nil
}

// GetAll returns all bindings for a given type (both named and unnamed).
// Returns empty slice if no bindings found.
//
// This method is goroutine-safe.
func (r *Registry) GetAll(abstractType reflect.Type) []*Binding {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Binding

	// Add default binding if exists
	if binding, exists := r.bindings[abstractType]; exists {
		result = append(result, binding)
	}

	// Add all named bindings
	if namedBindings, exists := r.namedBindings[abstractType]; exists {
		for _, binding := range namedBindings {
			result = append(result, binding)
		}
	}

	return result
}

// GetByTag returns all bindings that have the specified tag.
// Returns empty slice if no tagged bindings found.
//
// This method is goroutine-safe.
func (r *Registry) GetByTag(tag string) []*Binding {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Binding

	// Check unnamed bindings
	for _, binding := range r.bindings {
		if containsTag(binding.Tags, tag) {
			result = append(result, binding)
		}
	}

	// Check named bindings
	for _, namedMap := range r.namedBindings {
		for _, binding := range namedMap {
			if containsTag(binding.Tags, tag) {
				result = append(result, binding)
			}
		}
	}

	return result
}

// containsTag checks if a tag exists in a slice of tags.
func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// GetAllTypes returns all types that have bindings (named or unnamed).
func (r *Registry) GetAllTypes() []reflect.Type {
	r.mu.RLock()
	defer r.mu.RUnlock()

	typeSet := make(map[reflect.Type]bool)

	// Add unnamed binding types
	for abstractType := range r.bindings {
		typeSet[abstractType] = true
	}

	// Add named binding types
	for abstractType := range r.namedBindings {
		typeSet[abstractType] = true
	}

	// Convert to slice
	types := make([]reflect.Type, 0, len(typeSet))
	for t := range typeSet {
		types = append(types, t)
	}

	return types
}

// GetAllNamedFor returns all names for a given type.
func (r *Registry) GetAllNamedFor(abstractType reflect.Type) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	namedMap, exists := r.namedBindings[abstractType]
	if !exists {
		return nil
	}

	names := make([]string, 0, len(namedMap))
	for name := range namedMap {
		names = append(names, name)
	}

	return names
}

// HasUnnamedBinding checks if there's an unnamed binding for a type.
func (r *Registry) HasUnnamedBinding(abstractType reflect.Type) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.bindings[abstractType]
	return exists
}
