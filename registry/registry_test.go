package registry

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

// Test types for registry tests
type testInterface interface {
	DoSomething()
}

type testImplementation struct{}

func (t *testImplementation) DoSomething() {}

func TestNew(t *testing.T) {
	reg := New()
	if reg == nil {
		t.Fatal("New() returned nil")
	}
	if reg.bindings == nil {
		t.Error("Registry.bindings is nil")
	}
}

func TestRegister_Success(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()
	concreteType := reflect.TypeOf(&testImplementation{})

	binding := &Binding{
		AbstractType: interfaceType,
		ConcreteType: concreteType,
	}

	err := reg.Register(binding)
	if err != nil {
		t.Errorf("Register() returned error: %v", err)
	}

	// Verify binding was stored
	if !reg.Has(interfaceType) {
		t.Error("Binding not found after Register()")
	}
}

func TestRegister_Duplicate(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()
	concreteType := reflect.TypeOf(&testImplementation{})

	binding := &Binding{
		AbstractType: interfaceType,
		ConcreteType: concreteType,
	}

	// First registration should succeed
	err := reg.Register(binding)
	if err != nil {
		t.Fatalf("First Register() failed: %v", err)
	}

	// Second registration should fail
	err = reg.Register(binding)
	if err == nil {
		t.Error("Register() should return error for duplicate binding")
	}

	// Check error type
	if _, ok := err.(*BindingAlreadyExistsError); !ok {
		t.Errorf("Expected BindingAlreadyExistsError, got %T", err)
	}
}

func TestRegister_NilBinding(t *testing.T) {
	reg := New()
	err := reg.Register(nil)
	if err == nil {
		t.Error("Register(nil) should return error")
	}
}

func TestGet_Success(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()
	concreteType := reflect.TypeOf(&testImplementation{})

	expected := &Binding{
		AbstractType: interfaceType,
		ConcreteType: concreteType,
	}

	reg.Register(expected)

	got, err := reg.Get(interfaceType)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if got == nil {
		t.Fatal("Get() returned nil binding")
	}
	if got.AbstractType != expected.AbstractType {
		t.Errorf("AbstractType mismatch: got %v, want %v", got.AbstractType, expected.AbstractType)
	}
	if got.ConcreteType != expected.ConcreteType {
		t.Errorf("ConcreteType mismatch: got %v, want %v", got.ConcreteType, expected.ConcreteType)
	}
}

func TestGet_NotFound(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	got, err := reg.Get(interfaceType)
	if err == nil {
		t.Error("Get() should return error for non-existent binding")
	}
	if got != nil {
		t.Error("Get() should return nil binding when not found")
	}

	// Check error type
	if _, ok := err.(*BindingNotFoundError); !ok {
		t.Errorf("Expected BindingNotFoundError, got %T", err)
	}
}

func TestHas_ExistsAndNotExists(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()
	concreteType := reflect.TypeOf(&testImplementation{})

	// Should not exist initially
	if reg.Has(interfaceType) {
		t.Error("Has() returned true for non-existent binding")
	}

	// Register binding
	binding := &Binding{
		AbstractType: interfaceType,
		ConcreteType: concreteType,
	}
	reg.Register(binding)

	// Should exist now
	if !reg.Has(interfaceType) {
		t.Error("Has() returned false for existing binding")
	}
}

func TestConcurrentReads(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()
	concreteType := reflect.TypeOf(&testImplementation{})

	binding := &Binding{
		AbstractType: interfaceType,
		ConcreteType: concreteType,
	}
	reg.Register(binding)

	// Concurrent reads
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := reg.Get(interfaceType)
			if err != nil {
				t.Errorf("Concurrent Get() failed: %v", err)
			}
			if !reg.Has(interfaceType) {
				t.Error("Concurrent Has() returned false")
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	reg := New()
	var wg sync.WaitGroup

	// Pre-register a type for concurrent reads
	testType := reflect.TypeOf((*testInterface)(nil)).Elem()
	reg.Register(&Binding{
		AbstractType: testType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
	})

	// Concurrent writes with genuinely different types
	// Using a map to create unique type identities
	for i := 0; i < 10; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()

			// Create a struct type with unique name to ensure different types
			typeName := fmt.Sprintf("TestType%d", i)
			structType := reflect.StructOf([]reflect.StructField{
				{
					Name: "Field" + typeName,
					Type: reflect.TypeOf(""),
					Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, typeName)),
				},
			})

			binding := &Binding{
				AbstractType: structType,
				ConcreteType: reflect.TypeOf(&testImplementation{}),
			}
			err := reg.Register(binding)
			if err != nil {
				t.Errorf("Goroutine %d: Register() failed: %v", i, err)
			}
		}()
	}

	// Concurrent reads of the pre-registered type
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reg.Has(testType)
			reg.Get(testType)
		}()
	}

	wg.Wait()
}
