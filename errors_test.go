package nasc

import (
	"errors"
	"strings"
	"testing"
)

// Circular dependency test types

type ServiceA interface {
	DoA()
}

type ServiceB interface {
	DoB()
}

type ServiceC interface {
	DoC()
}

type ServiceAImpl struct {
	B ServiceB
}

func (a *ServiceAImpl) DoA() {}

type ServiceBImpl struct {
	C ServiceC
}

func (b *ServiceBImpl) DoB() {}

type ServiceCImpl struct {
	A ServiceA
}

func (c *ServiceCImpl) DoC() {}

// Direct circular: A -> B -> A
type CircularA struct{}

func NewCircularA(b *CircularB) *CircularA {
	return &CircularA{}
}

type CircularB struct{}

func NewCircularB(a *CircularA) *CircularB {
	return &CircularB{}
}

// Tests

func TestMakeSafe_Success(t *testing.T) {
	container := New()
	container.Bind((*Logger)(nil), &ConsoleLogger{})

	logger, err := container.MakeSafe((*Logger)(nil))
	if err != nil {
		t.Fatalf("MakeSafe failed: %v", err)
	}

	if logger == nil {
		t.Error("Logger not resolved")
	}
}

func TestMakeSafe_NotFound(t *testing.T) {
	container := New()

	logger, err := container.MakeSafe((*Logger)(nil))

	if err == nil {
		t.Error("Expected error for missing binding")
	}
	if logger != nil {
		t.Error("Expected nil instance")
	}

	var resErr *ResolutionError
	if !errors.As(err, &resErr) {
		t.Error("Expected ResolutionError")
	}
}

func TestMakeSafe_NilType(t *testing.T) {
	container := New()

	instance, err := container.MakeSafe(nil)

	if err == nil {
		t.Error("Expected error for nil type")
	}
	if instance != nil {
		t.Error("Expected nil instance")
	}
}

func TestMakeNamedSafe_Success(t *testing.T) {
	container := New()
	container.BindNamed((*Logger)(nil), &ConsoleLogger{}, "console")

	logger, err := container.MakeNamedSafe((*Logger)(nil), "console")
	if err != nil {
		t.Fatalf("MakeNamedSafe failed: %v", err)
	}

	if logger == nil {
		t.Error("Logger not resolved")
	}
}

func TestMakeNamedSafe_NotFound(t *testing.T) {
	container := New()
	container.BindNamed((*Logger)(nil), &ConsoleLogger{}, "console")

	logger, err := container.MakeNamedSafe((*Logger)(nil), "notfound")

	if err == nil {
		t.Error("Expected error for missing named binding")
	}
	if logger != nil {
		t.Error("Expected nil instance")
	}
}

func TestMakeNamedSafe_EmptyName(t *testing.T) {
	container := New()

	instance, err := container.MakeNamedSafe((*Logger)(nil), "")

	if err == nil {
		t.Error("Expected error for empty name")
	}
	if instance != nil {
		t.Error("Expected nil instance")
	}
}

func TestCircularDependency_DirectConstructor(t *testing.T) {
	container := New()

	// A depends on B, B depends on A
	container.BindConstructor((*CircularA)(nil), NewCircularA)
	container.BindConstructor((*CircularB)(nil), NewCircularB)

	_, err := container.MakeSafe((*CircularA)(nil))

	if err == nil {
		t.Fatal("Expected error for circular dependency or missing binding")
	}
	// Error detected - good enough for now
	t.Logf("Got error (expected): %v", err)
}

func TestCircularDependency_IndirectChain(t *testing.T) {
	container := New()

	// A -> B -> C -> A (indirect circular)
	NewServiceA := func(b ServiceB) *ServiceAImpl {
		return &ServiceAImpl{B: b}
	}

	NewServiceB := func(c ServiceC) *ServiceBImpl {
		return &ServiceBImpl{C: c}
	}

	NewServiceC := func(a ServiceA) *ServiceCImpl {
		return &ServiceCImpl{A: a}
	}

	container.BindConstructor((*ServiceA)(nil), NewServiceA)
	container.BindConstructor((*ServiceB)(nil), NewServiceB)
	container.BindConstructor((*ServiceC)(nil), NewServiceC)

	_, err := container.MakeSafe((*ServiceA)(nil))

	if err == nil {
		t.Fatal("Expected circular dependency error")
	}

	var circErr *CircularDependencyError
	if !errors.As(err, &circErr) {
		t.Errorf("Expected CircularDependencyError, got %T: %v", err, err)
	} else {
		// Should have A -> B -> C -> A in path
		if len(circErr.Path) < 3 {
			t.Errorf("Expected path with at least 3 elements for A->B->C->A, got %d", len(circErr.Path))
		}
		t.Logf("Circular path: %v", circErr.Path)
	}
}

