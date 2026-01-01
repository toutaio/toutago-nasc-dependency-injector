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

	_ = reg.Register(expected)

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
	_ = reg.Register(binding)

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
	_ = reg.Register(binding)

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
	_ = reg.Register(&Binding{
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
					Tag:  reflect.StructTag(fmt.Sprintf("json:%q", typeName)),
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
			_, _ = reg.Get(testType)
		}()
	}

	wg.Wait()
}

func TestRegisterNamed_Success(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()
	concreteType := reflect.TypeOf(&testImplementation{})

	binding := &Binding{
		AbstractType: interfaceType,
		ConcreteType: concreteType,
		Name:         "myImplementation",
	}

	err := reg.RegisterNamed(binding)
	if err != nil {
		t.Errorf("RegisterNamed() returned error: %v", err)
	}
}

func TestRegisterNamed_NilBinding(t *testing.T) {
	reg := New()
	err := reg.RegisterNamed(nil)
	if err == nil {
		t.Error("RegisterNamed(nil) should return error")
	}
}

func TestRegisterNamed_EmptyName(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()
	concreteType := reflect.TypeOf(&testImplementation{})

	binding := &Binding{
		AbstractType: interfaceType,
		ConcreteType: concreteType,
		Name:         "",
	}

	err := reg.RegisterNamed(binding)
	if err == nil {
		t.Error("RegisterNamed() should return error for empty name")
	}
}

func TestRegisterNamed_Duplicate(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()
	concreteType := reflect.TypeOf(&testImplementation{})

	binding := &Binding{
		AbstractType: interfaceType,
		ConcreteType: concreteType,
		Name:         "duplicate",
	}

	err := reg.RegisterNamed(binding)
	if err != nil {
		t.Fatalf("First RegisterNamed() failed: %v", err)
	}

	err = reg.RegisterNamed(binding)
	if err == nil {
		t.Error("RegisterNamed() should return error for duplicate name")
	}
}

func TestGetNamed_Success(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()
	concreteType := reflect.TypeOf(&testImplementation{})

	expected := &Binding{
		AbstractType: interfaceType,
		ConcreteType: concreteType,
		Name:         "myImplementation",
	}

	_ = reg.RegisterNamed(expected)

	got, err := reg.GetNamed(interfaceType, "myImplementation")
	if err != nil {
		t.Fatalf("GetNamed() returned error: %v", err)
	}
	if got == nil {
		t.Fatal("GetNamed() returned nil binding")
	}
	if got.Name != expected.Name {
		t.Errorf("Name mismatch: got %v, want %v", got.Name, expected.Name)
	}
}

func TestGetNamed_NotFound(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	_, err := reg.GetNamed(interfaceType, "nonexistent")
	if err == nil {
		t.Error("GetNamed() should return error for non-existent binding")
	}
}

func TestGetNamed_TypeNotFound(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()
	otherType := reflect.TypeOf("")

	binding := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
		Name:         "test",
	}
	_ = reg.RegisterNamed(binding)

	_, err := reg.GetNamed(otherType, "test")
	if err == nil {
		t.Error("GetNamed() should return error for non-existent type")
	}
}

func TestGetAll_UnnamedOnly(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	binding := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
	}
	_ = reg.Register(binding)

	result := reg.GetAll(interfaceType)
	if len(result) != 1 {
		t.Errorf("GetAll() returned %d bindings, want 1", len(result))
	}
}

func TestGetAll_NamedOnly(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	binding1 := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
		Name:         "impl1",
	}
	binding2 := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
		Name:         "impl2",
	}

	_ = reg.RegisterNamed(binding1)
	_ = reg.RegisterNamed(binding2)

	result := reg.GetAll(interfaceType)
	if len(result) != 2 {
		t.Errorf("GetAll() returned %d bindings, want 2", len(result))
	}
}

