// Package nasc provides a powerful, flexible dependency injection container for Go.
//
// Nasc (Old Irish: "Link" or "Bond") enables runtime dependency injection with
// compile-time safety through Go's type system. It supports multiple lifetime
// strategies, auto-wiring, constructor injection, and service providers.
//
// # Features
//
//   - Type-safe dependency injection
//   - Multiple lifetime strategies (Transient, Singleton, Scoped)
//   - Auto-wiring with reflection
//   - Constructor injection
//   - Named bindings and tags
//   - Service providers for modular configuration
//   - Circular dependency detection
//   - Thread-safe resolution
//   - Performance optimizations with caching
//
// # Quick Start
//
// Create a container and bind services:
//
//	container := nasc.New()
//	container.Bind((*Logger)(nil), &ConsoleLogger{})
//	logger := container.Make((*Logger)(nil)).(Logger)
//
// # Lifetimes
//
// Transient - New instance each time:
//
//	container.Transient((*Service)(nil), &MyService{})
//
// Singleton - Single shared instance:
//
//	container.Singleton((*Cache)(nil), &MemoryCache{})
//
// Scoped - One instance per scope:
//
//	scope := container.BeginScope()
//	defer scope.End()
//	service := scope.Make((*ScopedService)(nil))
//
// # Auto-Wiring
//
// Automatically resolve constructor dependencies:
//
//	type UserService struct {
//	    DB     Database
//	    Logger Logger
//	}
//
//	container.AutoWire((*UserService)(nil), nasc.LifetimeSingleton)
//	service := container.Make((*UserService)(nil)).(*UserService)
//
// # Named Bindings
//
// Register multiple implementations:
//
//	container.BindNamed("primary", (*DB)(nil), &PostgresDB{})
//	container.BindNamed("cache", (*DB)(nil), &RedisDB{})
//	db := container.MakeNamed("primary", (*DB)(nil)).(DB)
//
// # Service Providers
//
// Organize registrations in reusable modules:
//
//	type DatabaseProvider struct{}
//
//	func (p *DatabaseProvider) Register(c *nasc.Nasc) error {
//	    c.Singleton((*Database)(nil), &PostgresDB{})
//	    return nil
//	}
//
//	container.RegisterProvider(&DatabaseProvider{})
//
// # Error Handling
//
// Safe resolution with error checking:
//
//	service, err := container.MakeSafe((*Service)(nil))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Thread Safety
//
// All operations are thread-safe and can be used concurrently.
package nasc
