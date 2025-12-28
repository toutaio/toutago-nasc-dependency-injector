package nasc

import (
	"fmt"
	"reflect"
	"strings"
)

// tagOptions represents parsed options from an inject tag.
type tagOptions struct {
	skip     bool   // Don't inject this field
	optional bool   // Don't panic if binding not found
	name     string // Named binding to use
}

// parseInjectTag parses an inject struct tag and returns options.
// Supported formats:
//   - `inject:""` - basic injection
//   - `inject:"optional"` - optional injection
//   - `inject:"name=foo"` - named binding
//   - `inject:"optional,name=foo"` - combined options
func parseInjectTag(tag string) tagOptions {
	opts := tagOptions{}

	if tag == "" {
		return opts
	}

	if tag == "-" {
		opts.skip = true
		return opts
	}

	// Split by comma for multiple options
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part == "optional" {
			opts.optional = true
		} else if strings.HasPrefix(part, "name=") {
			opts.name = strings.TrimPrefix(part, "name=")
		}
	}

	return opts
}

// fieldInfo holds metadata about a field to inject.
type fieldInfo struct {
	field       reflect.StructField
	fieldValue  reflect.Value
	options     tagOptions
	fieldType   reflect.Type
	isInterface bool
}

// getInjectableFields scans a struct and returns fields that need injection.
func getInjectableFields(structValue reflect.Value) []fieldInfo {
	var fields []fieldInfo

	structType := structValue.Type()
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
		structValue = structValue.Elem()
	}

	if structType.Kind() != reflect.Struct {
		return fields
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)

		// Check for inject tag
		tag, ok := field.Tag.Lookup("inject")
		if !ok {
			continue
		}

		opts := parseInjectTag(tag)
		if opts.skip {
			continue
		}

		// Store field info
		info := fieldInfo{
			field:       field,
			fieldValue:  fieldValue,
			options:     opts,
			fieldType:   field.Type,
			isInterface: field.Type.Kind() == reflect.Interface,
		}

		fields = append(fields, info)
	}

	return fields
}

// AutoWire automatically injects dependencies into tagged struct fields.
// Fields with `inject` tags will be resolved from the container.
//
// Supported tag options:
//   - `inject:""` - basic injection (panics if not found)
//   - `inject:"optional"` - optional (skips if not found)
//   - `inject:"name=foo"` - uses named binding
//
// Example:
//
//	type Service struct {
//	    Logger   Logger   `inject:""`
//	    Cache    Cache    `inject:"optional"`
//	    FileLog  Logger   `inject:"name=file"`
//	}
//
//	service := &Service{}
//	container.AutoWire(service)
func (n *Nasc) AutoWire(instance interface{}) error {
	if instance == nil {
		return fmt.Errorf("cannot auto-wire nil instance")
	}

	value := reflect.ValueOf(instance)
	if value.Kind() != reflect.Ptr {
		return fmt.Errorf("AutoWire requires a pointer to struct, got %T", instance)
	}

	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("AutoWire requires a pointer to struct, got pointer to %v", elem.Kind())
	}

	// Get fields that need injection
	fields := getInjectableFields(value)

	// Inject each field
	for _, field := range fields {
		if err := n.injectField(field); err != nil {
			return fmt.Errorf("failed to inject field %s: %w", field.field.Name, err)
		}
	}

	return nil
}

// injectField injects a single field.
func (n *Nasc) injectField(field fieldInfo) error {
	if !field.fieldValue.CanSet() {
		return fmt.Errorf("field %s is not settable (not exported?)", field.field.Name)
	}

	// For now, Phase 3 doesn't support named bindings (that's Phase 6)
	// We'll just resolve by type
	if field.options.name != "" {
		return fmt.Errorf("named bindings not yet supported (Phase 6 feature)")
	}

	// Create type token for resolution
	var typeToken interface{}
	if field.isInterface {
		// For interface fields, we need to create a nil pointer to the interface type
		typeToken = reflect.Zero(reflect.PtrTo(field.fieldType)).Interface()
	} else {
		return fmt.Errorf("only interface fields are supported for injection, got %v", field.fieldType)
	}

	// Try to resolve
	var resolved interface{}
	var resolveErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				resolveErr = fmt.Errorf("resolution panicked: %v", r)
			}
		}()
		resolved = n.Make(typeToken)
	}()

	// Handle resolution failure
	if resolveErr != nil {
		if field.options.optional {
			// Optional field, skip injection
			return nil
		}
		return resolveErr
	}

	// Set the field value
	resolvedValue := reflect.ValueOf(resolved)
	if !resolvedValue.Type().AssignableTo(field.fieldType) {
		return fmt.Errorf("resolved type %v is not assignable to field type %v",
			resolvedValue.Type(), field.fieldType)
	}

	field.fieldValue.Set(resolvedValue)
	return nil
}

// autoWireInstance is a helper that auto-wires an instance if auto-wiring is enabled.
// This is called internally after creating instances.
func (n *Nasc) autoWireInstance(instance interface{}, autoWireEnabled bool) error {
	if !autoWireEnabled {
		return nil
	}
	return n.AutoWire(instance)
}
