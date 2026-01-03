package scopeconfig

import (
	"testing"
	"time"

	configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
)

func TestNewConfigCache(t *testing.T) {
	cache := newConfigCache(time.Minute)
	if cache == nil {
		t.Fatal("Expected cache to be created")
	}
	if cache.ttl != time.Minute {
		t.Errorf("Expected TTL to be 1 minute, got %v", cache.ttl)
	}
	if cache.configs == nil {
		t.Error("Expected configs map to be initialized")
	}
	if cache.templates == nil {
		t.Error("Expected templates map to be initialized")
	}
}

func TestCacheKey(t *testing.T) {
	tests := []struct {
		name       string
		identifier *configv1.ConfigIdentifier
	}{
		{
			name: "basic identifier",
			identifier: &configv1.ConfigIdentifier{
				ServiceName: "test-service",
				GroupId:     "test-group",
				Scope:       configv1.Scope_SYSTEM,
			},
		},
		{
			name: "identifier with project",
			identifier: &configv1.ConfigIdentifier{
				ServiceName: "test-service",
				GroupId:     "test-group",
				Scope:       configv1.Scope_PROJECT,
				ProjectId:   "proj-123",
			},
		},
		{
			name: "identifier with store",
			identifier: &configv1.ConfigIdentifier{
				ServiceName: "test-service",
				GroupId:     "test-group",
				Scope:       configv1.Scope_STORE,
				ProjectId:   "proj-123",
				StoreId:     "store-456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := cacheKey(tt.identifier)
			if key == "" {
				t.Error("Expected non-empty cache key")
			}

			// Same identifier should produce same key
			key2 := cacheKey(tt.identifier)
			if key != key2 {
				t.Error("Same identifier should produce same cache key")
			}
		})
	}
}

func TestCacheKeyUniqueness(t *testing.T) {
	id1 := &configv1.ConfigIdentifier{
		ServiceName: "service-a",
		GroupId:     "group",
		Scope:       configv1.Scope_SYSTEM,
	}
	id2 := &configv1.ConfigIdentifier{
		ServiceName: "service-b",
		GroupId:     "group",
		Scope:       configv1.Scope_SYSTEM,
	}
	id3 := &configv1.ConfigIdentifier{
		ServiceName: "service-a",
		GroupId:     "group",
		Scope:       configv1.Scope_PROJECT,
		ProjectId:   "proj-123",
	}

	key1 := cacheKey(id1)
	key2 := cacheKey(id2)
	key3 := cacheKey(id3)

	if key1 == key2 {
		t.Error("Different services should have different cache keys")
	}
	if key1 == key3 {
		t.Error("Different scopes should have different cache keys")
	}
}

func TestConfigCacheSetGet(t *testing.T) {
	cache := newConfigCache(time.Minute)

	identifier := &configv1.ConfigIdentifier{
		ServiceName: "test-service",
		GroupId:     "test-group",
		Scope:       configv1.Scope_SYSTEM,
	}

	config := &configv1.ScopeConfig{
		Fields: []*configv1.ConfigField{
			{Path: "test.key", Value: "test-value"},
		},
	}

	// Initially should not be found
	result, valid := cache.get(identifier)
	if result != nil || valid {
		t.Error("Expected cache miss for new identifier")
	}

	// Set and get
	cache.set(identifier, config)
	result, valid = cache.get(identifier)
	if result == nil {
		t.Fatal("Expected to get cached config")
	}
	if !valid {
		t.Error("Expected cache to be valid (not expired)")
	}
	if len(result.Fields) != 1 || result.Fields[0].Path != "test.key" {
		t.Error("Cached config data mismatch")
	}
}

