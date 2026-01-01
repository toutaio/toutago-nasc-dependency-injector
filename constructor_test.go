package nasc

import (
	"errors"
	"testing"
)

// Test service types
type ConstructorService interface {
	GetValue() string
}

type BasicConstructorService struct {
	value string
}

func (s *BasicConstructorService) GetValue() string {
	return s.value
}

type ConstructorServiceImpl struct {
	Logger   Logger
	Database Database
}

func (s *ConstructorServiceImpl) GetValue() string {
	return "service with deps"
}

// Test constructors
func NewBasicService() *BasicConstructorService {
	return &BasicConstructorService{value: "basic"}
}

func NewServiceWithLogger(logger Logger) *ConstructorServiceImpl {
	return &ConstructorServiceImpl{Logger: logger}
}

func NewServiceWithDeps(logger Logger, db Database) *ConstructorServiceImpl {
	return &ConstructorServiceImpl{Logger: logger, Database: db}
}

func NewServiceWithError(logger Logger) (*ConstructorServiceImpl, error) {
	return &ConstructorServiceImpl{Logger: logger}, nil
}

func NewServiceThatFails(logger Logger) (*ConstructorServiceImpl, error) {
	return nil, errors.New("constructor failed")
}

// Tests

func TestBindConstructor_NoParams(t *testing.T) {
	container := New()

	err := container.BindConstructor((*ConstructorService)(nil), NewBasicService)
	if err != nil {
		t.Fatalf("BindConstructor failed: %v", err)
	}

	service := container.Make((*ConstructorService)(nil))
	if service == nil {
		t.Error("Make() returned nil")
	}

	cs := service.(ConstructorService)
	if cs.GetValue() != "basic" {
		t.Error("Service not properly constructed")
	}
}

func TestBindConstructor_WithDependency(t *testing.T) {
	container := New()

	// Bind dependency
	_ = container.Bind((*Logger)(nil), &ConsoleLogger{})

	// Bind with constructor
	_ = container.BindConstructor((*ConstructorService)(nil), NewServiceWithLogger)

	service := container.Make((*ConstructorService)(nil))
	if service == nil {
		t.Error("Service was not created")
	}

	// Verify dependency was injected
	impl := service.(*ConstructorServiceImpl)
	if impl.Logger == nil {
		t.Error("Logger dependency not injected")
	}
}

func TestBindConstructor_MultipleDependencies(t *testing.T) {
	container := New()

	// Bind dependencies
	_ = container.Bind((*Logger)(nil), &ConsoleLogger{})
	_ = container.Bind((*Database)(nil), &MockDB{})

	// Bind with constructor
	_ = container.BindConstructor((*ConstructorService)(nil), NewServiceWithDeps)

	service := container.Make((*ConstructorService)(nil))
	if service == nil {
		t.Error("Service was not created")
	}

	impl := service.(*ConstructorServiceImpl)
	if impl.Logger == nil {
		t.Error("Logger dependency not injected")
	}
	if impl.Database == nil {
		t.Error("Database dependency not injected")
	}
}

func TestBindConstructor_WithError_Success(t *testing.T) {
	container := New()
	_ = container.Bind((*Logger)(nil), &ConsoleLogger{})

	_ = container.BindConstructor((*ConstructorService)(nil), NewServiceWithError)

	service := container.Make((*ConstructorService)(nil))
	if service == nil {
		t.Error("Service was not created")
	}
}

func TestBindConstructor_WithError_Failure(t *testing.T) {
	container := New()
	_ = container.Bind((*Logger)(nil), &ConsoleLogger{})

	_ = container.BindConstructor((*ConstructorService)(nil), NewServiceThatFails)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when constructor returns error")
		}
	}()

	container.Make((*ConstructorService)(nil))
}

