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
	registry        *registry.Registry
	singletonCache  *singletonCache
	reflectionCache *reflectionCache
	providers       []*providerEntry
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
		registry:        registry.New(),
		singletonCache:  newSingletonCache(),
		reflectionCache: newReflectionCache(),
		providers:       make([]*providerEntry, 0),
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
		Lifetime:     string(LifetimeTransient),
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
// The resolution behavior depends on the binding's lifetime:
//   - Transient: Creates a new instance every time
//   - Singleton: Returns the same instance (created lazily on first call)
//   - Factory: Calls the factory function to create an instance
//   - Scoped: Panics (scoped bindings must use Scope.Make())
//
// Example:
//
//	logger := container.Make((*Logger)(nil)).(Logger)
//
// Phase 1-2 behavior: Panics if the binding is not found.
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

	// Resolve based on lifetime
	switch Lifetime(binding.Lifetime) {
	case LifetimeTransient:
		// Check if this is a constructor binding
		if binding.Constructor != nil {
			info := binding.Constructor.(*constructorInfo)
			instance, err := n.invokeConstructor(info)
			if err != nil {
				panic(fmt.Sprintf("failed to invoke constructor for type %v: %v", abstractT, err))
			}
			return instance
		}
		// Create new instance using reflection
		instance := reflect.New(binding.ConcreteType.Elem())
		return instance.Interface()

	case LifetimeSingleton:
		// Get or create singleton
		instance, err := n.singletonCache.getOrCreate(abstractT, func() (interface{}, error) {
			// Check if this is a constructor binding
			if binding.Constructor != nil {
				info := binding.Constructor.(*constructorInfo)
				return n.invokeConstructor(info)
			}
			// Use reflection
			newInstance := reflect.New(binding.ConcreteType.Elem())
			return newInstance.Interface(), nil
		})
		if err != nil {
			panic(fmt.Sprintf("failed to create singleton for type %v: %v", abstractT, err))
		}
		return instance

	case LifetimeFactory:
		// Call factory function
		factory, ok := binding.Factory.(FactoryFunc)
		if !ok {
			panic(fmt.Sprintf("invalid factory function for type %v", abstractT))
		}
		instance, err := factory(n)
		if err != nil {
			panic(fmt.Sprintf("factory function failed for type %v: %v", abstractT, err))
		}
		return instance

	case LifetimeScoped:
		// Scoped bindings must be resolved through Scope.Make()
		panic(fmt.Sprintf("scoped binding for type %v must be resolved using Scope.Make(), not container.Make()", abstractT))

	default:
		panic(fmt.Sprintf("unknown lifetime %s for type %v", binding.Lifetime, abstractT))
	}
}

// Singleton registers a singleton binding.
// The instance is created lazily on first resolution and reused for all subsequent resolutions.
// Singleton creation is thread-safe using sync.Once.
//
// Example:
//
// container.Singleton((*Database)(nil), &PostgresDB{})
// db1 := container.Make((*Database)(nil)).(Database)
// db2 := container.Make((*Database)(nil)).(Database)
// // db1 == db2 (same instance)
func (n *Nasc) Singleton(abstractType, concreteType interface{}) error {
	if abstractType == nil {
		return &InvalidBindingError{Reason: "abstract type cannot be nil"}
	}
	if concreteType == nil {
		return &InvalidBindingError{Reason: "concrete type cannot be nil"}
	}

	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	concreteT := reflect.TypeOf(concreteType)
	if concreteT.Kind() == reflect.Ptr && concreteT.Elem().Kind() == reflect.Struct {
		// Valid pointer to struct
	} else {
		return &InvalidBindingError{
			Reason: fmt.Sprintf("concrete type must be pointer to struct, got %v", concreteT),
		}
	}

	binding := &registry.Binding{
		AbstractType: abstractT,
		ConcreteType: concreteT,
		Lifetime:     string(LifetimeSingleton),
	}

	return n.registry.Register(binding)
}

