package nasc

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/toutaio/toutago-nasc-dependency-injector/registry"
)

// Test interfaces and implementations
type Logger interface {
	Log(msg string)
}

type ConsoleLogger struct {
	messages []string
}

func (l *ConsoleLogger) Log(msg string) {
	l.messages = append(l.messages, msg)
}

type Database interface {
	Connect() error
}

type MockDB struct {
	connected bool
}

func (db *MockDB) Connect() error {
	db.connected = true
	return nil
}

func TestNew(t *testing.T) {
	container := New()
	if container == nil {
		t.Fatal("New() returned nil")
	}
	if container.registry == nil {
		t.Error("container.registry is nil")
	}
}

func TestNew_WithOptions(t *testing.T) {
	container := New(WithDebug(), WithValidation())
	if container == nil {
		t.Fatal("New() with options returned nil")
	}
}

func TestBind_Success(t *testing.T) {
	container := New()
	err := container.Bind((*Logger)(nil), &ConsoleLogger{})
	if err != nil {
		t.Errorf("Bind() returned error: %v", err)
	}
}

func TestBind_NilAbstractType(t *testing.T) {
	container := New()
	err := container.Bind(nil, &ConsoleLogger{})
	if err == nil {
		t.Error("Bind() with nil abstract type should return error")
	}
	if _, ok := err.(*InvalidBindingError); !ok {
		t.Errorf("Expected InvalidBindingError, got %T", err)
	}
}

func TestBind_NilConcreteType(t *testing.T) {
	container := New()
	err := container.Bind((*Logger)(nil), nil)
	if err == nil {
		t.Error("Bind() with nil concrete type should return error")
	}
	if _, ok := err.(*InvalidBindingError); !ok {
		t.Errorf("Expected InvalidBindingError, got %T", err)
	}
}

func TestBind_Duplicate(t *testing.T) {
	container := New()
	err := container.Bind((*Logger)(nil), &ConsoleLogger{})
	if err != nil {
		t.Fatalf("First Bind() failed: %v", err)
	}
	err = container.Bind((*Logger)(nil), &ConsoleLogger{})
	if err == nil {
		t.Error("Duplicate Bind() should return error")
	}
}

func TestMake_Success(t *testing.T) {
	container := New()
	container.Bind((*Logger)(nil), &ConsoleLogger{})
	instance := container.Make((*Logger)(nil))
	if instance == nil {
		t.Fatal("Make() returned nil")
	}
	logger, ok := instance.(Logger)
	if !ok {
		t.Fatalf("Make() returned wrong type: %T", instance)
	}
	logger.Log("test message")
	consoleLogger := instance.(*ConsoleLogger)
	if len(consoleLogger.messages) != 1 {
		t.Errorf("Logger didn't record message")
	}
}

func TestMake_MultipleInstances(t *testing.T) {
	container := New()
	container.Bind((*Logger)(nil), &ConsoleLogger{})
	instance1 := container.Make((*Logger)(nil))
	instance2 := container.Make((*Logger)(nil))
	if instance1 == instance2 {
		t.Error("Make() returned same instance (should be transient)")
	}
}

func TestMake_NotFound_Panics(t *testing.T) {
	container := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Make() should panic when binding not found")
		}
	}()
	container.Make((*Logger)(nil))
}

func TestMake_NilType_Panics(t *testing.T) {
	container := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Make() should panic with nil type")
		}
	}()
	container.Make(nil)
}

func TestIntegration_MultipleBindings(t *testing.T) {
	container := New()
	err := container.Bind((*Logger)(nil), &ConsoleLogger{})
	if err != nil {
		t.Fatalf("Failed to bind Logger: %v", err)
	}
	err = container.Bind((*Database)(nil), &MockDB{})
	if err != nil {
		t.Fatalf("Failed to bind Database: %v", err)
	}
	logger := container.Make((*Logger)(nil)).(Logger)
	db := container.Make((*Database)(nil)).(Database)
	logger.Log("connecting to database")
	err = db.Connect()
	if err != nil {
		t.Errorf("Database.Connect() failed: %v", err)
	}
	mockDB := db.(*MockDB)
	if !mockDB.connected {
		t.Error("Database not connected")
	}
}

func TestConcurrentMake(t *testing.T) {
	container := New()
	container.Bind((*Logger)(nil), &ConsoleLogger{})
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger := container.Make((*Logger)(nil))
			if logger == nil {
				t.Error("Concurrent Make() returned nil")
			}
		}()
	}
	wg.Wait()
}

func BenchmarkBind(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := New()
		c.Bind((*Logger)(nil), &ConsoleLogger{})
	}
}

