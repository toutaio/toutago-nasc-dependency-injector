package nasc

import (
	"fmt"
	"reflect"

	"github.com/toutaio/toutago-nasc-dependency-injector/registry"
)

// ConstructorFunc represents a constructor function type.
// Supported signatures:
//   - func() *T
//   - func() (*T, error)
//   - func(Dep1) *T
//   - func(Dep1) (*T, error)
//   - func(Dep1, Dep2, ...) *T
//   - func(Dep1, Dep2, ...) (*T, error)
type ConstructorFunc interface{}

// constructorInfo holds metadata about a constructor function.
type constructorInfo struct {
	fn           reflect.Value
	fnType       reflect.Type
	paramTypes   []reflect.Type
	returnsError bool
	returnType   reflect.Type
	numParams    int
}

// parseConstructor analyzes a constructor function and extracts metadata.
func parseConstructor(constructor ConstructorFunc) (*constructorInfo, error) {
	if constructor == nil {
		return nil, fmt.Errorf("constructor cannot be nil")
	}

	fnValue := reflect.ValueOf(constructor)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("constructor must be a function, got %v", fnType.Kind())
	}

	// Validate return values
	numOut := fnType.NumOut()
	if numOut == 0 || numOut > 2 {
		return nil, fmt.Errorf("constructor must return (*T) or (*T, error), got %d return values", numOut)
	}

	// First return value must be a pointer
	returnType := fnType.Out(0)
	if returnType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("constructor must return a pointer, got %v", returnType.Kind())
	}

	// Check if second return is error
	returnsError := false
	if numOut == 2 {
		errorInterface := reflect.TypeOf((*error)(nil)).Elem()
		if !fnType.Out(1).Implements(errorInterface) {
			return nil, fmt.Errorf("constructor's second return value must be error, got %v", fnType.Out(1))
		}
		returnsError = true
	}

	// Extract parameter types
	numParams := fnType.NumIn()
	paramTypes := make([]reflect.Type, numParams)
	for i := 0; i < numParams; i++ {
		paramTypes[i] = fnType.In(i)
	}

	return &constructorInfo{
		fn:           fnValue,
		fnType:       fnType,
		paramTypes:   paramTypes,
		returnsError: returnsError,
		returnType:   returnType,
		numParams:    numParams,
	}, nil
}

// invokeConstructor calls a constructor with resolved dependencies.
func (n *Nasc) invokeConstructor(info *constructorInfo) (interface{}, error) {
	// Resolve parameters
	params := make([]reflect.Value, info.numParams)
	for i, paramType := range info.paramTypes {
		// Create type token for resolution
		var typeToken interface{}
		if paramType.Kind() == reflect.Interface {
			// For interface parameters, create nil pointer to interface
			typeToken = reflect.Zero(reflect.PtrTo(paramType)).Interface()
		} else {
			return nil, fmt.Errorf("constructor parameter %d must be an interface, got %v", i, paramType)
		}

		// Resolve dependency
		var resolved interface{}
		var resolveErr error

		func() {
			defer func() {
				if r := recover(); r != nil {
					resolveErr = fmt.Errorf("failed to resolve parameter %d: %v", i, r)
				}
			}()
			resolved = n.Make(typeToken)
		}()

		if resolveErr != nil {
			return nil, resolveErr
		}

		params[i] = reflect.ValueOf(resolved)
	}

	// Invoke constructor
	results := info.fn.Call(params)

	// Handle return values
	instance := results[0].Interface()

	if info.returnsError {
		errValue := results[1]
		if !errValue.IsNil() {
			err := errValue.Interface().(error)
			return nil, fmt.Errorf("constructor returned error: %w", err)
		}
	}

	return instance, nil
}

// BindConstructor registers a binding using a constructor function.
// The constructor function's parameters are automatically resolved from the container.
//
// Supported constructor signatures:
//   - func() *Service
//   - func() (*Service, error)
//   - func(Logger) *Service
//   - func(Logger, Database) (*Service, error)
//
// Example:
//
//	container.BindConstructor((*UserService)(nil), NewUserService)
//	// Where: func NewUserService(logger Logger, db Database) (*UserService, error)
func (n *Nasc) BindConstructor(abstractType interface{}, constructor ConstructorFunc) error {
	return n.bindConstructorWithLifetime(abstractType, constructor, LifetimeTransient)
}

// SingletonConstructor registers a singleton binding using a constructor function.
//
// Example:
//
//	container.SingletonConstructor((*Database)(nil), NewDatabase)
func (n *Nasc) SingletonConstructor(abstractType interface{}, constructor ConstructorFunc) error {
	return n.bindConstructorWithLifetime(abstractType, constructor, LifetimeSingleton)
}

// ScopedConstructor registers a scoped binding using a constructor function.
//
// Example:
//
//	container.ScopedConstructor((*UnitOfWork)(nil), NewUnitOfWork)
func (n *Nasc) ScopedConstructor(abstractType interface{}, constructor ConstructorFunc) error {
	return n.bindConstructorWithLifetime(abstractType, constructor, LifetimeScoped)
}

// bindConstructorWithLifetime is the internal method that handles constructor binding.
func (n *Nasc) bindConstructorWithLifetime(abstractType interface{}, constructor ConstructorFunc, lifetime Lifetime) error {
	if abstractType == nil {
		return &InvalidBindingError{Reason: "abstract type cannot be nil"}
	}

	// Parse constructor
	info, err := parseConstructor(constructor)
	if err != nil {
		return &InvalidBindingError{Reason: fmt.Sprintf("invalid constructor: %v", err)}
	}

	// Extract abstract type
	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	// Create binding
	binding := &registry.Binding{
		AbstractType: abstractT,
		ConcreteType: info.returnType,
		Lifetime:     string(lifetime),
		Constructor:  info, // Store constructor info
	}

	return n.registry.Register(binding)
}
