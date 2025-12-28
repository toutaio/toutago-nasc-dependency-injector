package nasc

import (
	"fmt"
	"reflect"
)

// BindingNotFoundError is returned when a requested binding does not exist.
type BindingNotFoundError struct {
	Type reflect.Type
}

func (e *BindingNotFoundError) Error() string {
	return fmt.Sprintf("binding not found for type %v. Did you forget to register it with Bind()?", e.Type)
}

// BindingAlreadyExistsError is returned when attempting to register a duplicate binding.
type BindingAlreadyExistsError struct {
	Type reflect.Type
}

func (e *BindingAlreadyExistsError) Error() string {
	return fmt.Sprintf("binding already exists for type %v. Use a different binding or remove the existing one first.", e.Type)
}

// InvalidBindingError is returned when a binding has invalid parameters.
type InvalidBindingError struct {
	Reason string
}

func (e *InvalidBindingError) Error() string {
	return fmt.Sprintf("invalid binding: %s", e.Reason)
}

// ResolutionError is returned when instance resolution fails.
type ResolutionError struct {
	Type  reflect.Type
	Cause error
}

func (e *ResolutionError) Error() string {
	return fmt.Sprintf("failed to resolve type %v: %v", e.Type, e.Cause)
}

// Unwrap returns the underlying cause error.
func (e *ResolutionError) Unwrap() error {
	return e.Cause
}
