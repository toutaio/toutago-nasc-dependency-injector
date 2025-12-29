package nasc

import (
	"errors"
	"testing"
)

// Test providers

type BasicProvider struct {
	registerCalled bool
}

func (p *BasicProvider) Register(container *Nasc) error {
	p.registerCalled = true
	container.Bind((*Logger)(nil), &ConsoleLogger{})
	return nil
}

type BootableTestProvider struct {
	registerCalled bool
	bootCalled     bool
}

func (p *BootableTestProvider) Register(container *Nasc) error {
	p.registerCalled = true
	container.Bind((*Database)(nil), &MockDB{})
	return nil
}

func (p *BootableTestProvider) Boot(container *Nasc) error {
	p.bootCalled = true
	// Simulate initialization
	db := container.Make((*Database)(nil)).(Database)
	return db.Connect()
}

type FailingProvider struct{}

func (p *FailingProvider) Register(container *Nasc) error {
	return errors.New("registration failed")
}

type FailingBootProvider struct{}

func (p *FailingBootProvider) Register(container *Nasc) error {
	return nil
}

func (p *FailingBootProvider) Boot(container *Nasc) error {
	return errors.New("boot failed")
}

type DeferredTestProvider struct {
	shouldRegister bool
	registerCalled bool
}

func (p *DeferredTestProvider) ShouldRegister(container *Nasc) bool {
	return p.shouldRegister
}

func (p *DeferredTestProvider) Register(container *Nasc) error {
	p.registerCalled = true
	return nil
}

type CompositeProvider struct{}

func (p *CompositeProvider) Register(container *Nasc) error {
	// Register other providers
	container.RegisterProvider(&BasicProvider{})
	return nil
}

type LoggingProvider struct{}

func (p *LoggingProvider) Register(container *Nasc) error {
	container.Singleton((*Logger)(nil), &ConsoleLogger{})
	return nil
}

type DatabaseProvider struct {
	bootCalled bool
}

func (p *DatabaseProvider) Register(container *Nasc) error {
	// Constructor that needs Logger
	NewDB := func(logger Logger) *MockDB {
		return &MockDB{}
	}
	container.SingletonConstructor((*Database)(nil), NewDB)
	return nil
}

func (p *DatabaseProvider) Boot(container *Nasc) error {
	p.bootCalled = true
	db := container.Make((*Database)(nil)).(Database)
	return db.Connect()
}

// Tests

func TestRegisterProvider_Basic(t *testing.T) {
	container := New()
	provider := &BasicProvider{}

	err := container.RegisterProvider(provider)
	if err != nil {
		t.Fatalf("RegisterProvider failed: %v", err)
	}

	if !provider.registerCalled {
		t.Error("Provider.Register() was not called")
	}

	// Verify binding was registered
	logger := container.Make((*Logger)(nil))
	if logger == nil {
		t.Error("Provider did not register binding")
	}
}

func TestRegisterProvider_Bootable(t *testing.T) {
	container := New()
	provider := &BootableTestProvider{}

	err := container.RegisterProvider(provider)
	if err != nil {
		t.Fatalf("RegisterProvider failed: %v", err)
	}

	if !provider.registerCalled {
		t.Error("Provider.Register() was not called")
	}

	if provider.bootCalled {
		t.Error("Provider.Boot() called too early")
	}

	// Boot providers
	err = container.BootProviders()
	if err != nil {
		t.Fatalf("BootProviders failed: %v", err)
	}

	if !provider.bootCalled {
		t.Error("Provider.Boot() was not called")
	}
}

func TestRegisterProvider_Nil(t *testing.T) {
	container := New()

	err := container.RegisterProvider(nil)
	if err == nil {
		t.Error("Expected error when registering nil provider")
	}
}

func TestRegisterProvider_FailingRegistration(t *testing.T) {
	container := New()
	provider := &FailingProvider{}

	err := container.RegisterProvider(provider)
	if err == nil {
		t.Error("Expected error from failing provider")
	}
}

func TestBootProviders_FailingBoot(t *testing.T) {
	container := New()
	provider := &FailingBootProvider{}

	container.RegisterProvider(provider)

	err := container.BootProviders()
	if err == nil {
		t.Error("Expected error from failing boot")
	}
}

