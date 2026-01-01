package nasc

import (
	"testing"
)

// Test types for advanced features

type FileLogger struct {
	filename string
}

func (f *FileLogger) Log(msg string) {
	// Write to file
}

type NotificationService interface {
	Notify(msg string)
}

type EmailNotifier struct{}

func (e *EmailNotifier) Notify(msg string) {}

type SMSNotifier struct{}

func (s *SMSNotifier) Notify(msg string) {}

type PushNotifier struct{}

func (p *PushNotifier) Notify(msg string) {}

type Application struct {
	PrimaryNotifier   NotificationService `inject:"name=primary"`
	SecondaryNotifier NotificationService `inject:"name=secondary"`
}

// Named Binding Tests

func TestBindNamed_Basic(t *testing.T) {
	container := New()

	err := container.BindNamed((*Logger)(nil), &ConsoleLogger{}, "console")
	if err != nil {
		t.Fatalf("BindNamed failed: %v", err)
	}

	err = container.BindNamed((*Logger)(nil), &FileLogger{}, "file")
	if err != nil {
		t.Fatalf("BindNamed failed: %v", err)
	}

	// Resolve by name
	consoleLogger := container.MakeNamed((*Logger)(nil), "console")
	fileLogger := container.MakeNamed((*Logger)(nil), "file")

	if consoleLogger == nil {
		t.Error("Console logger not resolved")
	}
	if fileLogger == nil {
		t.Error("File logger not resolved")
	}

	// Verify types
	if _, ok := consoleLogger.(*ConsoleLogger); !ok {
		t.Error("Console logger has wrong type")
	}
	if _, ok := fileLogger.(*FileLogger); !ok {
		t.Error("File logger has wrong type")
	}
}

func TestBindNamed_DuplicateName(t *testing.T) {
	container := New()

	_ = container.BindNamed((*Logger)(nil), &ConsoleLogger{}, "main")
	err := container.BindNamed((*Logger)(nil), &FileLogger{}, "main")

	if err == nil {
		t.Error("Expected error for duplicate name")
	}
}

func TestMakeNamed_NotFound(t *testing.T) {
	container := New()
	_ = container.BindNamed((*Logger)(nil), &ConsoleLogger{}, "console")

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for missing named binding")
		}
	}()

	container.MakeNamed((*Logger)(nil), "notfound")
}

func TestMakeNamed_EmptyName(t *testing.T) {
	container := New()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for empty name")
		}
	}()

	container.MakeNamed((*Logger)(nil), "")
}

// MakeAll Tests

func TestMakeAll_MultipleBindings(t *testing.T) {
	container := New()

	// Default binding
	_ = container.Bind((*Logger)(nil), &ConsoleLogger{})

	// Named bindings
	_ = container.BindNamed((*Logger)(nil), &FileLogger{filename: "app.log"}, "file")
	_ = container.BindNamed((*Logger)(nil), &FileLogger{filename: "error.log"}, "error")

	loggers := container.MakeAll((*Logger)(nil))

	if len(loggers) != 3 {
		t.Errorf("Expected 3 loggers, got %d", len(loggers))
	}
}

func TestMakeAll_NoBindings(t *testing.T) {
	container := New()

	loggers := container.MakeAll((*Logger)(nil))

	if len(loggers) != 0 {
		t.Errorf("Expected 0 loggers, got %d", len(loggers))
	}
}

// Tagged Binding Tests

func TestBindWithTags_Basic(t *testing.T) {
	container := New()

	err := container.BindWithTags((*Logger)(nil), &ConsoleLogger{}, []string{"console", "stdout"})
	if err != nil {
		t.Fatalf("BindWithTags failed: %v", err)
	}

	err = container.BindWithTags((*Logger)(nil), &FileLogger{}, []string{"file", "persistent"})
	if err != nil {
		t.Fatalf("BindWithTags failed: %v", err)
	}

	// Resolve by tag
	consoleLoggers := container.MakeWithTag("console")
	fileLoggers := container.MakeWithTag("file")

	if len(consoleLoggers) != 1 {
		t.Errorf("Expected 1 console logger, got %d", len(consoleLoggers))
	}
	if len(fileLoggers) != 1 {
		t.Errorf("Expected 1 file logger, got %d", len(fileLoggers))
	}
}

