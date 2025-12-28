package nasc

// Lifetime represents the lifecycle strategy for a bound dependency.
type Lifetime string

const (
	// LifetimeTransient creates a new instance on every resolution.
	// This is the default lifetime for Bind() operations.
	LifetimeTransient Lifetime = "transient"

	// LifetimeSingleton creates a single instance that is reused for all resolutions.
	// The instance is created lazily on first resolution using sync.Once for thread safety.
	LifetimeSingleton Lifetime = "singleton"

	// LifetimeScoped creates one instance per scope.
	// Each scope maintains its own instance cache, isolated from other scopes.
	LifetimeScoped Lifetime = "scoped"

	// LifetimeFactory calls a custom factory function on every resolution.
	// The factory function receives the container for resolving dependencies.
	LifetimeFactory Lifetime = "factory"
)

// String returns the string representation of the lifetime.
func (l Lifetime) String() string {
	return string(l)
}

// FactoryFunc is a function that creates instances dynamically.
// It receives the container to resolve dependencies and returns the created instance or an error.
//
// Example:
//
//	factory := func(c *Nasc) (interface{}, error) {
//	    config := c.Make((*Config)(nil)).(*Config)
//	    return NewConnection(config.DSN), nil
//	}
//	container.Factory((*Connection)(nil), factory)
type FactoryFunc func(*Nasc) (interface{}, error)
