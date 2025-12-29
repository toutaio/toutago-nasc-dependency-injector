package nasc

import (
	"reflect"
	"sync"
)

// reflectionCache caches reflection metadata to avoid repeated type analysis.
// This significantly improves performance by reducing reflection overhead.
type reflectionCache struct {
	mu sync.RWMutex

	// Struct field cache for auto-wiring
	fields map[reflect.Type][]fieldInfo
}

// fieldInfo stores metadata about a struct field for auto-wiring.
type fieldInfo struct {
	index        int
	name         string
	typ          reflect.Type
	tag          reflect.StructTag
	isInjectable bool
}

// newReflectionCache creates a new reflection cache.
func newReflectionCache() *reflectionCache {
	return &reflectionCache{
		fields: make(map[reflect.Type][]fieldInfo),
	}
}

// getFieldInfo retrieves or computes struct field information.
func (rc *reflectionCache) getFieldInfo(typ reflect.Type) []fieldInfo {
	// Fast path: check cache with read lock
	rc.mu.RLock()
	fields, exists := rc.fields[typ]
	rc.mu.RUnlock()

	if exists {
		return fields
	}

	// Slow path: compute and cache with write lock
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Double-check after acquiring write lock
	fields, exists = rc.fields[typ]
	if exists {
		return fields
	}

	// Compute field information
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		rc.fields[typ] = nil
		return nil
	}

	numFields := typ.NumField()
	fields = make([]fieldInfo, 0, numFields)

	for i := 0; i < numFields; i++ {
		field := typ.Field(i)

		// Check if field is injectable (exported and has inject tag)
		_, hasInjectTag := field.Tag.Lookup("inject")
		isInjectable := field.PkgPath == "" && hasInjectTag

		fields = append(fields, fieldInfo{
			index:        i,
			name:         field.Name,
			typ:          field.Type,
			tag:          field.Tag,
			isInjectable: isInjectable,
		})
	}

	rc.fields[typ] = fields
	return fields
}

// clear clears all cached data.
func (rc *reflectionCache) clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.fields = make(map[reflect.Type][]fieldInfo)
}