func TestGetAll_MixedUnnamedAndNamed(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	unnamed := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
	}
	named := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
		Name:         "named",
	}

	_ = reg.Register(unnamed)
	_ = reg.RegisterNamed(named)

	result := reg.GetAll(interfaceType)
	if len(result) != 2 {
		t.Errorf("GetAll() returned %d bindings, want 2", len(result))
	}
}

func TestGetAll_NotFound(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	result := reg.GetAll(interfaceType)
	if len(result) != 0 {
		t.Errorf("GetAll() returned %d bindings, want 0", len(result))
	}
}

func TestGetByTag_Found(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	binding1 := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
		Tags:         []string{"tag1", "tag2"},
	}
	binding2 := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
		Tags:         []string{"tag2", "tag3"},
		Name:         "named",
	}

	_ = reg.Register(binding1)
	_ = reg.RegisterNamed(binding2)

	result := reg.GetByTag("tag2")
	if len(result) != 2 {
		t.Errorf("GetByTag() returned %d bindings, want 2", len(result))
	}
}

func TestGetByTag_NotFound(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	binding := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
		Tags:         []string{"tag1"},
	}
	_ = reg.Register(binding)

	result := reg.GetByTag("nonexistent")
	if len(result) != 0 {
		t.Errorf("GetByTag() returned %d bindings, want 0", len(result))
	}
}

func TestGetAllTypes(t *testing.T) {
	reg := New()
	type1 := reflect.TypeOf((*testInterface)(nil)).Elem()
	type2 := reflect.TypeOf("")

	binding1 := &Binding{
		AbstractType: type1,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
	}
	binding2 := &Binding{
		AbstractType: type2,
		ConcreteType: reflect.TypeOf(""),
	}

	_ = reg.Register(binding1)
	_ = reg.Register(binding2)

	types := reg.GetAllTypes()
	if len(types) != 2 {
		t.Errorf("GetAllTypes() returned %d types, want 2", len(types))
	}
}

func TestGetAllTypes_Empty(t *testing.T) {
	reg := New()
	types := reg.GetAllTypes()
	if len(types) != 0 {
		t.Errorf("GetAllTypes() returned %d types, want 0", len(types))
	}
}

func TestGetAllNamedFor_Success(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	binding1 := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
		Name:         "impl1",
	}
	binding2 := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
		Name:         "impl2",
	}

	_ = reg.RegisterNamed(binding1)
	_ = reg.RegisterNamed(binding2)

	names := reg.GetAllNamedFor(interfaceType)
	if len(names) != 2 {
		t.Errorf("GetAllNamedFor() returned %d names, want 2", len(names))
	}
}

func TestGetAllNamedFor_NotFound(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	names := reg.GetAllNamedFor(interfaceType)
	if names != nil {
		t.Errorf("GetAllNamedFor() should return nil for non-existent type")
	}
}

func TestHasUnnamedBinding_True(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	binding := &Binding{
		AbstractType: interfaceType,
		ConcreteType: reflect.TypeOf(&testImplementation{}),
	}
	_ = reg.Register(binding)

	if !reg.HasUnnamedBinding(interfaceType) {
		t.Error("HasUnnamedBinding() should return true")
	}
}

func TestHasUnnamedBinding_False(t *testing.T) {
	reg := New()
	interfaceType := reflect.TypeOf((*testInterface)(nil)).Elem()

	if reg.HasUnnamedBinding(interfaceType) {
		t.Error("HasUnnamedBinding() should return false")
	}
}

func TestBindingAlreadyExistsError_Error(t *testing.T) {
	err := &BindingAlreadyExistsError{
		Type: reflect.TypeOf((*testInterface)(nil)).Elem(),
	}
	msg := err.Error()
	if msg == "" {
		t.Error("Error() should return non-empty string")
	}
}

func TestBindingNotFoundError_Error(t *testing.T) {
	err := &BindingNotFoundError{
		Type: reflect.TypeOf((*testInterface)(nil)).Elem(),
	}
	msg := err.Error()
	if msg == "" {
		t.Error("Error() should return non-empty string")
	}
}