func TestSingletonConstructor(t *testing.T) {
	container := New()
	_ = container.Bind((*Logger)(nil), &ConsoleLogger{})

	callCount := 0
	NewServiceCounted := func(logger Logger) *ConstructorServiceImpl {
		callCount++
		return &ConstructorServiceImpl{Logger: logger}
	}

	_ = container.SingletonConstructor((*ConstructorService)(nil), NewServiceCounted)

	service1 := container.Make((*ConstructorService)(nil))
	service2 := container.Make((*ConstructorService)(nil))

	if service1 != service2 {
		t.Error("Singleton constructor should return same instance")
	}

	if callCount != 1 {
		t.Errorf("Constructor called %d times, expected 1", callCount)
	}
}

func TestScopedConstructor(t *testing.T) {
	container := New()
	_ = container.Bind((*Logger)(nil), &ConsoleLogger{})

	callCount := 0
	NewServiceCounted := func(logger Logger) *ConstructorServiceImpl {
		callCount++
		return &ConstructorServiceImpl{Logger: logger}
	}

	_ = container.ScopedConstructor((*ConstructorService)(nil), NewServiceCounted)

	scope1 := container.CreateScope()
	scope2 := container.CreateScope()

	service1a := scope1.Make((*ConstructorService)(nil))
	service1b := scope1.Make((*ConstructorService)(nil))
	service2 := scope2.Make((*ConstructorService)(nil))

	// Same scope should return same instance
	if service1a != service1b {
		t.Error("Scoped constructor should return same instance within scope")
	}

	// Different scopes should have different instances
	if service1a == service2 {
		t.Error("Scoped constructor should return different instances across scopes")
	}
}

func TestParseConstructor_ValidCases(t *testing.T) {
	validConstructors := []interface{}{
		func() *BasicConstructorService { return nil },
		func() (*BasicConstructorService, error) { return nil, nil },
		func(Logger) *BasicConstructorService { return nil },
		func(Logger, Database) (*BasicConstructorService, error) { return nil, nil },
	}

	for i, constructor := range validConstructors {
		_, err := parseConstructor(constructor)
		if err != nil {
			t.Errorf("Case %d: Expected no error but got: %v", i, err)
		}
	}
}

func TestParseConstructor_InvalidCases(t *testing.T) {
	invalidConstructors := []interface{}{
		nil,
		"not a function",
		func() {}, // no return
		func() (int, int, int) { return 0, 0, 0 },                // too many returns
		func() int { return 0 },                                  // returns non-pointer
		func() (*BasicConstructorService, int) { return nil, 0 }, // second return not error
	}

	for i, constructor := range invalidConstructors {
		_, err := parseConstructor(constructor)
		if err == nil {
			t.Errorf("Case %d: Expected error but got none", i)
		}
	}
}

func TestConstructor_MissingDependency(t *testing.T) {
	container := New()
	// Logger NOT bound

	_ = container.BindConstructor((*ConstructorService)(nil), NewServiceWithLogger)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when dependency is missing")
		}
	}()

	container.Make((*ConstructorService)(nil))
}

// Benchmark
func BenchmarkConstructorInvocation(b *testing.B) {
	container := New()
	_ = container.Bind((*Logger)(nil), &ConsoleLogger{})
	_ = container.Bind((*Database)(nil), &MockDB{})

	_ = container.BindConstructor((*ConstructorService)(nil), NewServiceWithDeps)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		container.Make((*ConstructorService)(nil))
	}
}

// Additional constructor error handling tests

func TestConstructorErrors(t *testing.T) {
	c := New()

	type Service struct{}

	errorConstructor := func() (*Service, error) {
		return nil, errors.New("construction failed")
	}

	_ = c.BindConstructor((*Service)(nil), errorConstructor)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from constructor error")
		}
	}()

	c.Make((*Service)(nil))
}

func TestSingletonConstructorError(t *testing.T) {
	c := New()

	type Service struct{}

	errorConstructor := func() (*Service, error) {
		return nil, errors.New("singleton construction failed")
	}

	_ = c.SingletonConstructor((*Service)(nil), errorConstructor)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from singleton constructor error")
		}
	}()

	c.Make((*Service)(nil))
}
