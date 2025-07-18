// Package plugin provides context sharing capabilities for orchestrator plugins
package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrKeyNotFound indicates a requested key does not exist in the context
	ErrKeyNotFound = errors.New("key not found in plugin context")
	
	// ErrContextNotFound indicates the plugin context is not available
	ErrContextNotFound = errors.New("plugin context not found")
	
	// ErrInvalidType indicates a type assertion failed
	ErrInvalidType = errors.New("invalid type for context value")
)

// PluginContext provides shared state between plugin phases
type PluginContext interface {
	// Set stores a value in the context
	Set(key string, value interface{})
	
	// Get retrieves a value from the context
	Get(key string) (interface{}, bool)
	
	// GetString retrieves a string value
	GetString(key string) (string, error)
	
	// GetInt retrieves an integer value
	GetInt(key string) (int, error)
	
	// GetBool retrieves a boolean value
	GetBool(key string) (bool, error)
	
	// GetMap retrieves a map value
	GetMap(key string) (map[string]interface{}, error)
	
	// GetSlice retrieves a slice value
	GetSlice(key string) ([]interface{}, error)
	
	// Delete removes a value from the context
	Delete(key string)
	
	// Clear removes all values from the context
	Clear()
	
	// Keys returns all keys in the context
	Keys() []string
	
	// Clone creates a deep copy of the context
	Clone() PluginContext
	
	// MarshalJSON serializes the context to JSON
	MarshalJSON() ([]byte, error)
	
	// UnmarshalJSON deserializes the context from JSON
	UnmarshalJSON(data []byte) error
}

// pluginContextImpl is the default thread-safe implementation
type pluginContextImpl struct {
	mu     sync.RWMutex
	data   map[string]interface{}
	metadata map[string]*contextMetadata
}

// contextMetadata tracks information about stored values
type contextMetadata struct {
	CreatedAt   time.Time
	UpdatedAt   time.Time
	AccessCount int64
	Type        string
}

// NewPluginContext creates a new plugin context
func NewPluginContext() PluginContext {
	return &pluginContextImpl{
		data:     make(map[string]interface{}),
		metadata: make(map[string]*contextMetadata),
	}
}

// Set stores a value in the context
func (pc *pluginContextImpl) Set(key string, value interface{}) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	now := time.Now()
	if meta, exists := pc.metadata[key]; exists {
		meta.UpdatedAt = now
		meta.Type = fmt.Sprintf("%T", value)
	} else {
		pc.metadata[key] = &contextMetadata{
			CreatedAt:   now,
			UpdatedAt:   now,
			AccessCount: 0,
			Type:        fmt.Sprintf("%T", value),
		}
	}
	
	pc.data[key] = value
}

// Get retrieves a value from the context
func (pc *pluginContextImpl) Get(key string) (interface{}, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	value, exists := pc.data[key]
	if exists {
		if meta := pc.metadata[key]; meta != nil {
			meta.AccessCount++
		}
	}
	return value, exists
}

// GetString retrieves a string value
func (pc *pluginContextImpl) GetString(key string) (string, error) {
	value, exists := pc.Get(key)
	if !exists {
		return "", fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	
	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%w: expected string, got %T", ErrInvalidType, value)
	}
	
	return str, nil
}

// GetInt retrieves an integer value
func (pc *pluginContextImpl) GetInt(key string) (int, error) {
	value, exists := pc.Get(key)
	if !exists {
		return 0, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	
	// Handle various numeric types
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case json.Number:
		i64, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("failed to convert json.Number to int: %w", err)
		}
		return int(i64), nil
	default:
		return 0, fmt.Errorf("%w: expected numeric type, got %T", ErrInvalidType, value)
	}
}

// GetBool retrieves a boolean value
func (pc *pluginContextImpl) GetBool(key string) (bool, error) {
	value, exists := pc.Get(key)
	if !exists {
		return false, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	
	b, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%w: expected bool, got %T", ErrInvalidType, value)
	}
	
	return b, nil
}

// GetMap retrieves a map value
func (pc *pluginContextImpl) GetMap(key string) (map[string]interface{}, error) {
	value, exists := pc.Get(key)
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	
	m, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: expected map[string]interface{}, got %T", ErrInvalidType, value)
	}
	
	return m, nil
}

// GetSlice retrieves a slice value
func (pc *pluginContextImpl) GetSlice(key string) ([]interface{}, error) {
	value, exists := pc.Get(key)
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	
	s, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: expected []interface{}, got %T", ErrInvalidType, value)
	}
	
	return s, nil
}

// Delete removes a value from the context
func (pc *pluginContextImpl) Delete(key string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	delete(pc.data, key)
	delete(pc.metadata, key)
}

// Clear removes all values from the context
func (pc *pluginContextImpl) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	pc.data = make(map[string]interface{})
	pc.metadata = make(map[string]*contextMetadata)
}

// Keys returns all keys in the context
func (pc *pluginContextImpl) Keys() []string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	keys := make([]string, 0, len(pc.data))
	for key := range pc.data {
		keys = append(keys, key)
	}
	return keys
}

// Clone creates a deep copy of the context
func (pc *pluginContextImpl) Clone() PluginContext {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	clone := &pluginContextImpl{
		data:     make(map[string]interface{}),
		metadata: make(map[string]*contextMetadata),
	}
	
	// Deep copy the data using JSON marshaling
	data, err := json.Marshal(pc.data)
	if err == nil {
		var clonedData map[string]interface{}
		if err := json.Unmarshal(data, &clonedData); err == nil {
			clone.data = clonedData
		}
	}
	
	// Copy metadata
	for key, meta := range pc.metadata {
		clone.metadata[key] = &contextMetadata{
			CreatedAt:   meta.CreatedAt,
			UpdatedAt:   meta.UpdatedAt,
			AccessCount: meta.AccessCount,
			Type:        meta.Type,
		}
	}
	
	return clone
}

// MarshalJSON serializes the context to JSON
func (pc *pluginContextImpl) MarshalJSON() ([]byte, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	return json.Marshal(pc.data)
}

// UnmarshalJSON deserializes the context from JSON
func (pc *pluginContextImpl) UnmarshalJSON(data []byte) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	var newData map[string]interface{}
	if err := json.Unmarshal(data, &newData); err != nil {
		return err
	}
	
	pc.data = newData
	
	// Reset metadata for new data
	pc.metadata = make(map[string]*contextMetadata)
	now := time.Now()
	for key, value := range pc.data {
		pc.metadata[key] = &contextMetadata{
			CreatedAt:   now,
			UpdatedAt:   now,
			AccessCount: 0,
			Type:        fmt.Sprintf("%T", value),
		}
	}
	
	return nil
}

// contextKey is the key type for storing PluginContext in context.Context
type contextKey struct{}

// WithPluginContext adds a PluginContext to the context
func WithPluginContext(ctx context.Context, pc PluginContext) context.Context {
	return context.WithValue(ctx, contextKey{}, pc)
}

// GetPluginContext retrieves the PluginContext from the context
func GetPluginContext(ctx context.Context) (PluginContext, error) {
	pc, ok := ctx.Value(contextKey{}).(PluginContext)
	if !ok {
		return nil, ErrContextNotFound
	}
	return pc, nil
}

// MustGetPluginContext retrieves the PluginContext or panics if not found
func MustGetPluginContext(ctx context.Context) PluginContext {
	pc, err := GetPluginContext(ctx)
	if err != nil {
		panic(fmt.Sprintf("plugin context not found: %v", err))
	}
	return pc
}