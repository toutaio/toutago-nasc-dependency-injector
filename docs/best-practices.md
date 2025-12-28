# Best Practices

This guide covers best practices for using Nasc effectively in production applications.

## SOLID Principles with DI

### Single Responsibility Principle

Each service should have one reason to change:

```go
// Good - Single responsibility
type UserRepository interface {
    FindByID(id int) (*User, error)
    Save(user *User) error
}

type UserNotifier interface {
    NotifyUserCreated(user *User) error
}

// Bad - Multiple responsibilities
type UserService interface {
    FindByID(id int) (*User, error)
    Save(user *User) error
    SendEmail(user *User) error  // Email logic doesn't belong here
    LogActivity(msg string)       // Logging doesn't belong here
}
```

### Open/Closed Principle

Design for extension via interfaces:

```go
// Extensible logger
type Logger interface {
    Log(level string, message string)
}

// Can add new implementations without modifying existing code
type FileLogger struct{}
type SyslogLogger struct{}
type CloudLogger struct{}

// All implement Logger interface
container.Bind((*Logger)(nil), &FileLogger{})
```

### Liskov Substitution Principle

All implementations must be interchangeable:

```go
type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{})
}

// Both implementations work identically
container.BindSingleton((*Cache)(nil), &RedisCache{})
// OR
container.BindSingleton((*Cache)(nil), &MemoryCache{})

// Code using Cache doesn't need to change
```

### Interface Segregation Principle

Keep interfaces small and focused:

```go
// Good - Small, focused interfaces
type Reader interface {
    Read(id int) (interface{}, error)
}

type Writer interface {
    Write(data interface{}) error
}

type Deleter interface {
    Delete(id int) error
}

// Compose when needed
type Repository interface {
    Reader
    Writer
    Deleter
}

// Bad - Bloated interface
type DataService interface {
    Read(id int) (interface{}, error)
    Write(data interface{}) error
    Delete(id int) error
    ValidateSchema() error
    Migrate() error
    Backup() error
    // ... many more methods
}
```

### Dependency Inversion Principle

Depend on abstractions, not concrete types:

```go
// Good - Depends on interface
type UserService struct {
    repo   UserRepository  // Interface
    logger Logger          // Interface
}

func NewUserService(repo UserRepository, logger Logger) *UserService {
    return &UserService{repo: repo, logger: logger}
}

// Bad - Depends on concrete types
type UserService struct {
    repo   *PostgresUserRepository  // Concrete type
    logger *FileLogger               // Concrete type
}
```

## Avoiding Anti-Patterns

### Service Locator Anti-Pattern

**Don't** pass the container around:

```go
// ❌ BAD - Service Locator anti-pattern
type UserService struct {
    container nasc.Container
}

func (s *UserService) GetUser(id int) (*User, error) {
    // Looking up dependencies at runtime
    repo := s.container.Make((*UserRepository)(nil)).(UserRepository)
    return repo.FindByID(id)
}
```

**Do** inject dependencies explicitly:

```go
// ✅ GOOD - Explicit dependencies
type UserService struct {
    repo   UserRepository
    logger Logger
}

func NewUserService(repo UserRepository, logger Logger) *UserService {
    return &UserService{repo: repo, logger: logger}
}

func (s *UserService) GetUser(id int) (*User, error) {
    s.logger.Log("info", fmt.Sprintf("Getting user %d", id))
    return s.repo.FindByID(id)
}
```

### Over-Registration

Don't register everything - only register what needs to be injected:

```go
// ❌ BAD - Over-registration
container.Bind((*User)(nil), &User{})       // Don't register data models
container.Bind((*Config)(nil), config)      // Don't register config structs
container.Bind((*int)(nil), 42)             // Don't register primitives

// ✅ GOOD - Register services only
container.BindSingleton((*UserService)(nil), NewUserService)
container.BindSingleton((*Logger)(nil), &ConsoleLogger{})
container.BindScoped((*Database)(nil), &PostgresDB{})
```

## Testing Strategies

### Constructor Testing

Test constructors work correctly:

```go
func TestNewUserService(t *testing.T) {
    mockRepo := &MockUserRepository{}
    mockLogger := &MockLogger{}
    
    service := NewUserService(mockRepo, mockLogger)
    
    if service.repo != mockRepo {
        t.Error("Repository not set correctly")
    }
    if service.logger != mockLogger {
        t.Error("Logger not set correctly")
    }
}
```

### Integration Testing with Container

Test that the container wires dependencies correctly:

```go
func TestContainerConfiguration(t *testing.T) {
    container := nasc.New()
    
    // Register all dependencies
    container.BindSingleton((*Logger)(nil), &ConsoleLogger{})
    container.BindSingleton((*UserRepository)(nil), &InMemoryUserRepository{})
    container.BindConstructor((*UserService)(nil), NewUserService)
    
    // Verify resolution works
    service, err := container.TryMake((*UserService)(nil))
    if err != nil {
        t.Fatalf("Failed to resolve UserService: %v", err)
    }
    
    if service == nil {
        t.Error("Expected non-nil service")
    }
}
```

### Mock Dependencies

Use interfaces to easily mock dependencies:

```go
type MockLogger struct {
    LoggedMessages []string
}

func (m *MockLogger) Log(level, message string) {
    m.LoggedMessages = append(m.LoggedMessages, message)
}

func TestUserServiceLogging(t *testing.T) {
    container := nasc.New()
    mockLogger := &MockLogger{}
    
    container.BindSingleton((*Logger)(nil), mockLogger)
    container.BindSingleton((*UserRepository)(nil), &InMemoryUserRepository{})
    container.BindConstructor((*UserService)(nil), NewUserService)
    
    service := container.Make((*UserService)(nil)).(UserService)
    service.GetUser(42)
    
    if len(mockLogger.LoggedMessages) == 0 {
        t.Error("Expected service to log messages")
    }
}
```

## Performance Considerations

### Choose the Right Lifetime

```go
// Singleton - Best performance, use for stateless services
container.BindSingleton((*Logger)(nil), &ConsoleLogger{})
container.BindSingleton((*Config)(nil), config)

// Scoped - Good for request-scoped state
container.BindScoped((*Database)(nil), &DBConnection{})
container.BindScoped((*Transaction)(nil), &DBTransaction{})

// Transient - Use sparingly, creates new instance each time
container.BindTransient((*RequestContext)(nil), &RequestContext{})
```

### Lazy Resolution

Resolve dependencies only when needed:

```go
type AppService struct {
    // Don't resolve in constructor
    container nasc.Container  // ❌ Anti-pattern
    
    // Better: Accept dependencies directly
    logger Logger             // ✅ Good
}

// Or use lazy initialization with sync.Once
type AppService struct {
    logger     Logger
    loggerOnce sync.Once
    container  nasc.Container
}

func (s *AppService) getLogger() Logger {
    s.loggerOnce.Do(func() {
        s.logger = s.container.Make((*Logger)(nil)).(Logger)
    })
    return s.logger
}
```

### Prefer Constructor Injection Over Auto-Wiring

Constructor injection is faster and more explicit:

```go
// ✅ Faster - Constructor injection
container.BindConstructor((*UserService)(nil), NewUserService)

// ⚠️ Slower - Auto-wiring (uses reflection)
container.AutoWire(&UserService{})
```

## Organization Patterns

### Service Providers for Modules

Group related dependencies:

```go
type DatabaseProvider struct{}

func (p *DatabaseProvider) Register(c nasc.Container) error {
    c.BindSingleton((*Database)(nil), &PostgresDB{})
    c.BindScoped((*Transaction)(nil), &DBTransaction{})
    c.BindSingleton((*UserRepository)(nil), &PGUserRepository{})
    return nil
}

type LoggingProvider struct{}

func (p *LoggingProvider) Register(c nasc.Container) error {
    c.BindSingleton((*Logger)(nil), &StructuredLogger{})
    c.BindSingleton((*MetricsCollector)(nil), &PrometheusCollector{})
    return nil
}

// In main
container.RegisterProvider(&DatabaseProvider{})
container.RegisterProvider(&LoggingProvider{})
```

### Configuration Structure

Keep container configuration centralized:

```go
// config/container.go
package config

func ConfigureContainer() nasc.Container {
    container := nasc.New()
    
    // Infrastructure
    container.RegisterProvider(&DatabaseProvider{})
    container.RegisterProvider(&CacheProvider{})
    container.RegisterProvider(&LoggingProvider{})
    
    // Domain Services
    container.BindConstructor((*UserService)(nil), NewUserService)
    container.BindConstructor((*OrderService)(nil), NewOrderService)
    
    // Application Services
    container.BindConstructor((*AuthService)(nil), NewAuthService)
    
    return container
}

// main.go
func main() {
    container := config.ConfigureContainer()
    app := container.Make((*Application)(nil)).(*Application)
    app.Run()
}
```

## Common Mistakes

### Circular Dependencies

Avoid circular dependencies - refactor if needed:

```go
// ❌ BAD - Circular dependency
type ServiceA struct {
    b ServiceB
}

type ServiceB struct {
    a ServiceA  // Circular!
}

// ✅ GOOD - Extract common interface or use events
type ServiceA struct {
    eventBus EventBus
}

type ServiceB struct {
    eventBus EventBus
}
```

### Forgetting to Dispose Scopes

Always dispose scoped containers:

```go
// ❌ BAD - Memory leak
func HandleRequest(container nasc.Container) {
    scope := container.BeginScope()
    // Missing scope.Dispose() - resources leak!
    
    service := scope.Make((*RequestService)(nil))
    service.Process()
}

// ✅ GOOD - Proper cleanup
func HandleRequest(container nasc.Container) {
    scope := container.BeginScope()
    defer scope.Dispose()  // Ensures cleanup
    
    service := scope.Make((*RequestService)(nil)).(RequestService)
    service.Process()
}
```

### Not Validating at Startup

Validate container configuration at startup:

```go
func main() {
    container := config.ConfigureContainer()
    
    // Validate all registrations can be resolved
    if err := container.Validate(); err != nil {
        log.Fatal("Container configuration invalid:", err)
    }
    
    app := container.Make((*Application)(nil)).(*Application)
    app.Run()
}
```

## Summary

1. **Follow SOLID principles** - Design clean, maintainable code
2. **Inject dependencies explicitly** - Avoid service locator pattern
3. **Use appropriate lifetimes** - Singleton for stateless, scoped for requests
4. **Test thoroughly** - Unit test constructors, integration test container
5. **Organize with providers** - Group related dependencies
6. **Validate early** - Catch configuration errors at startup
7. **Clean up resources** - Always dispose scopes
8. **Keep it simple** - Don't over-engineer, register only what's needed

Following these practices will help you build robust, maintainable applications with Nasc!