// Scoped registers a scoped binding.
// One instance is created per scope. Scoped bindings must be resolved using Scope.Make().
//
// Example:
//
// container.Scoped((*UnitOfWork)(nil), &DbUnitOfWork{})
// scope := container.CreateScope()
// uow := scope.Make((*UnitOfWork)(nil)).(UnitOfWork)
func (n *Nasc) Scoped(abstractType, concreteType interface{}) error {
	if abstractType == nil {
		return &InvalidBindingError{Reason: "abstract type cannot be nil"}
	}
	if concreteType == nil {
		return &InvalidBindingError{Reason: "concrete type cannot be nil"}
	}

	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	concreteT := reflect.TypeOf(concreteType)
	if concreteT.Kind() == reflect.Ptr && concreteT.Elem().Kind() == reflect.Struct {
		// Valid pointer to struct
	} else {
		return &InvalidBindingError{
			Reason: fmt.Sprintf("concrete type must be pointer to struct, got %v", concreteT),
		}
	}

	binding := &registry.Binding{
		AbstractType: abstractT,
		ConcreteType: concreteT,
		Lifetime:     string(LifetimeScoped),
	}

	return n.registry.Register(binding)
}

// Factory registers a factory binding.
// The factory function is called on every resolution to create instances.
//
// Example:
//
//	container.Factory((*Connection)(nil), func(c *Nasc) (interface{}, error) {
//	   config := c.Make((*Config)(nil)).(*Config)
//	   return NewConnection(config.DSN), nil
//	})
func (n *Nasc) Factory(abstractType interface{}, factory FactoryFunc) error {
	if abstractType == nil {
		return &InvalidBindingError{Reason: "abstract type cannot be nil"}
	}
	if factory == nil {
		return &InvalidBindingError{Reason: "factory function cannot be nil"}
	}

	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	binding := &registry.Binding{
		AbstractType: abstractT,
		ConcreteType: nil, // Factory doesn't have a concrete type
		Lifetime:     string(LifetimeFactory),
		Factory:      factory,
	}

	return n.registry.Register(binding)
}

// CreateScope creates a new dependency resolution scope.
// Scoped bindings create one instance per scope.
//
// Example:
//
// scope := container.CreateScope()
// defer scope.Dispose()
// uow := scope.Make((*UnitOfWork)(nil)).(UnitOfWork)
func (n *Nasc) CreateScope() *Scope {
	return newScope(n)
}

// BindNamed registers a named binding.
// Named bindings allow multiple implementations of the same interface.
//
// Example:
//
// container.BindNamed((*Logger)(nil), &FileLogger{}, "file")
// container.BindNamed((*Logger)(nil), &ConsoleLogger{}, "console")
//
// fileLogger := container.MakeNamed((*Logger)(nil), "file").(Logger)
func (n *Nasc) BindNamed(abstractType, concreteType interface{}, name string) error {
	if abstractType == nil {
		return &InvalidBindingError{Reason: "abstract type cannot be nil"}
	}
	if concreteType == nil {
		return &InvalidBindingError{Reason: "concrete type cannot be nil"}
	}
	if name == "" {
		return &InvalidBindingError{Reason: "name cannot be empty"}
	}

	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	concreteT := reflect.TypeOf(concreteType)
	if concreteT.Kind() == reflect.Ptr && concreteT.Elem().Kind() == reflect.Struct {
		// Valid pointer to struct
	} else {
		return &InvalidBindingError{
			Reason: fmt.Sprintf("concrete type must be pointer to struct, got %v", concreteT),
		}
	}

	binding := &registry.Binding{
		AbstractType: abstractT,
		ConcreteType: concreteT,
		Lifetime:     string(LifetimeTransient),
		Name:         name,
	}

	return n.registry.RegisterNamed(binding)
}

// MakeNamed resolves and returns a named instance.
//
// Example:
//
// logger := container.MakeNamed((*Logger)(nil), "file").(Logger)
func (n *Nasc) MakeNamed(abstractType interface{}, name string) interface{} {
	if abstractType == nil {
		panic("cannot resolve nil type")
	}
	if name == "" {
		panic("name cannot be empty")
	}

	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	binding, err := n.registry.GetNamed(abstractT, name)
	if err != nil {
		panic(fmt.Sprintf("named binding '%s' not found for type %v: %v", name, abstractT, err))
	}

	// Create instance based on binding type
	return n.createInstanceFromBinding(binding, abstractT)
}

// MakeAll resolves and returns all implementations of an interface.
// This includes both named and unnamed bindings.
//
// Example:
//
// loggers := container.MakeAll((*Logger)(nil))
//
//	for _, logger := range loggers {
//	   logger.(Logger).Log("message")
//	}
func (n *Nasc) MakeAll(abstractType interface{}) []interface{} {
	if abstractType == nil {
		panic("cannot resolve nil type")
	}

	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	bindings := n.registry.GetAll(abstractT)
	instances := make([]interface{}, 0, len(bindings))

	for _, binding := range bindings {
		instance := n.createInstanceFromBinding(binding, abstractT)
		instances = append(instances, instance)
	}

	return instances
}