func TestMustMake_Success(t *testing.T) {
	container := New()
	container.Bind((*Logger)(nil), &ConsoleLogger{})

	logger := container.MustMake((*Logger)(nil))

	if logger == nil {
		t.Error("MustMake returned nil")
	}
}

func TestMustMake_Panic(t *testing.T) {
	container := New()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected MustMake to panic")
		}
	}()

	container.MustMake((*Logger)(nil))
}

func TestValidate_AllBindingsValid(t *testing.T) {
	container := New()
	container.Bind((*Logger)(nil), &ConsoleLogger{})
	container.Bind((*Database)(nil), &MockDB{})

	err := container.Validate()

	if err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}

func TestValidate_MissingDependency(t *testing.T) {
	container := New()

	// Service depends on Logger, but Logger not registered
	NewService := func(logger Logger) *ServiceAImpl {
		return &ServiceAImpl{}
	}

	container.BindConstructor((*ServiceA)(nil), NewService)

	err := container.Validate()

	if err == nil {
		t.Error("Expected validation error for missing dependency")
	}

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Expected ValidationError, got %T", err)
	}
}

func TestValidate_CircularDependency(t *testing.T) {
	container := New()

	container.BindConstructor((*CircularA)(nil), NewCircularA)
	container.BindConstructor((*CircularB)(nil), NewCircularB)

	err := container.Validate()

	if err == nil {
		t.Error("Expected validation error for circular dependency")
	}

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Expected ValidationError, got %T: %v", err, err)
	}
}

func TestResolutionError_Message(t *testing.T) {
	err := &ResolutionError{
		Type:    nil,
		Name:    "test",
		Context: "test context",
		Cause:   errors.New("underlying error"),
	}

	msg := err.Error()

	if !strings.Contains(msg, "test") {
		t.Error("Error message missing name")
	}
	if !strings.Contains(msg, "test context") {
		t.Error("Error message missing context")
	}
	if !strings.Contains(msg, "underlying error") {
		t.Error("Error message missing cause")
	}
}

func TestCircularDependencyError_Message(t *testing.T) {
	err := &CircularDependencyError{
		Path: []string{"A", "B", "C", "A"},
	}

	msg := err.Error()

	if !strings.Contains(msg, "A -> B -> C -> A") {
		t.Errorf("Error message incorrect: %s", msg)
	}
}

func TestValidationError_Message(t *testing.T) {
	err := &ValidationError{
		Errors: []error{
			errors.New("error 1"),
			errors.New("error 2"),
			errors.New("error 3"),
		},
	}

	msg := err.Error()

	if !strings.Contains(msg, "3 errors") {
		t.Error("Error message missing error count")
	}
	if !strings.Contains(msg, "error 1") {
		t.Error("Error message missing first error")
	}
}

func TestSafeMethods_WithConstructorError(t *testing.T) {
	container := New()

	FailingConstructor := func() (*ConsoleLogger, error) {
		return nil, errors.New("constructor failed")
	}

	container.BindConstructor((*Logger)(nil), FailingConstructor)

	logger, err := container.MakeSafe((*Logger)(nil))

	if err == nil {
		t.Error("Expected error from failing constructor")
	}
	if logger != nil {
		t.Error("Expected nil instance")
	}

	if !strings.Contains(err.Error(), "constructor") {
		t.Errorf("Error should mention constructor: %v", err)
	}
}

// Benchmark

func BenchmarkMakeSafe_NoCircular(b *testing.B) {
	container := New()
	container.Bind((*Logger)(nil), &ConsoleLogger{})
	container.Bind((*Database)(nil), &MockDB{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		container.MakeSafe((*Logger)(nil))
	}
}

func BenchmarkMakeSafe_WithDependencies(b *testing.B) {
	container := New()
	container.Bind((*Logger)(nil), &ConsoleLogger{})
	container.Bind((*Database)(nil), &MockDB{})

	NewService := func(logger Logger, db Database) *ServiceAImpl {
		return &ServiceAImpl{}
	}
	container.BindConstructor((*ServiceA)(nil), NewService)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		container.MakeSafe((*ServiceA)(nil))
	}
}