func TestMakeWithTag_MultipleMatches(t *testing.T) {
	container := New()

	_ = container.BindWithTags((*Logger)(nil), &ConsoleLogger{}, []string{"logger", "enabled"})
	_ = container.BindWithTags((*Logger)(nil), &FileLogger{}, []string{"logger", "enabled"})

	loggers := container.MakeWithTag("logger")

	if len(loggers) != 2 {
		t.Errorf("Expected 2 loggers with 'logger' tag, got %d", len(loggers))
	}

	enabledLoggers := container.MakeWithTag("enabled")
	if len(enabledLoggers) != 2 {
		t.Errorf("Expected 2 loggers with 'enabled' tag, got %d", len(enabledLoggers))
	}
}

func TestMakeWithTag_NoMatches(t *testing.T) {
	container := New()

	_ = container.BindWithTags((*Logger)(nil), &ConsoleLogger{}, []string{"console"})

	loggers := container.MakeWithTag("file")

	if len(loggers) != 0 {
		t.Errorf("Expected 0 loggers, got %d", len(loggers))
	}
}

func TestMakeWithTag_EmptyTag(t *testing.T) {
	container := New()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for empty tag")
		}
	}()

	container.MakeWithTag("")
}

// Auto-wire Named Dependencies Test

func TestAutoWire_NamedDependency(t *testing.T) {
	type ServiceWithNamedDeps struct {
		ConsoleLogger Logger `inject:"name=console"`
		FileLogger    Logger `inject:"name=file"`
	}

	container := New()
	_ = container.BindNamed((*Logger)(nil), &ConsoleLogger{}, "console")
	_ = container.BindNamed((*Logger)(nil), &FileLogger{}, "file")

	service := &ServiceWithNamedDeps{}
	err := container.AutoWire(service)
	if err != nil {
		t.Fatalf("AutoWire failed: %v", err)
	}

	if service.ConsoleLogger == nil {
		t.Error("ConsoleLogger not injected")
	}
	if service.FileLogger == nil {
		t.Error("FileLogger not injected")
	}

	// Verify types
	if _, ok := service.ConsoleLogger.(*ConsoleLogger); !ok {
		t.Error("ConsoleLogger has wrong type")
	}
	if _, ok := service.FileLogger.(*FileLogger); !ok {
		t.Error("FileLogger has wrong type")
	}
}

// Integration Tests

func TestAdvanced_RealWorldScenario(t *testing.T) {
	container := New()

	// Register named services
	_ = container.BindNamed((*NotificationService)(nil), &EmailNotifier{}, "primary")
	_ = container.BindNamed((*NotificationService)(nil), &SMSNotifier{}, "secondary")

	// Register tagged service
	_ = container.BindWithTags((*NotificationService)(nil), &PushNotifier{}, []string{"notifier", "mobile"})

	// Auto-wire application
	app := &Application{}
	err := container.AutoWire(app)
	if err != nil {
		t.Fatalf("AutoWire failed: %v", err)
	}

	if app.PrimaryNotifier == nil {
		t.Error("Primary notifier not injected")
	}
	if app.SecondaryNotifier == nil {
		t.Error("Secondary notifier not injected")
	}

	// Get all notifiers by tag
	mobileNotifiers := container.MakeWithTag("mobile")
	if len(mobileNotifiers) != 1 {
		t.Errorf("Expected 1 mobile notifier, got %d", len(mobileNotifiers))
	}

	// Get all notifiers
	allNotifiers := container.MakeAll((*NotificationService)(nil))
	if len(allNotifiers) != 3 {
		t.Errorf("Expected 3 total notifiers, got %d", len(allNotifiers))
	}
}

func TestAdvanced_NamedSingleton(t *testing.T) {
	container := New()

	// Singleton constructor
	callCount := 0
	NewCounter := func() *ConsoleLogger {
		callCount++
		return &ConsoleLogger{}
	}

	_ = container.SingletonConstructor((*Logger)(nil), NewCounter)

	// Make multiple times
	logger1 := container.Make((*Logger)(nil))
	logger2 := container.Make((*Logger)(nil))

	if logger1 != logger2 {
		t.Error("Singleton should return same instance")
	}

	if callCount != 1 {
		t.Errorf("Constructor called %d times, expected 1", callCount)
	}
}
