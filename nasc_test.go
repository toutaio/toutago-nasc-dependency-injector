package nasc

import (
	"sync"
	"testing"
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