func BenchmarkMake(b *testing.B) {
	container := New()
	container.Bind((*Logger)(nil), &ConsoleLogger{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		container.Make((*Logger)(nil))
	}
}

// Additional core functionality tests

func TestLifetime_String(t *testing.T) {
	tests := []struct {
		lifetime Lifetime
		want     string
	}{
		{LifetimeTransient, "transient"},
		{LifetimeSingleton, "singleton"},
		{LifetimeScoped, "scoped"},
		{LifetimeFactory, "factory"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.lifetime.String()
			if got != tt.want {
				t.Errorf("Lifetime.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFactoryErrorHandling(t *testing.T) {
	c := New()

	type Service struct{}

	factory := func(n *Nasc) (interface{}, error) {
		return nil, errors.New("factory error")
	}

	c.Factory((*Service)(nil), factory)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from factory error")
		}
	}()

	c.Make((*Service)(nil))
}

func TestInvalidFactoryType(t *testing.T) {
	c := New()

	type Service struct{}

	c.registry.Register(&registry.Binding{
		AbstractType: reflect.TypeOf((*Service)(nil)).Elem(),
		ConcreteType: reflect.TypeOf(&Service{}),
		Lifetime:     string(LifetimeFactory),
		Factory:      "not a function",
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from invalid factory")
		}
	}()

	c.Make((*Service)(nil))
}

func TestUnknownLifetime(t *testing.T) {
	c := New()

	type Service struct{}

	c.registry.Register(&registry.Binding{
		AbstractType: reflect.TypeOf((*Service)(nil)).Elem(),
		ConcreteType: reflect.TypeOf(&Service{}),
		Lifetime:     "unknown-lifetime",
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from unknown lifetime")
		}
	}()

	c.Make((*Service)(nil))
}

func TestReflectionCache_Clear(t *testing.T) {
	cache := newReflectionCache()

	type TestStruct struct {
		Field string
	}

	testType := reflect.TypeOf(TestStruct{})
	cache.getFieldInfo(testType)

	cache.clear()

	fields := cache.getFieldInfo(testType)
	if len(fields) != 1 {
		t.Errorf("Cache should still work after clear, got %d fields", len(fields))
	}
}

// Singleton lifetime tests

func TestSingleton_ReturnsSameInstance(t *testing.T) {
	container := New()
	container.Singleton((*Logger)(nil), &ConsoleLogger{})

	instance1 := container.Make((*Logger)(nil))
	instance2 := container.Make((*Logger)(nil))

	if fmt.Sprintf("%p", instance1) != fmt.Sprintf("%p", instance2) {
		t.Error("Singleton returned different instances")
	}
}

func TestSingleton_ThreadSafe(t *testing.T) {
	container := New()
	container.Singleton((*Logger)(nil), &ConsoleLogger{})

	var wg sync.WaitGroup
	instances := make([]interface{}, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			instances[i] = container.Make((*Logger)(nil))
		}()
	}

	wg.Wait()

	first := fmt.Sprintf("%p", instances[0])
	for i := 1; i < 50; i++ {
		if fmt.Sprintf("%p", instances[i]) != first {
			t.Errorf("Instance %d is different", i)
			break
		}
	}
}

// Factory lifetime tests

func TestFactory_CalledEachTime(t *testing.T) {
	container := New()
	callCount := 0

	factory := func(c *Nasc) (interface{}, error) {
		callCount++
		return &ConsoleLogger{}, nil
	}

	container.Factory((*Logger)(nil), factory)

	container.Make((*Logger)(nil))
	container.Make((*Logger)(nil))
	container.Make((*Logger)(nil))

	if callCount != 3 {
		t.Errorf("Factory called %d times, expected 3", callCount)
	}
}

func TestFactory_ReturnsNewInstances(t *testing.T) {
	container := New()

	factory := func(c *Nasc) (interface{}, error) {
		return &ConsoleLogger{}, nil
	}

	container.Factory((*Logger)(nil), factory)

	instance1 := container.Make((*Logger)(nil))
	instance2 := container.Make((*Logger)(nil))

	if fmt.Sprintf("%p", instance1) == fmt.Sprintf("%p", instance2) {
		t.Error("Factory should return different instances")
	}
}

func TestFactory_ReceivesContainer(t *testing.T) {
	container := New()
	container.Singleton((*Database)(nil), &MockDB{})

	receivedContainer := false
	factory := func(c *Nasc) (interface{}, error) {
		if c != nil {
			receivedContainer = true
			c.Make((*Database)(nil))
		}
		return &ConsoleLogger{}, nil
	}

	container.Factory((*Logger)(nil), factory)
	container.Make((*Logger)(nil))

	if !receivedContainer {
		t.Error("Factory did not receive container")
	}
}
