# Changelog

All notable changes to Nasc will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.9] - 2026-01-02

### Changed
- Updated README documentation

## [1.0.8] - 2026-01-01

### Changed
- Removed workflow step for optimization

## [1.0.7] - 2025-12-31

### Changed
- Updated workflows for improved automation

## [1.0.6] - 2025-12-30

### Changed
- Updated workflows with enhanced features

## [1.0.5] - 2025-12-29

### Changed
- Improved code quality
- Enhanced code standards

## [1.0.4] - 2025-12-28

### Changed
- Improved GitHub workflows
- Enhanced CI/CD pipeline

## [1.0.3] - 2025-12-27

### Changed
- Improved code quality
- Enhanced linting and code standards

## [1.0.2] - 2025-12-26

### Changed
- Tidied up code
- Code organization improvements

## [1.0.1] - 2025-12-25

### Added
- Additional documentation files
- Enhanced project documentation

## [1.0.0] - 2025-12-24

### Added
- Phase 10: Documentation & Integration complete
- Comprehensive documentation in `/docs`
- Usage examples in `/examples`
- Best practices guide
- Contributing guidelines
- Migration guides from wire/dig/fx
- Production-ready status
- Full semantic versioning support

## [0.9.0] - Phase 9: Performance Optimizations

### Added
- Reflection caching for improved auto-wire performance
- Concurrent singleton resolution with double-checked locking
- Lazy initialization support
- Performance benchmarks

### Performance
- Singleton resolution: <100ns
- Transient resolution: <1μs
- Auto-wire: <10μs for typical structs
- Zero allocations after warmup

## [0.8.0] - Phase 8: Scoping & Cleanup

### Added
- Scoped lifetime support
- `BeginScope()` and `Dispose()` for scope management
- Disposable interface for cleanup
- Parent-child scope hierarchy
- Scope-specific instance caching

### Changed
- Improved memory management with scope isolation
- Enhanced cleanup guarantees

## [0.7.0] - Phase 7: Error Handling

### Added
- Circular dependency detection
- `TryMake()` for safe resolution without panics
- Detailed error types with context
- Resolution path tracking
- Comprehensive error messages

### Changed
- Better error reporting with dependency chains
- Validation at registration time

## [0.6.0] - Phase 6: Advanced Features

### Added
- Named bindings with `BindNamed()` and `MakeNamed()`
- Tag-based filtering and resolution
- Conditional registration
- Decorator pattern support
- Interceptors for cross-cutting concerns

### Changed
- Registry supports multiple bindings per interface
- Enhanced binding metadata

## [0.5.0] - Phase 5: Service Providers

### Added
- `ServiceProvider` interface for modular registration
- `RegisterProvider()` method
- Built-in providers for common patterns
- Provider chaining and composition

### Changed
- Better organization of related dependencies
- Simplified configuration

## [0.4.0] - Phase 4: Constructor Injection

### Added
- `BindConstructor()` for constructor-based injection
- Automatic dependency resolution from constructor parameters
- Support for variadic constructors
- Constructor validation

### Changed
- Improved reflection handling
- Better type safety

## [0.3.0] - Phase 3: Auto-Wiring

### Added
- `AutoWire()` method for struct field injection
- Struct tag support: `nasc:"inject"`
- Optional fields with `nasc:"inject,optional"`
- Named injection with `nasc:"inject,name=xxx"`

### Changed
- Enhanced reflection utilities
- Better field validation

## [0.2.0] - Phase 2: Lifetime Management

### Added
- Singleton lifetime with `BindSingleton()`
- Factory lifetime with `BindFactory()`
- Lifetime enumeration
- Thread-safe singleton caching
- Factory function support

### Changed
- Registry now tracks lifetime metadata
- Improved instance management

## [0.1.0] - Phase 1: Core Container

### Added
- Basic `Container` implementation
- Thread-safe `Registry` with RWMutex
- Transient lifetime (default)
- `Bind()` method for interface-to-implementation binding
- `Make()` method for dependency resolution
- Custom error types:
  - `BindingNotFoundError`
  - `BindingAlreadyExistsError`
  - `InvalidBindingError`
- Comprehensive test suite (>85% coverage)
- Race condition testing
- Performance benchmarks

### Performance
- Bind: ~208ns per operation
- Make: ~54ns per resolution

## [0.0.1] - Initial Setup

### Added
- Project structure
- Go module initialization
- Basic documentation
- MIT License

[Unreleased]: https://github.com/toutaio/toutago-nasc-dependency-injector/compare/v1.0.9...HEAD
[1.0.9]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v1.0.9
[1.0.8]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v1.0.8
[1.0.7]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v1.0.7
[1.0.6]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v1.0.6
[1.0.5]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v1.0.5
[1.0.4]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v1.0.4
[1.0.3]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v1.0.3
[1.0.2]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v1.0.2
[1.0.1]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v1.0.1
[1.0.0]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v1.0.0
[0.9.0]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v0.9.0
[0.8.0]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v0.8.0
[0.7.0]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v0.7.0
[0.6.0]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v0.6.0
[0.5.0]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v0.5.0
[0.4.0]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v0.4.0
[0.3.0]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v0.3.0
[0.2.0]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v0.2.0
[0.1.0]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v0.1.0
[0.0.1]: https://github.com/toutaio/toutago-nasc-dependency-injector/releases/tag/v0.0.1