// BindWithTags registers a binding with tags.
// Tags enable grouping and batch resolution of related services.
//
// Example:
//
// container.BindWithTags((*Plugin)(nil), &PluginA{}, []string{"plugin", "enabled"})
// container.BindWithTags((*Plugin)(nil), &PluginB{}, []string{"plugin", "enabled"})
//
// plugins := container.MakeWithTag("plugin")
func (n *Nasc) BindWithTags(abstractType, concreteType interface{}, tags []string) error {
	if abstractType == nil {
		return &InvalidBindingError{Reason: "abstract type cannot be nil"}
	}
	if concreteType == nil {
		return &InvalidBindingError{Reason: "concrete type cannot be nil"}
	}

	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	concreteT := reflect.TypeOf(concreteType)
	if concreteT.Kind() == reflect.Ptr && concreteT.Elem().Kind() == reflect.Struct {
		// Valid pointer to struct
	} else {
		return &InvalidBindingError{
			Reason: fmt.Sprintf("concrete type must be pointer to struct, got %v", concreteT),
		}
	}

	binding := &registry.Binding{
		AbstractType: abstractT,
		ConcreteType: concreteT,
		Lifetime:     string(LifetimeTransient),
		Tags:         tags,
	}

	// Tagged bindings need unique names to avoid conflicts
	binding.Name = fmt.Sprintf("_tag_%s_%p", tags[0], concreteType)
	return n.registry.RegisterNamed(binding)
}

// MakeWithTag resolves all instances with the specified tag.
//
// Example:
//
// plugins := container.MakeWithTag("plugin")
func (n *Nasc) MakeWithTag(tag string) []interface{} {
	if tag == "" {
		panic("tag cannot be empty")
	}

	bindings := n.registry.GetByTag(tag)
	instances := make([]interface{}, 0, len(bindings))

	for _, binding := range bindings {
		instance := n.createInstanceFromBinding(binding, binding.AbstractType)
		instances = append(instances, instance)
	}

	return instances
}

// createInstanceFromBinding creates an instance from a binding.
// This centralizes instance creation logic for reuse.
func (n *Nasc) createInstanceFromBinding(binding *registry.Binding, abstractT reflect.Type) interface{} {
	switch Lifetime(binding.Lifetime) {
	case LifetimeTransient:
		return n.createTransientInstance(binding, abstractT)
	case LifetimeSingleton:
		return n.createSingletonInstance(binding, abstractT)
	case LifetimeFactory:
		return n.createFactoryInstance(binding, abstractT)
	default:
		panic(fmt.Sprintf("unknown lifetime %s for type %v", binding.Lifetime, abstractT))
	}
}

// createTransientInstance creates a new transient instance
func (n *Nasc) createTransientInstance(binding *registry.Binding, abstractT reflect.Type) interface{} {
	instance := n.createRawInstance(binding)

	// Auto-wire if enabled
	if binding.AutoWireEnabled {
		if err := n.AutoWire(instance); err != nil {
			panic(fmt.Sprintf("failed to auto-wire instance for type %v: %v", abstractT, err))
		}
	}

	return instance
}

