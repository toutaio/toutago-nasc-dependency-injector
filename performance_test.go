package nasc

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

// Benchmark types
type BenchLogger interface {
	Log(string)
}

type BenchConsoleLogger struct {
	prefix string
}

func (l *BenchConsoleLogger) Log(msg string) {
	_ = fmt.Sprintf("%s: %s", l.prefix, msg)
}

type BenchDatabase interface {
	Query(string) string
}

type BenchPostgresDB struct {
	Logger BenchLogger `inject:""`
}

func (db *BenchPostgresDB) Query(q string) string {
	return "result"
}

type BenchService interface {
	Process(string) string
}

type BenchUserService struct {
	DB     BenchDatabase `inject:""`
	Logger BenchLogger   `inject:""`
}

func (s *BenchUserService) Process(data string) string {
	return s.DB.Query(data)
}

// BenchmarkSingletonResolution benchmarks singleton instance retrieval.
func BenchmarkSingletonResolution(b *testing.B) {
	container := New()
	container.Singleton((*BenchLogger)(nil), &BenchConsoleLogger{})

	// Warm up the cache
	_ = container.Make((*BenchLogger)(nil))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = container.Make((*BenchLogger)(nil))
	}
}

// BenchmarkTransientResolution benchmarks transient instance creation.
func BenchmarkTransientResolution(b *testing.B) {
	container := New()
	container.Bind((*BenchLogger)(nil), &BenchConsoleLogger{})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = container.Make((*BenchLogger)(nil))
	}
}

// BenchmarkConstructorResolution benchmarks constructor-based resolution.
func BenchmarkConstructorResolution(b *testing.B) {
	container := New()
	container.Singleton((*BenchLogger)(nil), &BenchConsoleLogger{})

	err := container.BindConstructor((*BenchDatabase)(nil), func(logger BenchLogger) *BenchPostgresDB {
		return &BenchPostgresDB{Logger: logger}
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = container.Make((*BenchDatabase)(nil))
	}
}

// BenchmarkAutoWireResolution benchmarks auto-wiring performance.
func BenchmarkAutoWireResolution(b *testing.B) {
	container := New()
	container.Singleton((*BenchLogger)(nil), &BenchConsoleLogger{})
	container.Singleton((*BenchDatabase)(nil), &BenchPostgresDB{})
	container.BindAutoWire((*BenchService)(nil), &BenchUserService{})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = container.Make((*BenchService)(nil))
	}
}

// BenchmarkConcurrentResolution benchmarks concurrent singleton access.
func BenchmarkConcurrentResolution(b *testing.B) {
	container := New()
	container.Singleton((*BenchLogger)(nil), &BenchConsoleLogger{})

	// Warm up cache
	_ = container.Make((*BenchLogger)(nil))

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = container.Make((*BenchLogger)(nil))
		}
	})
}

// BenchmarkReflectionCache benchmarks reflection cache performance.
func BenchmarkReflectionCache(b *testing.B) {
	cache := newReflectionCache()
	typ := (*BenchUserService)(nil)
	structType := reflect.TypeOf(typ).Elem()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cache.getFieldInfo(structType)
	}
}

// BenchmarkFactoryResolution benchmarks factory function performance.
func BenchmarkFactoryResolution(b *testing.B) {
	container := New()

	err := container.Factory((*BenchLogger)(nil), func(c *Nasc) (interface{}, error) {
		return &BenchConsoleLogger{prefix: "factory"}, nil
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = container.Make((*BenchLogger)(nil))
	}
}

// BenchmarkNamedResolution benchmarks named binding resolution.
func BenchmarkNamedResolution(b *testing.B) {
	container := New()

	err := container.BindNamed((*BenchLogger)(nil), &BenchConsoleLogger{prefix: "file"}, "file")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = container.MakeNamed((*BenchLogger)(nil), "file")
	}
}

// BenchmarkScopedResolution benchmarks scoped lifetime performance.
func BenchmarkScopedResolution(b *testing.B) {
	container := New()
	container.Scoped((*BenchLogger)(nil), &BenchConsoleLogger{})
	scope := container.CreateScope()
	defer scope.Dispose()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = scope.Make((*BenchLogger)(nil))
	}
}

// BenchmarkDeepDependencyGraph benchmarks complex dependency resolution.
func BenchmarkDeepDependencyGraph(b *testing.B) {
	container := New()

	// Build a dependency graph
	container.Singleton((*BenchLogger)(nil), &BenchConsoleLogger{})
	container.Singleton((*BenchDatabase)(nil), &BenchPostgresDB{})
	container.BindAutoWire((*BenchService)(nil), &BenchUserService{})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = container.Make((*BenchService)(nil))
	}
}

// BenchmarkMakeAllResolution benchmarks MakeAll performance.
func BenchmarkMakeAllResolution(b *testing.B) {
	container := New()

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("logger%d", i)
		container.BindNamed((*BenchLogger)(nil), &BenchConsoleLogger{prefix: name}, name)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = container.MakeAll((*BenchLogger)(nil))
	}
}

// BenchmarkConcurrentSingletonCreation benchmarks first-time singleton creation under load.
func BenchmarkConcurrentSingletonCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		container := New()
		container.Singleton((*BenchLogger)(nil), &BenchConsoleLogger{})

		var wg sync.WaitGroup
		goroutines := 100
		wg.Add(goroutines)

		b.StartTimer()
		for j := 0; j < goroutines; j++ {
			go func() {
				defer wg.Done()
				_ = container.Make((*BenchLogger)(nil))
			}()
		}
		wg.Wait()
	}
}

// BenchmarkValidation benchmarks container validation performance.
func BenchmarkValidation(b *testing.B) {
	container := New()
	container.Singleton((*BenchLogger)(nil), &BenchConsoleLogger{})
	container.Singleton((*BenchDatabase)(nil), &BenchPostgresDB{})
	container.BindAutoWire((*BenchService)(nil), &BenchUserService{})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = container.Validate()
	}
}
