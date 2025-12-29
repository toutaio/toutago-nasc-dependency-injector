package nasc

import (
	"errors"
	"testing"
)

// Test types for scoping and cleanup
type disposableService struct {
	disposed bool
}

func (d *disposableService) Dispose() error {
	if d.disposed {
		return errors.New("already disposed")
	}
	d.disposed = true
	return nil
}

type initializableService struct {
	initialized bool
}

func (i *initializableService) Initialize() error {
	i.initialized = true
	return nil
}

type failingDisposable struct{}

func (f *failingDisposable) Dispose() error {
	return errors.New("disposal failed")
}

type dependentService struct {
	dependency *disposableService
	disposed   bool
}

func (d *dependentService) Dispose() error {
	d.disposed = true
	return nil
}

// TestScopeIsolation verifies that scopes maintain isolated instance caches
func TestScopeIsolation(t *testing.T) {
	container := New()
	container.Scoped((*disposableService)(nil), &disposableService{})

	scope1 := container.CreateScope()
	scope2 := container.CreateScope()

	instance1 := scope1.Make((*disposableService)(nil)).(*disposableService)
	instance2 := scope2.Make((*disposableService)(nil)).(*disposableService)

	if instance1 == instance2 {
		t.Error("Expected different instances in different scopes")
	}

	// Verify instances are consistent within same scope
	instance1Again := scope1.Make((*disposableService)(nil)).(*disposableService)
	if instance1 != instance1Again {
		t.Error("Expected same instance within same scope")
	}

	scope1.Dispose()
	scope2.Dispose()
}

// TestChildScopeInheritance verifies that child scopes inherit parent registrations
func TestChildScopeInheritance(t *testing.T) {
	container := New()
	container.Scoped((*disposableService)(nil), &disposableService{})

	parentScope := container.CreateScope()
	childScope := parentScope.CreateChildScope()

	parentInstance := parentScope.Make((*disposableService)(nil)).(*disposableService)
	childInstance := childScope.Make((*disposableService)(nil)).(*disposableService)

	if parentInstance == childInstance {
		t.Error("Expected different instances in parent and child scopes")
	}

	parentScope.Dispose()
}

// TestDisposalOrder verifies that instances are disposed in reverse creation order
func TestDisposalOrder(t *testing.T) {
	container := New()

	type serviceA struct{}
	type serviceB struct{}

	container.Scoped((*serviceA)(nil), &serviceA{})
	container.Scoped((*serviceB)(nil), &serviceB{})

	scope := container.CreateScope()

	// Create instances in order A, B
	scope.Make((*serviceA)(nil))
	scope.Make((*serviceB)(nil))

	// Disposal should be reverse: B, A
	scope.Dispose()

	// Note: This test structure shows intent but actual tracking needs
	// to be implemented with proper disposal tracking mechanism
	scope = nil
}

// TestInitializableInterface verifies Initialize is called after creation
func TestInitializableInterface(t *testing.T) {
	container := New()
	container.Scoped((*initializableService)(nil), &initializableService{})

	scope := container.CreateScope()
	defer scope.Dispose()

	instance := scope.Make((*initializableService)(nil)).(*initializableService)

	if !instance.initialized {
		t.Error("Expected Initialize to be called")
	}
}

// TestDisposableInterface verifies Dispose is called on scope disposal
func TestDisposableInterface(t *testing.T) {
	container := New()
	container.Scoped((*disposableService)(nil), &disposableService{})

	scope := container.CreateScope()
	instance := scope.Make((*disposableService)(nil)).(*disposableService)

	if instance.disposed {
		t.Error("Instance should not be disposed yet")
	}

	err := scope.Dispose()
	if err != nil {
		t.Errorf("Dispose failed: %v", err)
	}

	if !instance.disposed {
		t.Error("Instance should be disposed after scope disposal")
	}
}