func TestRegisterProvider_Duplicate(t *testing.T) {
	container := New()
	provider1 := &BasicProvider{}
	provider2 := &BasicProvider{}

	err := container.RegisterProvider(provider1)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Second registration of same type should be skipped
	err = container.RegisterProvider(provider2)
	if err != nil {
		t.Fatalf("Second registration failed: %v", err)
	}

	// Should only have one provider
	providers := container.GetProviders()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	// First provider should be registered, second should be skipped
	if !provider1.registerCalled {
		t.Error("First provider not registered")
	}
	if provider2.registerCalled {
		t.Error("Second provider should have been skipped")
	}
}

func TestRegisterProvider_Deferred_Registered(t *testing.T) {
	container := New()
	provider := &DeferredTestProvider{shouldRegister: true}

	err := container.RegisterProvider(provider)
	if err != nil {
		t.Fatalf("RegisterProvider failed: %v", err)
	}

	if !provider.registerCalled {
		t.Error("Deferred provider should have been registered")
	}
}

func TestRegisterProvider_Deferred_Skipped(t *testing.T) {
	container := New()
	provider := &DeferredTestProvider{shouldRegister: false}

	err := container.RegisterProvider(provider)
	if err != nil {
		t.Fatalf("RegisterProvider failed: %v", err)
	}

	if provider.registerCalled {
		t.Error("Deferred provider should have been skipped")
	}
}

func TestRegisterProvider_Composite(t *testing.T) {
	container := New()
	provider := &CompositeProvider{}

	err := container.RegisterProvider(provider)
	if err != nil {
		t.Fatalf("RegisterProvider failed: %v", err)
	}

	// Composite provider should have registered BasicProvider
	providers := container.GetProviders()
	if len(providers) < 2 {
		t.Errorf("Expected at least 2 providers, got %d", len(providers))
	}

	// Logger should be available (registered by BasicProvider)
	logger := container.Make((*Logger)(nil))
	if logger == nil {
		t.Error("Nested provider did not register binding")
	}
}

func TestBootProviders_MultipleProviders(t *testing.T) {
	container := New()

	provider1 := &BootableTestProvider{}
	provider2 := &BootableTestProvider{}

	container.RegisterProvider(provider1)
	container.RegisterProvider(provider2)

	err := container.BootProviders()
	if err != nil {
		t.Fatalf("BootProviders failed: %v", err)
	}

	if !provider1.bootCalled {
		t.Error("Provider1 not booted")
	}
	// provider2 is duplicate type, should not be registered
}

func TestBootProviders_Idempotent(t *testing.T) {
	container := New()
	provider := &BootableTestProvider{}

	container.RegisterProvider(provider)

	// Boot once
	err := container.BootProviders()
	if err != nil {
		t.Fatalf("First boot failed: %v", err)
	}

	provider.bootCalled = false

	// Boot again - should not call Boot again
	err = container.BootProviders()
	if err != nil {
		t.Fatalf("Second boot failed: %v", err)
	}

	if provider.bootCalled {
		t.Error("Provider booted twice")
	}
}

func TestGetProviders(t *testing.T) {
	container := New()

	if len(container.GetProviders()) != 0 {
		t.Error("New container should have no providers")
	}

	container.RegisterProvider(&BasicProvider{})
	container.RegisterProvider(&BootableTestProvider{})

	providers := container.GetProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}
}

// Integration test
func TestProvider_RealWorldScenario(t *testing.T) {
	container := New()

	// Register providers
	container.RegisterProvider(&LoggingProvider{})
	container.RegisterProvider(&DatabaseProvider{})

	// Boot
	err := container.BootProviders()
	if err != nil {
		t.Fatalf("Boot failed: %v", err)
	}

	// Use services
	logger := container.Make((*Logger)(nil)).(Logger)
	db := container.Make((*Database)(nil)).(Database)

	if logger == nil {
		t.Error("Logger not available")
	}
	if db == nil {
		t.Error("Database not available")
	}

	mockDB := db.(*MockDB)
	if !mockDB.connected {
		t.Error("Database not connected during boot")
	}
}
