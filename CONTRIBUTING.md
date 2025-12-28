# CONTRIBUTING

Thank you for considering contributing to Nasc! This document provides guidelines for contributing to the project.

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in Issues
2. Create a new issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Go version and OS
   - Minimal code example

### Suggesting Features

1. Open an issue with tag `enhancement`
2. Describe the use case
3. Explain why existing features don't solve it
4. Provide example API if possible

### Submitting Pull Requests

1. **Fork the repository**
2. **Create a feature branch**
   ```bash
   git checkout -b feature/my-feature
   ```

3. **Write your code**
   - Follow Go best practices
   - Adhere to SOLID principles
   - Keep changes focused and minimal

4. **Write tests**
   - Unit tests for all new functionality
   - Aim for >90% coverage
   - Include table-driven tests where appropriate
   - Test error cases

5. **Run tests**
   ```bash
   # All tests
   go test ./...
   
   # With race detector
   go test -race ./...
   
   # With coverage
   go test -cover ./...
   ```

6. **Format and lint**
   ```bash
   go fmt ./...
   go vet ./...
   ```

7. **Commit your changes**
   ```bash
   git commit -m "feat: add feature X"
   ```
   
   Follow conventional commits:
   - `feat:` New feature
   - `fix:` Bug fix
   - `docs:` Documentation
   - `test:` Tests
   - `refactor:` Code refactoring
   - `perf:` Performance improvement

8. **Push and create PR**
   ```bash
   git push origin feature/my-feature
   ```
   
   In your PR description:
   - Explain what and why
   - Reference any related issues
   - Include before/after behavior
   - List any breaking changes

## Development Setup

```bash
# Clone the repository
git clone https://github.com/toutaio/toutago-nasc-dependency-injector.git
cd toutago-nasc-dependency-injector

# Run tests
go test ./...

# Run benchmarks
go test -bench=. -benchmem

# Check coverage
go test -cover ./...
```

## Code Standards

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use meaningful variable names
- Keep functions small and focused
- Prefer composition over inheritance

### SOLID Principles

All code must adhere to SOLID principles:

1. **Single Responsibility** - Each type has one reason to change
2. **Open/Closed** - Open for extension, closed for modification
3. **Liskov Substitution** - Implementations must be interchangeable
4. **Interface Segregation** - Small, focused interfaces
5. **Dependency Inversion** - Depend on abstractions

### Testing

- Use table-driven tests for multiple cases
- Test both success and error paths
- Include edge cases
- Use meaningful test names: `TestXxx_WhenYyy_ShouldZzz`
- Keep tests readable and maintainable

Example:
```go
func TestContainer_Make_WhenBindingExists_ShouldResolve(t *testing.T) {
    container := nasc.New()
    container.Bind((*Logger)(nil), &ConsoleLogger{})
    
    result := container.Make((*Logger)(nil))
    
    if result == nil {
        t.Error("Expected non-nil result")
    }
}
```

### Documentation

- Add godoc comments to all exported types and functions
- Include usage examples in comments
- Update README.md for new features
- Add entries to CHANGELOG.md

Example:
```go
// Bind registers a transient binding in the container.
// A new instance is created each time the binding is resolved.
//
// Example:
//   container.Bind((*Logger)(nil), &ConsoleLogger{})
//   logger := container.Make((*Logger)(nil)).(Logger)
func (c *Container) Bind(abstraction, concrete interface{}) error {
    // ...
}
```

## Project Structure

```
nasc/
├── nasc.go              # Main container implementation
├── registry/            # Internal registry package
├── errors.go            # Error types
├── lifetime.go          # Lifetime management
├── autowire.go          # Auto-wiring logic
├── constructor.go       # Constructor injection
├── provider.go          # Service providers
├── scope.go             # Scoping logic
├── singleton.go         # Singleton cache
├── options.go           # Container options
├── *_test.go            # Tests alongside implementation
├── docs/                # Documentation
└── examples/            # Usage examples
```

## Performance Guidelines

- Minimize allocations in hot paths
- Use sync.Pool for frequently allocated objects
- Avoid reflection when possible
- Run benchmarks before/after changes
- Don't optimize prematurely - measure first

```bash
# Run benchmarks
go test -bench=. -benchmem

# Profile CPU
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Profile memory
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

## Questions?

- Open an issue for questions
- Join discussions in GitHub Discussions
- Check existing documentation in `/docs`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing to Nasc! 🔗