// createSingletonInstance creates or retrieves a singleton instance
func (n *Nasc) createSingletonInstance(binding *registry.Binding, abstractT reflect.Type) interface{} {
	// For named/tagged singletons, use name as cache key
	cacheKey := abstractT
	if binding.Name != "" {
		// Create unique key combining type and name
		cacheKey = reflect.TypeOf(struct {
			t reflect.Type
			n string
		}{abstractT, binding.Name})
	}

	instance, err := n.singletonCache.getOrCreate(cacheKey, func() (interface{}, error) {
		inst := n.createRawInstance(binding)

		// Auto-wire if enabled
		if binding.AutoWireEnabled {
			if err := n.AutoWire(inst); err != nil {
				return nil, err
			}
		}

		return inst, nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create singleton for type %v: %v", abstractT, err))
	}
	return instance
}

// createFactoryInstance creates an instance using a factory function
func (n *Nasc) createFactoryInstance(binding *registry.Binding, abstractT reflect.Type) interface{} {
	factory, ok := binding.Factory.(FactoryFunc)
	if !ok {
		panic(fmt.Sprintf("invalid factory function for type %v", abstractT))
	}
	instance, err := factory(n)
	if err != nil {
		panic(fmt.Sprintf("factory function failed for type %v: %v", abstractT, err))
	}
	return instance
}

// createRawInstance creates an instance using constructor or reflection
func (n *Nasc) createRawInstance(binding *registry.Binding) interface{} {
	if binding.Constructor != nil {
		info := binding.Constructor.(*constructorInfo)
		inst, err := n.invokeConstructor(info)
		if err != nil {
			panic(fmt.Sprintf("failed to invoke constructor: %v", err))
		}
		return inst
	}
	return reflect.New(binding.ConcreteType.Elem()).Interface()
}

// resolutionContext tracks the current resolution path for circular dependency detection.
type resolutionContext struct {
	stack []string
	seen  map[string]bool
}

// newResolutionContext creates a new resolution context.
func newResolutionContext() *resolutionContext {
	return &resolutionContext{
		stack: make([]string, 0),
		seen:  make(map[string]bool),
	}
}

// push adds a type to the resolution stack.
func (rc *resolutionContext) push(typeName string) error {
	if rc.seen[typeName] {
		// Circular dependency detected
		rc.stack = append(rc.stack, typeName)
		return &CircularDependencyError{Path: rc.stack}
	}
	rc.seen[typeName] = true
	rc.stack = append(rc.stack, typeName)
	return nil
}

// pop removes the last type from the resolution stack.
func (rc *resolutionContext) pop() {
	if len(rc.stack) > 0 {
		last := rc.stack[len(rc.stack)-1]
		delete(rc.seen, last)
		rc.stack = rc.stack[:len(rc.stack)-1]
	}
}

// MakeSafe resolves and returns an instance without panicking.
// Returns (instance, nil) on success or (nil, error) on failure.
//
// Example:
//
// logger, err := container.MakeSafe((*Logger)(nil))
//
//	if err != nil {
//	   return fmt.Errorf("failed to get logger: %w", err)
//	}
func (n *Nasc) MakeSafe(abstractType interface{}) (interface{}, error) {
	if abstractType == nil {
		return nil, &InvalidBindingError{Reason: "cannot resolve nil type"}
	}

	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	ctx := newResolutionContext()
	return n.makeSafeWithContext(abstractT, "", ctx)
}

// MakeNamedSafe resolves a named instance without panicking.
func (n *Nasc) MakeNamedSafe(abstractType interface{}, name string) (interface{}, error) {
	if abstractType == nil {
		return nil, &InvalidBindingError{Reason: "cannot resolve nil type"}
	}
	if name == "" {
		return nil, &InvalidBindingError{Reason: "name cannot be empty"}
	}

	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	ctx := newResolutionContext()
	return n.makeSafeWithContext(abstractT, name, ctx)
}

// makeSafeWithContext performs safe resolution with circular dependency detection.
func (n *Nasc) makeSafeWithContext(abstractT reflect.Type, name string, ctx *resolutionContext) (interface{}, error) {
	// Build type key for tracking
	typeKey := abstractT.String()
	if name != "" {
		typeKey = fmt.Sprintf("%s[%s]", typeKey, name)
	}

	// Check for circular dependency
	if err := ctx.push(typeKey); err != nil {
		return nil, err
	}
	defer ctx.pop()

	// Get binding
	var binding *registry.Binding
	var err error

	if name != "" {
		binding, err = n.registry.GetNamed(abstractT, name)
	} else {
		binding, err = n.registry.Get(abstractT)
	}

	if err != nil {
		return nil, &ResolutionError{
			Type:  abstractT,
			Name:  name,
			Cause: err,
		}
	}

	// Create instance
	return n.createInstanceSafe(binding, abstractT, ctx)
}

// createInstanceSafe creates an instance safely with context.
func (n *Nasc) createInstanceSafe(binding *registry.Binding, abstractT reflect.Type, ctx *resolutionContext) (interface{}, error) {
	switch Lifetime(binding.Lifetime) {
	case LifetimeTransient:
		if binding.Constructor != nil {
			info := binding.Constructor.(*constructorInfo)
			return n.invokeConstructorSafe(info, ctx)
		}
		instance := reflect.New(binding.ConcreteType.Elem())
		return instance.Interface(), nil

	case LifetimeSingleton:
		cacheKey := abstractT
		if binding.Name != "" {
			cacheKey = reflect.TypeOf(struct {
				t reflect.Type
				n string
			}{abstractT, binding.Name})
		}

		// For singletons, we need to handle potential circular deps in factory
		instance, err := n.singletonCache.getOrCreate(cacheKey, func() (interface{}, error) {
			if binding.Constructor != nil {
				info := binding.Constructor.(*constructorInfo)
				return n.invokeConstructorSafe(info, ctx)
			}
			newInstance := reflect.New(binding.ConcreteType.Elem())
			return newInstance.Interface(), nil
		})
		return instance, err

	case LifetimeFactory:
		factory, ok := binding.Factory.(FactoryFunc)
		if !ok {
			return nil, &ResolutionError{
				Type:    abstractT,
				Context: "invalid factory function",
			}
		}
		return factory(n)

	default:
		return nil, &ResolutionError{
			Type:    abstractT,
			Context: fmt.Sprintf("unknown lifetime: %s", binding.Lifetime),
		}
	}
}

// invokeConstructorSafe invokes a constructor safely with circular detection.
func (n *Nasc) invokeConstructorSafe(info *constructorInfo, ctx *resolutionContext) (interface{}, error) {
	params := make([]reflect.Value, len(info.paramTypes))

	for i, paramType := range info.paramTypes {
		// Resolve parameter with context
		param, err := n.makeSafeWithContext(paramType, "", ctx)
		if err != nil {
			return nil, &ResolutionError{
				Type:    info.returnType,
				Context: fmt.Sprintf("failed to resolve constructor parameter %d (%v)", i, paramType),
				Cause:   err,
			}
		}
		params[i] = reflect.ValueOf(param)
	}

	// Call constructor
	results := info.fn.Call(params)

	// Handle error return
	if len(results) == 2 {
		if !results[1].IsNil() {
			err := results[1].Interface().(error)
			return nil, &ResolutionError{
				Type:    info.returnType,
				Context: "constructor returned error",
				Cause:   err,
			}
		}
	}

	return results[0].Interface(), nil
}

// Validate checks the container's bindings for potential issues.
// Returns nil if validation passes, or ValidationError with all found issues.
//
// Example:
//
//	if err := container.Validate(); err != nil {
//	   log.Fatalf("Container validation failed: %v", err)
//	}
func (n *Nasc) Validate() error {
	var validationErrors []error

	// Get all types
	allTypes := n.registry.GetAllTypes()

	// Try to resolve each type
	for _, abstractType := range allTypes {
		// Try unnamed binding if exists
		if n.registry.HasUnnamedBinding(abstractType) {
			ctx := newResolutionContext()
			_, err := n.makeSafeWithContext(abstractType, "", ctx)
			if err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("binding %v: %w", abstractType, err))
			}
		}

		// Try all named bindings for this type
		names := n.registry.GetAllNamedFor(abstractType)
		for _, name := range names {
			ctx := newResolutionContext()
			_, err := n.makeSafeWithContext(abstractType, name, ctx)
			if err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("binding %v[%s]: %w", abstractType, name, err))
			}
		}
	}

	if len(validationErrors) > 0 {
		return &ValidationError{Errors: validationErrors}
	}

	return nil
}

