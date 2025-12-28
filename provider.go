package nasc

import (
	"fmt"
	"reflect"
)

// ServiceProvider is the interface that must be implemented by service providers.
// Service providers encapsulate related service registrations.
//
// Example:
//
//	type LoggingProvider struct{}
//
//	func (p *LoggingProvider) Register(container *Nasc) error {
//	    container.Singleton((*Logger)(nil), &ConsoleLogger{})
//	    container.Singleton((*FileLogger)(nil), &FileLoggerImpl{})
//	    return nil
//	}
type ServiceProvider interface {
	Register(container *Nasc) error
}

// BootableProvider is an optional interface for providers that need a boot phase.
// The Boot method is called after all providers have been registered.
//
// Example:
//
//	type DatabaseProvider struct{}
//
//	func (p *DatabaseProvider) Register(container *Nasc) error {
//	    container.Singleton((*Database)(nil), &PostgresDB{})
//	    return nil
//	}
//
//	func (p *DatabaseProvider) Boot(container *Nasc) error {
//	    db := container.Make((*Database)(nil)).(Database)
//	    return db.Connect() // Initialize connection
//	}
type BootableProvider interface {
	ServiceProvider
	Boot(container *Nasc) error
}

// DeferredProvider is an optional interface for providers that should be registered
// conditionally or on-demand.
//
// Example:
//
//	type CacheProvider struct{}
//
//	func (p *CacheProvider) ShouldRegister(container *Nasc) bool {
//	    // Only register if cache is enabled in config
//	    return config.CacheEnabled
//	}
type DeferredProvider interface {
	ServiceProvider
	ShouldRegister(container *Nasc) bool
}

// providerEntry tracks a registered provider.
type providerEntry struct {
	provider ServiceProvider
	booted   bool
}

// RegisterProvider registers a service provider with the container.
// The provider's Register method is called immediately.
// If the provider implements BootableProvider, its Boot method will be called
// when BootProviders() is invoked.
//
// Example:
//
//	container.RegisterProvider(&LoggingProvider{})
//	container.RegisterProvider(&DatabaseProvider{})
//	container.BootProviders() // Call boot phase
func (n *Nasc) RegisterProvider(provider ServiceProvider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	// Check if provider is deferred
	if deferred, ok := provider.(DeferredProvider); ok {
		if !deferred.ShouldRegister(n) {
			// Skip registration
			return nil
		}
	}

	// Check if already registered (by type)
	providerType := reflect.TypeOf(provider)
	for _, entry := range n.providers {
		if reflect.TypeOf(entry.provider) == providerType {
			// Already registered, skip
			return nil
		}
	}

	// Call Register method
	if err := provider.Register(n); err != nil {
		return fmt.Errorf("provider registration failed: %w", err)
	}

	// Track provider
	n.providers = append(n.providers, &providerEntry{
		provider: provider,
		booted:   false,
	})

	return nil
}

// BootProviders calls the Boot method on all registered providers that implement
// BootableProvider. This should be called after all providers have been registered.
//
// Example:
//
//	container.RegisterProvider(&DatabaseProvider{})
//	container.RegisterProvider(&CacheProvider{})
//	
//	// Boot all providers
//	if err := container.BootProviders(); err != nil {
//	    log.Fatal(err)
//	}
func (n *Nasc) BootProviders() error {
	for _, entry := range n.providers {
		if entry.booted {
			continue
		}

		if bootable, ok := entry.provider.(BootableProvider); ok {
			if err := bootable.Boot(n); err != nil {
				return fmt.Errorf("provider boot failed: %w", err)
			}
			entry.booted = true
		}
	}

	return nil
}

// GetProviders returns a list of all registered providers.
// This is useful for debugging and introspection.
func (n *Nasc) GetProviders() []ServiceProvider {
	providers := make([]ServiceProvider, len(n.providers))
	for i, entry := range n.providers {
		providers[i] = entry.provider
	}
	return providers
}
