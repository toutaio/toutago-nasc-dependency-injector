# Getting Started with Nasc

Nasc is a powerful, flexible dependency injection container for Go that helps you build maintainable, testable applications by managing component dependencies automatically.

## Installation

```bash
go get github.com/toutaio/toutago-nasc-dependency-injector
```

## Your First Container

Let's build a simple application with dependency injection:

### Step 1: Define Your Interfaces

```go
package main

import "fmt"

// Logger defines how to log messages
type Logger interface {
    Log(message string)
}

// UserService defines user operations
type UserService interface {
    GetUser(id int) string
}
```

### Step 2: Create Implementations

```go
// ConsoleLogger writes to stdout
type ConsoleLogger struct{}

func (l *ConsoleLogger) Log(message string) {
    fmt.Println("[LOG]", message)
}

// DefaultUserService implements UserService
type DefaultUserService struct {
    logger Logger
}

func NewUserService(logger Logger) *DefaultUserService {
    return &DefaultUserService{logger: logger}
}

func (s *DefaultUserService) GetUser(id int) string {
    s.logger.Log(fmt.Sprintf("Fetching user %d", id))
    return fmt.Sprintf("User-%d", id)
}
```

### Step 3: Configure the Container

```go
import "github.com/toutaio/toutago-nasc-dependency-injector"

func main() {
    // Create container
    container := nasc.New()
    
    // Bind logger (singleton - shared instance)
    container.BindSingleton((*Logger)(nil), &ConsoleLogger{})
    
    // Bind user service with constructor injection
    container.BindConstructor((*UserService)(nil), NewUserService)
    
    // Resolve and use
    service := container.Make((*UserService)(nil)).(UserService)
    user := service.GetUser(42)
    fmt.Println("Got:", user)
}
```

## Lifetimes Explained

Nasc supports three dependency lifetimes:

### Transient
Creates a new instance every time:
```go
container.Bind((*Logger)(nil), &ConsoleLogger{})
// Each Make() creates a new ConsoleLogger
```

### Singleton
Creates one instance, reused across all resolutions:
```go
container.BindSingleton((*Logger)(nil), &ConsoleLogger{})
// All Make() calls return the same ConsoleLogger instance
```

### Scoped
Creates one instance per scope (useful for request-scoped dependencies):
```go
container.BindScoped((*Database)(nil), &DBConnection{})
scope := container.BeginScope()
defer scope.Dispose()
// All Make() calls within this scope return the same instance
```

## Auto-Wiring

Nasc can automatically inject dependencies using struct tags:

```go
type MyService struct {
    Logger   Logger   `nasc:"inject"`
    Database Database `nasc:"inject"`
    Cache    Cache    `nasc:"inject"`
}

// Register all dependencies
container.BindSingleton((*Logger)(nil), &ConsoleLogger{})
container.BindScoped((*Database)(nil), &PostgresDB{})
container.BindTransient((*Cache)(nil), &RedisCache{})

// Auto-wire creates the service with all dependencies
service := container.AutoWire(&MyService{}).(*MyService)
```

## Error Handling

Always check for errors when configuring the container:

```go
// Safe binding
if err := container.Bind((*Logger)(nil), &ConsoleLogger{}); err != nil {
    log.Fatal("Failed to bind logger:", err)
}

// Safe resolution
logger, err := container.TryMake((*Logger)(nil))
if err != nil {
    log.Fatal("Failed to resolve logger:", err)
}
```

## Testing

Nasc makes testing easy - swap implementations for mocks:

```go
func TestUserService(t *testing.T) {
    container := nasc.New()
    
    // Bind mock logger
    mockLogger := &MockLogger{}
    container.BindSingleton((*Logger)(nil), mockLogger)
    
    // Bind service under test
    container.BindConstructor((*UserService)(nil), NewUserService)
    
    // Test the service
    service := container.Make((*UserService)(nil)).(UserService)
    user := service.GetUser(42)
    
    if user != "User-42" {
        t.Errorf("Expected User-42, got %s", user)
    }
    
    if !mockLogger.WasCalled() {
        t.Error("Expected logger to be called")
    }
}
```

## Next Steps

- Learn about [Core Concepts](core-concepts.md)
- Explore [Lifetimes](lifetimes.md) in detail
- Master [Auto-Wiring](auto-wiring.md)
- Review [Best Practices](best-practices.md)
- See [Examples](../examples/) for real-world usage

## Common Patterns

### Factory Pattern
```go
container.BindFactory((*Connection)(nil), func() interface{} {
    return &Connection{ID: uuid.New()}
})
```

### Named Bindings
```go
container.BindNamed((*Logger)(nil), "file", &FileLogger{})
container.BindNamed((*Logger)(nil), "console", &ConsoleLogger{})

fileLogger := container.MakeNamed((*Logger)(nil), "file").(Logger)
```

### Service Providers
```go
type DatabaseProvider struct{}

func (p *DatabaseProvider) Register(c nasc.Container) error {
    c.BindSingleton((*Database)(nil), &PostgresDB{})
    c.BindScoped((*Transaction)(nil), &DBTransaction{})
    return nil
}

container.RegisterProvider(&DatabaseProvider{})
```

## Tips

1. **Prefer constructor injection** - Makes dependencies explicit
2. **Use interfaces** - Enables flexibility and testing
3. **Bind in main()** - Keep configuration centralized
4. **Use singletons for stateless services** - Better performance
5. **Use scoped for request-specific state** - Prevents memory leaks

Ready to dive deeper? Check out the [Core Concepts](core-concepts.md) guide!