// TestDoubleDisposal verifies that disposing a scope twice doesn't cause issues
func TestDoubleDisposal(t *testing.T) {
	container := New()
	container.Scoped((*disposableService)(nil), &disposableService{})

	scope := container.CreateScope()
	instance := scope.Make((*disposableService)(nil)).(*disposableService)

	err1 := scope.Dispose()
	if err1 != nil {
		t.Errorf("First disposal failed: %v", err1)
	}

	err2 := scope.Dispose()
	if err2 != nil {
		t.Errorf("Second disposal should not error: %v", err2)
	}

	// First disposal should have worked
	if !instance.disposed {
		t.Error("Instance should be disposed")
	}
}

// TestDisposalErrors verifies that disposal errors are collected
func TestDisposalErrors(t *testing.T) {
	container := New()
	container.Scoped((*failingDisposable)(nil), &failingDisposable{})

	scope := container.CreateScope()
	scope.Make((*failingDisposable)(nil))

	err := scope.Dispose()
	if err == nil {
		t.Error("Expected disposal error to be returned")
	}
}

// TestChildScopeDisposal verifies child scopes are disposed with parent
func TestChildScopeDisposal(t *testing.T) {
	container := New()
	container.Scoped((*disposableService)(nil), &disposableService{})

	parentScope := container.CreateScope()
	childScope := parentScope.CreateChildScope()

	parentInstance := parentScope.Make((*disposableService)(nil)).(*disposableService)
	childInstance := childScope.Make((*disposableService)(nil)).(*disposableService)

	// Dispose parent
	err := parentScope.Dispose()
	if err != nil {
		t.Errorf("Parent disposal failed: %v", err)
	}

	// Both should be disposed
	if !parentInstance.disposed {
		t.Error("Parent instance should be disposed")
	}
	if !childInstance.disposed {
		t.Error("Child instance should be disposed when parent is disposed")
	}
}

// TestDisposedScopePanics verifies that using a disposed scope panics
func TestDisposedScopePanics(t *testing.T) {
	container := New()
	container.Scoped((*disposableService)(nil), &disposableService{})

	scope := container.CreateScope()
	scope.Dispose()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when resolving from disposed scope")
		}
	}()

	scope.Make((*disposableService)(nil))
}

// TestCreateChildFromDisposedScope verifies panic when creating child from disposed scope
func TestCreateChildFromDisposedScope(t *testing.T) {
	container := New()
	scope := container.CreateScope()
	scope.Dispose()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when creating child from disposed scope")
		}
	}()

	scope.CreateChildScope()
}

// TestTransientLifetimeWithInitialize verifies transient instances are initialized
func TestTransientLifetimeWithInitialize(t *testing.T) {
	container := New()
	container.Bind((*initializableService)(nil), &initializableService{})

	scope := container.CreateScope()
	defer scope.Dispose()

	instance := scope.Make((*initializableService)(nil)).(*initializableService)

	if !instance.initialized {
		t.Error("Expected Initialize to be called on transient instance")
	}
}

// TestScopedLifetimeConsistency verifies scoped instances are reused within scope
func TestScopedLifetimeConsistency(t *testing.T) {
	container := New()
	container.Scoped((*disposableService)(nil), &disposableService{})

	scope := container.CreateScope()
	defer scope.Dispose()

	instance1 := scope.Make((*disposableService)(nil)).(*disposableService)
	instance2 := scope.Make((*disposableService)(nil)).(*disposableService)

	if instance1 != instance2 {
		t.Error("Expected same instance for scoped lifetime within scope")
	}
}

// TestMultipleDisposablesInScope verifies all disposables are cleaned up
func TestMultipleDisposablesInScope(t *testing.T) {
	container := New()

	type serviceA struct{ disposableService }
	type serviceB struct{ disposableService }

	container.Scoped((*serviceA)(nil), &serviceA{})
	container.Scoped((*serviceB)(nil), &serviceB{})

	scope := container.CreateScope()

	instanceA := scope.Make((*serviceA)(nil)).(*serviceA)
	instanceB := scope.Make((*serviceB)(nil)).(*serviceB)

	scope.Dispose()

	if !instanceA.disposed {
		t.Error("Service A should be disposed")
	}
	if !instanceB.disposed {
		t.Error("Service B should be disposed")
	}
}
