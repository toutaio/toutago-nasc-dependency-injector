# Nasc - Dependency Injection Container for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/toutaio/toutago-nasc-dependency-injector.svg)](https://pkg.go.dev/github.com/toutaio/toutago-nasc-dependency-injector)
[![Go Report Card](https://goreportcard.com/badge/github.com/toutaio/toutago-nasc-dependency-injector)](https://goreportcard.com/report/github.com/toutaio/toutago-nasc-dependency-injector)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A production-ready, high-performance dependency injection container for Go, designed with SOLID principles and best practices.

> **Nasc** (Old Irish): *Link or bond* - representing the connections between components in your application.

## ✨ Features

- 🚀 **High Performance** - Singleton resolution <100ns, transient <1μs
- 🔒 **Thread-Safe** - All operations are goroutine-safe
- 🎯 **Multiple Lifetimes** - Singleton, transient, scoped, and factory
- 🔌 **Auto-Wiring** - Automatic dependency injection via struct tags
- 🏗️ **Constructor Injection** - Type-safe constructor-based resolution
- 📦 **Service Providers** - Modular dependency registration
- 🏷️ **Named Bindings** - Multiple implementations per interface
- 🔍 **Circular Dependency Detection** - Prevents runtime errors
- 🧹 **Automatic Cleanup** - Disposable pattern for resource management
- ⚡ **Zero Dependencies** - Only uses Go standard library
- ✅ **>95% Test Coverage** - Thoroughly tested with race detection

## Installation

```bash
go get github.com/toutaio/toutago-nasc-dependency-injector
```

## 🚀 Quick Start

```go
package main

import (
	"fmt"
	nasc "github.com/toutaio/toutago-nasc-dependency-injector"
)

// Define interfaces
type Logger interface {
	Log(msg string)
}

type UserService interface {
	GetUser(id int) string
}

// Implementations
type ConsoleLogger struct{}

func (l *ConsoleLogger) Log(msg string) {
	fmt.Println("[LOG]", msg)
}

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

func main() {
	// Create container
	container := nasc.New()

	// Bind dependencies
	container.BindSingleton((*Logger)(nil), &ConsoleLogger{})
	container.BindConstructor((*UserService)(nil), NewUserService)

	// Resolve and use
	service := container.Make((*UserService)(nil)).(UserService)
	user := service.GetUser(42)
	fmt.Println("Got:", user)
}
```

**Output:**
```
[LOG] Fetching user 42
Got: User-42
```

## 📖 Core Concepts

### Lifetimes

Nasc supports four dependency lifetimes:

```go
// Transient - New instance every time
container.Bind((*Logger)(nil), &ConsoleLogger{})

// Singleton - Single shared instance
container.BindSingleton((*Cache)(nil), &RedisCache{})

// Scoped - One instance per scope
container.BindScoped((*Database)(nil), &DBConnection{})

// Factory - Custom creation logic
container.BindFactory((*Connection)(nil), func() interface{} {
	return &Connection{ID: uuid.New()}
})
```

### Constructor Injection

Type-safe dependency resolution through constructors:

```go
func NewUserService(logger Logger, repo UserRepository) *UserService {
	return &UserService{logger: logger, repo: repo}
}

container.BindConstructor((*UserService)(nil), NewUserService)
service := container.Make((*UserService)(nil)).(UserService)
```

### Auto-Wiring

Automatic injection using struct tags:

```go
type MyService struct {
	Logger   Logger   `nasc:"inject"`
	Database Database `nasc:"inject"`
	Cache    Cache    `nasc:"inject,optional"`
}

// Auto-wire creates instance with all dependencies
service := container.AutoWire(&MyService{}).(*MyService)
```

### Named Bindings

Multiple implementations for the same interface:

```go
container.BindNamed((*Logger)(nil), "file", &FileLogger{})
container.BindNamed((*Logger)(nil), "console", &ConsoleLogger{})

fileLogger := container.MakeNamed((*Logger)(nil), "file").(Logger)
consoleLogger := container.MakeNamed((*Logger)(nil), "console").(Logger)
```

### Service Providers

Organize related dependencies into modules:

```go
type DatabaseProvider struct{}

func (p *DatabaseProvider) Register(c nasc.Container) error {
	c.BindSingleton((*Database)(nil), &PostgresDB{})
	c.BindScoped((*Transaction)(nil), &DBTransaction{})
	return nil
}

container.RegisterProvider(&DatabaseProvider{})
```

### Scoping

Create scopes for request-level dependencies:

```go
func HandleRequest(container nasc.Container) {
	scope := container.BeginScope()
	defer scope.Dispose()  // Cleanup resources
	
	// All scoped dependencies share instances within this scope
	service := scope.Make((*RequestService)(nil)).(RequestService)
	service.Process()
}

## ⚡ Performance

Nasc is designed for high-performance applications:

| Operation | Time | Memory |
|-----------|------|--------|
| Singleton Resolution | <100ns | 0 allocs (cached) |
| Transient Resolution | <1μs | 24 B/op |
| Auto-Wire (typical) | <10μs | Minimal |
| Constructor Injection | <500ns | 48 B/op |

Benchmark results on AMD Ryzen 7:
```
BenchmarkSingletonResolution-16    50000000    22.5 ns/op     0 B/op    0 allocs/op
BenchmarkTransientResolution-16    12000000    95.3 ns/op    24 B/op    1 allocs/op
BenchmarkConstructorInjection-16    5000000   412.0 ns/op    48 B/op    2 allocs/op
BenchmarkAutoWire-16                 200000  8750.0 ns/op  1024 B/op   12 allocs/op
```

Run benchmarks yourself:
```bash
go test -bench=. -benchmem
```

## 🛡️ Error Handling

Nasc provides comprehensive error handling:

```go
// Safe resolution without panics
logger, err := container.TryMake((*Logger)(nil))
if err != nil {
	log.Fatal("Failed to resolve logger:", err)
}

// Circular dependency detection
// A → B → C → A
// Error: circular dependency detected: A → B → C → A

// Detailed error messages
err := container.Bind(nil, &ConsoleLogger{})
// Returns: InvalidBindingError: abstraction cannot be nil

// Validation at startup
if err := container.Validate(); err != nil {
	log.Fatal("Container configuration invalid:", err)
}
```

## 🔒 Thread Safety

All container operations are fully thread-safe:

```go
// Safe concurrent access
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
	wg.Add(1)
	go func() {
		defer wg.Done()
		service := container.Make((*MyService)(nil)).(MyService)
		service.DoWork()
	}()
}
wg.Wait()

// Verified with -race detector
// go test -race ./...
```

## 🧪 Testing

Nasc makes testing easy by allowing dependency substitution:

```go
func TestUserService(t *testing.T) {
	container := nasc.New()
	
	// Use mock implementations
	mockLogger := &MockLogger{}
	mockRepo := &MockUserRepository{}
	
	container.BindSingleton((*Logger)(nil), mockLogger)
	container.BindSingleton((*UserRepository)(nil), mockRepo)
	container.BindConstructor((*UserService)(nil), NewUserService)
	
	// Test the service
	service := container.Make((*UserService)(nil)).(UserService)
	user := service.GetUser(42)
	
	assert.Equal(t, "User-42", user)
	assert.True(t, mockLogger.WasCalled())
}
```

### Run Tests

```bash
# All tests
go test ./...

# With race detector
go test -race ./...

# With coverage
go test -cover ./...

# Benchmarks
go test -bench=. -benchmem
```

**Test Coverage:** >95%

## 📚 Documentation

- **[Getting Started](docs/getting-started.md)** - Your first steps with Nasc
- **[Best Practices](docs/best-practices.md)** - SOLID principles and patterns
- **[Examples](examples/)** - Real-world usage examples
  - [Basic Usage](examples/basic/main.go)
  - [Web Server](examples/web-server/main.go)
- **[API Reference](https://pkg.go.dev/github.com/toutaio/toutago-nasc-dependency-injector)** - Full API documentation
- **[CHANGELOG](CHANGELOG.md)** - Version history and changes

## 🗺️ Roadmap

### ✅ Completed

- **Phase 1**: Core container with basic binding/resolution
- **Phase 2**: Lifetime management (singleton, scoped, factory)
- **Phase 3**: Auto-wiring via struct tags
- **Phase 4**: Constructor injection
- **Phase 5**: Service providers
- **Phase 6**: Advanced features (named bindings, tags, conditions)
- **Phase 7**: Enhanced error handling (circular dependency detection)
- **Phase 8**: Scoping and cleanup
- **Phase 9**: Performance optimization
- **Phase 10**: Documentation and integration

### 🎯 v1.0.0 Goals

- Production-tested in Toutā framework
- API stability guarantees
- Performance benchmarks published
- Complete documentation
- Zero critical bugs

## 🏛️ Architecture

Nasc is built on **SOLID principles**:

| Principle | Implementation |
|-----------|----------------|
| **Single Responsibility** | Separate concerns: Container, Registry, Lifetime, Scope |
| **Open/Closed** | Extensible via interfaces (ServiceProvider, Disposable) |
| **Liskov Substitution** | All implementations are fully interchangeable |
| **Interface Segregation** | Small, focused interfaces (Logger, Database, etc.) |
| **Dependency Inversion** | Depend on abstractions, not concrete types |

### Design Highlights

- **Registry** - Thread-safe binding storage with RWMutex
- **Lifetime Manager** - Handles singleton/transient/scoped lifetimes
- **Reflection Cache** - Optimizes auto-wiring performance
- **Scope Hierarchy** - Parent-child relationships for cleanup
- **Error Context** - Rich error messages with dependency paths

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Quick checklist:**
- ✅ Tests pass with `-race`
- ✅ Coverage >90% for new code
- ✅ Code formatted with `gofmt`
- ✅ No `go vet` issues
- ✅ Follow SOLID principles
- ✅ Update documentation

## 📝 License

MIT License - see [LICENSE](LICENSE) for details

## 🌟 Why Nasc?

| Feature | Nasc | wire | dig | fx |
|---------|------|------|-----|-----|
| Runtime DI | ✅ | ❌ | ✅ | ✅ |
| Constructor Injection | ✅ | ✅ | ✅ | ✅ |
| Auto-Wiring | ✅ | ❌ | ❌ | ❌ |
| Named Bindings | ✅ | ❌ | ✅ | ✅ |
| Scoped Lifetimes | ✅ | ❌ | ❌ | ✅ |
| Circular Detection | ✅ | ✅ | ✅ | ✅ |
| Zero Dependencies | ✅ | ✅ | ❌ | ❌ |
| Performance | High | N/A | Medium | Medium |
| SOLID Focused | ✅ | ⚠️ | ⚠️ | ⚠️ |

## 🔗 Links

- **Repository:** https://github.com/toutaio/toutago-nasc-dependency-injector
- **Documentation:** https://pkg.go.dev/github.com/toutaio/toutago-nasc-dependency-injector
- **Issues:** https://github.com/toutaio/toutago-nasc-dependency-injector/issues
- **Discussions:** https://github.com/toutaio/toutago-nasc-dependency-injector/discussions

---

<p align="center">
  <strong>Part of the Toutā framework ecosystem 🔗</strong><br>
  Built with ❤️ using SOLID principles and Go best practices
</p>