func TestConfigCacheExpiration(t *testing.T) {
	// Use very short TTL for testing
	cache := newConfigCache(10 * time.Millisecond)

	identifier := &configv1.ConfigIdentifier{
		ServiceName: "test-service",
		GroupId:     "test-group",
		Scope:       configv1.Scope_SYSTEM,
	}

	config := &configv1.ScopeConfig{
		Fields: []*configv1.ConfigField{
			{Path: "test.key", Value: "test-value"},
		},
	}

	cache.set(identifier, config)

	// Should be valid immediately
	_, valid := cache.get(identifier)
	if !valid {
		t.Error("Cache should be valid immediately after set")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should be expired but still return data
	result, valid := cache.get(identifier)
	if valid {
		t.Error("Cache should be expired after TTL")
	}
	if result == nil {
		t.Error("Expired cache should still return data (for stale fallback)")
	}
}

func TestConfigCacheStale(t *testing.T) {
	cache := newConfigCache(10 * time.Millisecond)

	identifier := &configv1.ConfigIdentifier{
		ServiceName: "test-service",
		GroupId:     "test-group",
		Scope:       configv1.Scope_SYSTEM,
	}

	config := &configv1.ScopeConfig{
		Fields: []*configv1.ConfigField{
			{Path: "stale.key", Value: "stale-value"},
		},
	}

	cache.set(identifier, config)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// getStale should return data regardless of expiration
	result := cache.getStale(identifier)
	if result == nil {
		t.Error("getStale should return data even when expired")
	}
	if result.Fields[0].Value != "stale-value" {
		t.Error("getStale should return correct data")
	}
}

func TestConfigCacheInvalidate(t *testing.T) {
	cache := newConfigCache(time.Minute)

	identifier := &configv1.ConfigIdentifier{
		ServiceName: "test-service",
		GroupId:     "test-group",
		Scope:       configv1.Scope_SYSTEM,
	}

	config := &configv1.ScopeConfig{}

	cache.set(identifier, config)

	// Verify it's cached
	result, _ := cache.get(identifier)
	if result == nil {
		t.Fatal("Config should be cached")
	}

	// Invalidate
	cache.invalidate(identifier)

	// Should no longer be found
	result, valid := cache.get(identifier)
	if result != nil || valid {
		t.Error("Invalidated cache should return nil")
	}
}

func TestConfigCacheClear(t *testing.T) {
	cache := newConfigCache(time.Minute)

	// Add multiple configs
	for i := 0; i < 3; i++ {
		identifier := &configv1.ConfigIdentifier{
			ServiceName: "test-service",
			GroupId:     "group-" + string(rune('a'+i)),
			Scope:       configv1.Scope_SYSTEM,
		}
		cache.set(identifier, &configv1.ScopeConfig{})
	}

	// Clear all
	cache.clear()

	// All should be gone
	for i := 0; i < 3; i++ {
		identifier := &configv1.ConfigIdentifier{
			ServiceName: "test-service",
			GroupId:     "group-" + string(rune('a'+i)),
			Scope:       configv1.Scope_SYSTEM,
		}
		result, _ := cache.get(identifier)
		if result != nil {
			t.Error("Cache should be empty after clear")
		}
	}
}

func TestTemplateCacheSetGet(t *testing.T) {
	cache := newConfigCache(time.Minute)

	identifier := &configv1.ConfigIdentifier{
		ServiceName: "test-service",
		GroupId:     "test-group",
	}

	template := &configv1.ConfigTemplate{
		GroupLabel: "Test Group",
		Fields: []*configv1.ConfigFieldTemplate{
			{Path: "test.field", DefaultValue: "default"},
		},
	}

	// Initially should not be found
	result, valid := cache.getTemplate(identifier)
	if result != nil || valid {
		t.Error("Expected cache miss for new template identifier")
	}

	// Set and get
	cache.setTemplate(identifier, template)
	result, valid = cache.getTemplate(identifier)
	if result == nil {
		t.Fatal("Expected to get cached template")
	}
	if !valid {
		t.Error("Expected template cache to be valid")
	}
	if result.GroupLabel != "Test Group" {
		t.Error("Cached template data mismatch")
	}
}

func TestTemplateCacheExpiration(t *testing.T) {
	cache := newConfigCache(10 * time.Millisecond)

	identifier := &configv1.ConfigIdentifier{
		ServiceName: "test-service",
		GroupId:     "test-group",
	}

	template := &configv1.ConfigTemplate{GroupLabel: "Test"}

	cache.setTemplate(identifier, template)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should be expired
	_, valid := cache.getTemplate(identifier)
	if valid {
		t.Error("Template cache should be expired")
	}
}

func TestTemplateCacheStale(t *testing.T) {
	cache := newConfigCache(10 * time.Millisecond)

	identifier := &configv1.ConfigIdentifier{
		ServiceName: "test-service",
		GroupId:     "test-group",
	}

	template := &configv1.ConfigTemplate{GroupLabel: "Stale Template"}

	cache.setTemplate(identifier, template)
	time.Sleep(20 * time.Millisecond)

	result := cache.getTemplateStale(identifier)
	if result == nil {
		t.Error("getTemplateStale should return data even when expired")
	}
	if result.GroupLabel != "Stale Template" {
		t.Error("getTemplateStale should return correct data")
	}
}

func TestTemplateCacheInvalidate(t *testing.T) {
	cache := newConfigCache(time.Minute)

	identifier := &configv1.ConfigIdentifier{
		ServiceName: "test-service",
		GroupId:     "test-group",
	}

	cache.setTemplate(identifier, &configv1.ConfigTemplate{})
	cache.invalidateTemplate(identifier)

	result, _ := cache.getTemplate(identifier)
	if result != nil {
		t.Error("Invalidated template should not be found")
	}
}

func TestCacheStop(t *testing.T) {
	cache := newConfigCache(time.Minute)

	// Should not panic when called multiple times
	cache.stop()
	cache.stop()
}

func TestTemplateCacheKey(t *testing.T) {
	id1 := &configv1.ConfigIdentifier{
		ServiceName: "service-a",
		GroupId:     "group-a",
	}
	id2 := &configv1.ConfigIdentifier{
		ServiceName: "service-a",
		GroupId:     "group-b",
	}

	key1 := templateCacheKey(id1)
	key2 := templateCacheKey(id2)

	if key1 == key2 {
		t.Error("Different groups should have different template cache keys")
	}
}
