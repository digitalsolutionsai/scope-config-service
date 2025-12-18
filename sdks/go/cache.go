package scopeconfig

import (
	"sync"
	"time"

	configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
)

// cacheEntry represents a cached configuration with its expiration time.
type cacheEntry struct {
	config    *configv1.ScopeConfig
	expiresAt time.Time
}

// templateCacheEntry represents a cached template with its expiration time.
type templateCacheEntry struct {
	template  *configv1.ConfigTemplate
	expiresAt time.Time
}

// configCache provides thread-safe in-memory caching for configuration values and templates.
// - Config values are cached by group to reduce gRPC calls
// - Templates are cached for default value lookups
type configCache struct {
	mu        sync.RWMutex
	configs   map[string]*cacheEntry
	templates map[string]*templateCacheEntry
	ttl       time.Duration
	stopChan  chan struct{}
}

// newConfigCache creates a new config cache with the specified TTL.
func newConfigCache(ttl time.Duration) *configCache {
	return &configCache{
		configs:   make(map[string]*cacheEntry),
		templates: make(map[string]*templateCacheEntry),
		ttl:       ttl,
		stopChan:  make(chan struct{}),
	}
}

// cacheKey generates a unique key for a config identifier.
func cacheKey(identifier *configv1.ConfigIdentifier) string {
	return identifier.ServiceName + ":" +
		identifier.GroupId + ":" +
		identifier.Scope.String() + ":" +
		identifier.ProjectId + ":" +
		identifier.StoreId + ":" +
		identifier.UserId
}

// templateCacheKey generates a unique key for a template identifier.
func templateCacheKey(identifier *configv1.ConfigIdentifier) string {
	return "template:" + identifier.ServiceName + ":" + identifier.GroupId
}

// get retrieves a config from the cache if it exists and hasn't expired.
// Returns (config, true) if found and valid, (config, false) if stale, (nil, false) if not found.
func (c *configCache) get(identifier *configv1.ConfigIdentifier) (*configv1.ScopeConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := cacheKey(identifier)
	entry, exists := c.configs[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		// Return stale config but indicate it's expired
		return entry.config, false
	}

	return entry.config, true
}

// getStale retrieves a config from the cache even if expired.
func (c *configCache) getStale(identifier *configv1.ConfigIdentifier) *configv1.ScopeConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := cacheKey(identifier)
	entry, exists := c.configs[key]
	if !exists {
		return nil
	}
	return entry.config
}

// set stores a config in the cache with the configured TTL.
func (c *configCache) set(identifier *configv1.ConfigIdentifier, config *configv1.ScopeConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := cacheKey(identifier)
	c.configs[key] = &cacheEntry{
		config:    config,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// getTemplate retrieves a template from the cache if it exists and hasn't expired.
// Returns (template, true) if found and valid, (template, false) if stale, (nil, false) if not found.
func (c *configCache) getTemplate(identifier *configv1.ConfigIdentifier) (*configv1.ConfigTemplate, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := templateCacheKey(identifier)
	entry, exists := c.templates[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return entry.template, false
	}

	return entry.template, true
}

// getTemplateStale retrieves a template from the cache even if expired.
func (c *configCache) getTemplateStale(identifier *configv1.ConfigIdentifier) *configv1.ConfigTemplate {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := templateCacheKey(identifier)
	entry, exists := c.templates[key]
	if !exists {
		return nil
	}
	return entry.template
}

// setTemplate stores a template in the cache with the configured TTL.
func (c *configCache) setTemplate(identifier *configv1.ConfigIdentifier, template *configv1.ConfigTemplate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := templateCacheKey(identifier)
	c.templates[key] = &templateCacheEntry{
		template:  template,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// invalidate removes a specific config from the cache.
func (c *configCache) invalidate(identifier *configv1.ConfigIdentifier) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := cacheKey(identifier)
	delete(c.configs, key)
}

// invalidateTemplate removes a specific template from the cache.
func (c *configCache) invalidateTemplate(identifier *configv1.ConfigIdentifier) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := templateCacheKey(identifier)
	delete(c.templates, key)
}

// clear removes all entries from the cache.
func (c *configCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.configs = make(map[string]*cacheEntry)
	c.templates = make(map[string]*templateCacheEntry)
}

// stop signals the cache to stop any background operations.
func (c *configCache) stop() {
	select {
	case <-c.stopChan:
		// Already closed
	default:
		close(c.stopChan)
	}
}
