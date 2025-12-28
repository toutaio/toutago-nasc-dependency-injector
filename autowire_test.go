package nasc

import (
	"testing"
)

// Test service with dependencies
type ServiceWithDeps struct {
Logger   Logger   `inject:""`
Database Database `inject:""`
}

type ServiceWithOptional struct {
Logger Logger `inject:""`
Database Database `inject:"optional"` // Optional, may not be bound
}

type ServicePartialTags struct {
Logger   Logger   `inject:""`
Database Database // No tag, won't be injected
}

type ServiceNoTags struct {
Logger   Logger
Database Database
}

// Tests

func TestAutoWire_BasicInjection(t *testing.T) {
container := New()
container.Bind((*Logger)(nil), &ConsoleLogger{})
container.Bind((*Database)(nil), &MockDB{})

service := &ServiceWithDeps{}
err := container.AutoWire(service)
if err != nil {
t.Fatalf("AutoWire failed: %v", err)
}

if service.Logger == nil {
t.Error("Logger was not injected")
}
if service.Database == nil {
t.Error("Database was not injected")
}
}

func TestAutoWire_OptionalDependency(t *testing.T) {
container := New()
container.Bind((*Logger)(nil), &ConsoleLogger{})
// Note: Database is NOT bound

service := &ServiceWithOptional{}
err := container.AutoWire(service)
if err != nil {
t.Fatalf("AutoWire should not fail with optional dependency: %v", err)
}

if service.Logger == nil {
t.Error("Logger (required) was not injected")
}
if service.Database != nil {
t.Error("Database (optional, unbound) should remain nil")
}
}

func TestAutoWire_PartialTags(t *testing.T) {
container := New()
container.Bind((*Logger)(nil), &ConsoleLogger{})
// Database intentionally not bound to test partial injection

service := &ServicePartialTags{}
err := container.AutoWire(service)
if err != nil {
t.Fatalf("AutoWire failed: %v", err)
}

if service.Logger == nil {
t.Error("Logger (tagged) was not injected")
}
if service.Database != nil {
t.Log("Database (not tagged) should remain nil - this is expected")
}
}

func TestAutoWire_NoTags(t *testing.T) {
container := New()
container.Bind((*Logger)(nil), &ConsoleLogger{})
container.Bind((*Database)(nil), &MockDB{})

service := &ServiceNoTags{}
err := container.AutoWire(service)
if err != nil {
t.Fatalf("AutoWire should succeed even with no tags: %v", err)
}

// Nothing should be injected
if service.Logger != nil {
t.Error("Logger should not be injected without tag")
}
if service.Database != nil {
t.Error("Database should not be injected without tag")
}
}

func TestAutoWire_NilInstance(t *testing.T) {
container := New()
err := container.AutoWire(nil)
if err == nil {
t.Error("AutoWire should fail with nil instance")
}
}

func TestAutoWire_NotAPointer(t *testing.T) {
container := New()
service := ServiceNoTags{}
err := container.AutoWire(service) // Not a pointer
if err == nil {
t.Error("AutoWire should fail with non-pointer")
}
}

func TestAutoWire_MissingDependency(t *testing.T) {
container := New()
// Logger is required but not bound

service := &ServiceWithDeps{}
err := container.AutoWire(service)
if err == nil {
t.Error("AutoWire should fail when required dependency is missing")
}
}

func TestAutoWire_SingletonInjection(t *testing.T) {
container := New()
container.Singleton((*Logger)(nil), &ConsoleLogger{})
container.Bind((*Database)(nil), &MockDB{})

service1 := &ServiceWithDeps{}
service2 := &ServiceWithDeps{}

container.AutoWire(service1)
container.AutoWire(service2)

// Both should have the same Logger instance (singleton)
if service1.Logger != service2.Logger {
t.Error("Singleton logger should be same instance in both services")
}

// But different Database instances (transient)
if service1.Database == service2.Database {
t.Error("Transient database should be different instances")
}
}

func TestParseInjectTag(t *testing.T) {
tests := []struct {
tag      string
expected tagOptions
}{
{"", tagOptions{skip: false, optional: false, name: ""}},
{"-", tagOptions{skip: true, optional: false, name: ""}},
{"optional", tagOptions{skip: false, optional: true, name: ""}},
{"name=foo", tagOptions{skip: false, optional: false, name: "foo"}},
{"optional,name=bar", tagOptions{skip: false, optional: true, name: "bar"}},
{"name=baz,optional", tagOptions{skip: false, optional: true, name: "baz"}},
}

for _, tt := range tests {
t.Run(tt.tag, func(t *testing.T) {
result := parseInjectTag(tt.tag)
if result.skip != tt.expected.skip {
t.Errorf("skip: got %v, want %v", result.skip, tt.expected.skip)
}
if result.optional != tt.expected.optional {
t.Errorf("optional: got %v, want %v", result.optional, tt.expected.optional)
}
if result.name != tt.expected.name {
t.Errorf("name: got %v, want %v", result.name, tt.expected.name)
}
})
}
}

// Example test
func ExampleNasc_AutoWire() {
type MyService struct {
Logger   Logger   `inject:""`
Database Database `inject:""`
}

container := New()
container.Bind((*Logger)(nil), &ConsoleLogger{})
container.Bind((*Database)(nil), &MockDB{})

service := &MyService{}
container.AutoWire(service)

// service.Logger and service.Database are now injected
_ = service.Logger
_ = service.Database
}

// Benchmark
func BenchmarkAutoWire(b *testing.B) {
container := New()
container.Bind((*Logger)(nil), &ConsoleLogger{})
container.Bind((*Database)(nil), &MockDB{})

b.ResetTimer()
for i := 0; i < b.N; i++ {
service := &ServiceWithDeps{}
container.AutoWire(service)
}
}

