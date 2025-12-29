package nasc

import (
	"fmt"
	"reflect"
	"strings"
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
	Type    reflect.Type
	Name    string
	Cause   error
	Context string
}

func (e *ResolutionError) Error() string {
	typeStr := "unknown"
	if e.Type != nil {
		typeStr = e.Type.String()
	}

	nameStr := ""
	if e.Name != "" {
		nameStr = fmt.Sprintf(" (name=%s)", e.Name)
	}

	contextStr := ""
	if e.Context != "" {
		contextStr = fmt.Sprintf(": %s", e.Context)
	}

	causeStr := ""
	if e.Cause != nil {
		causeStr = fmt.Sprintf(": %v", e.Cause)
	}

	return fmt.Sprintf("failed to resolve %s%s%s%s", typeStr, nameStr, contextStr, causeStr)
}

// Unwrap returns the underlying cause error.
func (e *ResolutionError) Unwrap() error {
	return e.Cause
}

// CircularDependencyError indicates a circular dependency was detected.
type CircularDependencyError struct {
	Path []string
}

func (e *CircularDependencyError) Error() string {
	if len(e.Path) == 0 {
		return "circular dependency detected"
	}
	return fmt.Sprintf("circular dependency detected: %s", strings.Join(e.Path, " -> "))
}

// ValidationError indicates a problem found during binding validation.
type ValidationError struct {
	Errors []error
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("validation failed: %v", e.Errors[0])
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("validation failed with %d errors:\n", len(e.Errors)))
	for i, err := range e.Errors {
		b.WriteString(fmt.Sprintf("  %d. %v\n", i+1, err))
	}
	return b.String()
}

func (e *ValidationError) Unwrap() []error {
	return e.Errors
}
