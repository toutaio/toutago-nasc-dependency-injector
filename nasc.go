// Package nasc provides a powerful, flexible dependency injection container for Go.
//
// Nasc (Old Irish: "Link" or "Bond") enables runtime dependency injection with
// compile-time safety through Go's type system. It supports multiple lifetime
// strategies, auto-wiring, and service providers.
//
// Basic usage:
//
//	// Create container
//	container := nasc.New()
//
//	// Bind interface to implementation
//	container.Bind((*Logger)(nil), &ConsoleLogger{})
//
//	// Resolve instance
//	logger := container.Make((*Logger)(nil)).(Logger)
//	logger.Log("Hello, Nasc!")
package nasc

import (
	"fmt"
	"reflect"

	"github.com/toutaio/toutago-nasc-dependency-injector/registry"
)

// Nasc is the main dependency injection container.
// It manages bindings and resolves dependencies in a thread-safe manner.
type Nasc struct {
	registry *registry.Registry
}

// New creates a new Nasc container instance.
// Options can be provided to configure the container behavior.
//
// Example:
//
//	container := nasc.New()
//	// or with options:
//	container := nasc.New(nasc.WithDebug())
func New(options ...Option) *Nasc {
	n := &Nasc{
		registry: registry.New(),
	}

	// Apply options
	for _, opt := range options {
		if err := opt(n); err != nil {
			panic(fmt.Sprintf("failed to apply option: %v", err))
		}
	}

	return n
}

// Bind registers a binding between an interface type and a concrete implementation.
// The abstractType should be an interface pointer like (*Logger)(nil).
// The concreteType should be a pointer to the concrete implementation.
//
// Example:
//
//	container.Bind((*Logger)(nil), &ConsoleLogger{})
//
// Returns an error if:
//   - Either parameter is nil
//   - The binding already exists
//   - The types are invalid
func (n *Nasc) Bind(abstractType, concreteType interface{}) error {
	if abstractType == nil {
		return &InvalidBindingError{Reason: "abstract type cannot be nil"}
	}
	if concreteType == nil {
		return &InvalidBindingError{Reason: "concrete type cannot be nil"}
	}

	// Extract reflect.Type from interface pointers
	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	concreteT := reflect.TypeOf(concreteType)
	// For concrete types, we expect a pointer to struct
	if concreteT.Kind() == reflect.Ptr && concreteT.Elem().Kind() == reflect.Struct {
		// Keep the pointer type for instantiation
	} else {
		return &InvalidBindingError{
			Reason: fmt.Sprintf("concrete type must be pointer to struct, got %v", concreteT),
		}
	}

	// Create binding
	binding := &registry.Binding{
		AbstractType: abstractT,
		ConcreteType: concreteT,
	}

	// Register binding
	if err := n.registry.Register(binding); err != nil {
		return err
	}

	return nil
}

// Make resolves and returns an instance of the registered type.
// The abstractType should be an interface pointer like (*Logger)(nil).
//
// Example:
//
//	logger := container.Make((*Logger)(nil)).(Logger)
//
// Phase 1 behavior: Panics if the binding is not found.
// Future phases will add MakeSafe() for error handling.
func (n *Nasc) Make(abstractType interface{}) interface{} {
	if abstractType == nil {
		panic("cannot resolve nil type")
	}

	// Extract reflect.Type
	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	// Get binding
	binding, err := n.registry.Get(abstractT)
	if err != nil {
		panic(fmt.Sprintf("binding not found for type %v: %v", abstractT, err))
	}

	// Create instance using reflection
	// ConcreteType is already a pointer type, so we create a new instance
	instance := reflect.New(binding.ConcreteType.Elem())

	return instance.Interface()
}
