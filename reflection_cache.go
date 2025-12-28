package nasc

import (
	"reflect"
	"sync"
)

// reflectionCache caches reflection metadata to avoid repeated type analysis.
// This significantly improves performance by reducing reflection overhead.
type reflectionCache struct {
	mu sync.RWMutex

	// Type information cache
	types map[reflect.Type]*typeInfo

	// Constructor signature cache
	constructors map[reflect.Value]*constructorMetadata

	// Struct field cache for auto-wiring
	fields map[reflect.Type][]fieldInfo
}

// typeInfo stores cached metadata about a type.
type typeInfo struct {
	typ        reflect.Type
	isPtr      bool
	elem       reflect.Type
	numField   int
	implements map[reflect.Type]bool
}

// constructorMetadata stores parsed constructor information.
type constructorMetadata struct {
	paramTypes  []reflect.Type
	returnType  reflect.Type
	hasError    bool
	numIn       int
	numOut      int
}

// fieldInfo stores metadata about a struct field for auto-wiring.
type fieldInfo struct {
	index       int
	name        string
	typ         reflect.Type
	tag         reflect.StructTag
	isInjectable bool
}

// newReflectionCache creates a new reflection cache.
func newReflectionCache() *reflectionCache {
	return &reflectionCache{
		types:        make(map[reflect.Type]*typeInfo),
		constructors: make(map[reflect.Value]*constructorMetadata),
		fields:       make(map[reflect.Type][]fieldInfo),
	}
}

// getTypeInfo retrieves or computes type information.
func (rc *reflectionCache) getTypeInfo(typ reflect.Type) *typeInfo {
	// Fast path: check cache with read lock
	rc.mu.RLock()
	info, exists := rc.types[typ]
	rc.mu.RUnlock()

	if exists {
		return info
	}

	// Slow path: compute and cache with write lock
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Double-check after acquiring write lock
	info, exists = rc.types[typ]
	if exists {
		return info
	}

	// Compute type information
	info = &typeInfo{
		typ:        typ,
		isPtr:      typ.Kind() == reflect.Ptr,
		implements: make(map[reflect.Type]bool),
	}

	if info.isPtr {
		info.elem = typ.Elem()
		if info.elem.Kind() == reflect.Struct {
			info.numField = info.elem.NumField()
		}
	} else if typ.Kind() == reflect.Struct {
		info.numField = typ.NumField()
	}

	rc.types[typ] = info
	return info
}

// getConstructorMetadata retrieves or computes constructor metadata.
func (rc *reflectionCache) getConstructorMetadata(fn reflect.Value) *constructorMetadata {
	// Fast path: check cache with read lock
	rc.mu.RLock()
	meta, exists := rc.constructors[fn]
	rc.mu.RUnlock()

	if exists {
		return meta
	}

	// Slow path: compute and cache with write lock
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Double-check after acquiring write lock
	meta, exists = rc.constructors[fn]
	if exists {
		return meta
	}

	// Compute constructor metadata
	fnType := fn.Type()
	meta = &constructorMetadata{
		numIn:  fnType.NumIn(),
		numOut: fnType.NumOut(),
	}

	// Cache parameter types
	meta.paramTypes = make([]reflect.Type, meta.numIn)
	for i := 0; i < meta.numIn; i++ {
		meta.paramTypes[i] = fnType.In(i)
	}

	// Cache return type and error handling
	if meta.numOut > 0 {
		meta.returnType = fnType.Out(0)
	}
	if meta.numOut == 2 {
		meta.hasError = fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem())
	}

	rc.constructors[fn] = meta
	return meta
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

	rc.types = make(map[reflect.Type]*typeInfo)
	rc.constructors = make(map[reflect.Value]*constructorMetadata)
	rc.fields = make(map[reflect.Type][]fieldInfo)
}