// BindAutoWire registers a binding with automatic dependency injection enabled.
// The instance will have its fields with `inject` tags automatically resolved.
//
// Example:
//
//	type Service struct {
//	    Logger Logger `inject:""`
//	}
//	container.BindAutoWire((*ServiceInterface)(nil), &Service{})
func (n *Nasc) BindAutoWire(abstractType, concreteType interface{}) error {
	if abstractType == nil {
		return &InvalidBindingError{Reason: "abstract type cannot be nil"}
	}
	if concreteType == nil {
		return &InvalidBindingError{Reason: "concrete type cannot be nil"}
	}

	abstractT := reflect.TypeOf(abstractType)
	if abstractT.Kind() == reflect.Ptr {
		abstractT = abstractT.Elem()
	}

	concreteT := reflect.TypeOf(concreteType)
	if concreteT.Kind() == reflect.Ptr && concreteT.Elem().Kind() == reflect.Struct {
		// Valid pointer to struct
	} else {
		return &InvalidBindingError{
			Reason: fmt.Sprintf("concrete type must be pointer to struct, got %v", concreteT),
		}
	}

	binding := &registry.Binding{
		AbstractType:    abstractT,
		ConcreteType:    concreteT,
		Lifetime:        string(LifetimeTransient),
		AutoWireEnabled: true,
	}

	return n.registry.Register(binding)
}

// MustMake is an explicit panic version of Make for cases where panic is desired.
// This is useful to distinguish intentional panics from unintended ones.
func (n *Nasc) MustMake(abstractType interface{}) interface{} {
	instance, err := n.MakeSafe(abstractType)
	if err != nil {
		panic(err)
	}
	return instance
}
