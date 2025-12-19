/*
Go client for the ScopeConfig service using gRPC.

Features:
  - In-memory caching for config values by group with configurable TTL (default: 1 minute)
  - In-memory caching for templates (for default value lookups)
  - Background sync to refresh cached config values periodically
  - Stale cache fallback when server is unavailable
  - GetValue extracts specific field from cached group config (reduces gRPC calls)
  - GetValue with inheritance and default value support
  - Template import functionality
*/
package scopeconfig

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Default cache settings
const (
	DefaultCacheTTL      = 1 * time.Minute
	DefaultSyncInterval  = 30 * time.Second
	DefaultSyncBatchSize = 10
)

// Client is a gRPC client for the ScopeConfig service with caching support.
type Client struct {
	conn   *grpc.ClientConn
	client configv1.ConfigServiceClient

	// Cache configuration
	cache        *configCache
	cacheEnabled bool

	// Background sync
	syncInterval time.Duration
	syncStopChan chan struct{}
	syncWg       sync.WaitGroup
	syncMu       sync.Mutex
	syncTargets  []*configv1.ConfigIdentifier
}

/*
NewClient creates a new ScopeConfig client with the provided options.

Example:

	client, err := NewClient(
	    WithAddress("localhost:50051"),
	    WithInsecure(),
	)
	if err != nil {
	    log.Fatal(err)
	}
	defer client.Close()

Example with caching:

	client, err := NewClient(
	    WithAddress("localhost:50051"),
	    WithInsecure(),
	    WithCache(time.Minute),
	    WithBackgroundSync(30*time.Second),
	)
*/
func NewClient(opts ...ClientOption) (*Client, error) {
	cfg := &clientConfig{
		cacheTTL:     DefaultCacheTTL,
		syncInterval: DefaultSyncInterval,
	}

	// Apply all options
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate configuration
	if cfg.address == "" {
		return nil, fmt.Errorf("address is required")
	}

	if len(cfg.dialOptions) == 0 {
		return nil, fmt.Errorf("transport credentials required (use WithInsecure() or WithTLS())")
	}

	// Establish connection
	conn, err := grpc.NewClient(cfg.address, cfg.dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", cfg.address, err)
	}

	c := &Client{
		conn:         conn,
		client:       configv1.NewConfigServiceClient(conn),
		cacheEnabled: cfg.cacheEnabled,
		syncInterval: cfg.syncInterval,
		syncStopChan: make(chan struct{}),
		syncTargets:  make([]*configv1.ConfigIdentifier, 0),
	}

	// Initialize cache if enabled
	if cfg.cacheEnabled {
		c.cache = newConfigCache(cfg.cacheTTL)

		// Start background sync if enabled
		if cfg.syncEnabled {
			c.startBackgroundSync()
		}
	}

	return c, nil
}

// Close closes the underlying gRPC connection and stops background sync.
func (c *Client) Close() error {
	// Stop background sync
	c.stopBackgroundSync()

	// Stop cache
	if c.cache != nil {
		c.cache.stop()
	}

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

/*
GetConfig retrieves the published configuration for the given identifier.
This method bypasses the cache and always fetches from the server.

Example:

	identifier := NewIdentifier("my-service").
	    WithScope(configv1.Scope_SYSTEM).
	    WithGroupID("my-group").
	    Build()
	config, err := client.GetConfig(ctx, identifier)
*/
func (c *Client) GetConfig(ctx context.Context, identifier *configv1.ConfigIdentifier) (*configv1.ScopeConfig, error) {
	req := &configv1.GetConfigRequest{
		Identifier: identifier,
	}

	resp, err := c.client.GetConfig(ctx, req)
	if err != nil {
		return nil, wrapError("GetConfig", err)
	}

	// Update cache if enabled
	if c.cacheEnabled && c.cache != nil {
		c.cache.set(identifier, resp)
		c.addSyncTarget(identifier)
	}

	return resp, nil
}

/*
GetConfigCached retrieves the configuration with caching support.
If the cache is enabled and has a valid entry, it returns the cached value.
If the server is unavailable, it falls back to stale cache data.

Example:

	config, err := client.GetConfigCached(ctx, identifier)
*/
func (c *Client) GetConfigCached(ctx context.Context, identifier *configv1.ConfigIdentifier) (*configv1.ScopeConfig, error) {
	// Try cache first if enabled
	if c.cacheEnabled && c.cache != nil {
		if config, valid := c.cache.get(identifier); valid {
			return config, nil
		}
	}

	// Fetch from server
	req := &configv1.GetConfigRequest{
		Identifier: identifier,
	}

	resp, err := c.client.GetConfig(ctx, req)
	if err != nil {
		// On error, try to return stale cache
		if c.cacheEnabled && c.cache != nil {
			if stale := c.cache.getStale(identifier); stale != nil {
				log.Printf("Using stale cache for %s/%s due to error: %v",
					identifier.ServiceName, identifier.GroupId, err)
				return stale, nil
			}
		}
		return nil, wrapError("GetConfigCached", err)
	}

	// Update cache
	if c.cacheEnabled && c.cache != nil {
		c.cache.set(identifier, resp)
		c.addSyncTarget(identifier)
	}

	return resp, nil
}

// GetLatestConfig retrieves the latest configuration (published or not) for the given identifier.
func (c *Client) GetLatestConfig(ctx context.Context, identifier *configv1.ConfigIdentifier) (*configv1.ScopeConfig, error) {
    req := &configv1.GetConfigRequest{
        Identifier: identifier,
    }

    resp, err := c.client.GetLatestConfig(ctx, req)
    if err != nil {
        return nil, wrapError("GetLatestConfig", err)
    }

    return resp, nil
}

/*
UpdateConfig creates or updates a configuration with the provided fields.

Example:

    identifier := NewIdentifier("my-service").
        WithScope(configv1.Scope_SYSTEM).
        WithGroupID("my-group").
        Build()
    fields := []*configv1.ConfigField{
        {Path: "log.level", Value: "INFO", Type: configv1.FieldType_STRING},
    }
    config, err := client.UpdateConfig(ctx, identifier, fields, "user@example.com")
*/
func (c *Client) UpdateConfig(
    ctx context.Context,
    identifier *configv1.ConfigIdentifier,
    fields []*configv1.ConfigField,
    user string,
) (*configv1.ScopeConfig, error) {
    req := &configv1.UpdateConfigRequest{
        Identifier: identifier,
        Fields:     fields,
        User:       user,
    }

    resp, err := c.client.UpdateConfig(ctx, req)
    if err != nil {
        return nil, wrapError("UpdateConfig", err)
    }

    return resp, nil
}

/*
GetConfigTemplate retrieves the configuration template for the given identifier.
This method always fetches from the server and does not use cache.

Example:

	identifier := NewIdentifier("my-service").
	    WithGroupID("my-group").
	    Build()
	template, err := client.GetConfigTemplate(ctx, identifier)
*/
func (c *Client) GetConfigTemplate(ctx context.Context, identifier *configv1.ConfigIdentifier) (*configv1.ConfigTemplate, error) {
	req := &configv1.GetConfigTemplateRequest{
		Identifier: identifier,
	}

	resp, err := c.client.GetConfigTemplate(ctx, req)
	if err != nil {
		return nil, wrapError("GetConfigTemplate", err)
	}

	// Update cache if enabled (for default value lookups)
	if c.cacheEnabled && c.cache != nil {
		c.cache.setTemplate(identifier, resp)
	}

	return resp, nil
}

/*
GetConfigTemplateCached retrieves the configuration template with caching support.
Templates are cached to support default value lookups in GetValue without extra gRPC calls.

Example:

	template, err := client.GetConfigTemplateCached(ctx, identifier)
*/
func (c *Client) GetConfigTemplateCached(ctx context.Context, identifier *configv1.ConfigIdentifier) (*configv1.ConfigTemplate, error) {
	// Try cache first if enabled
	if c.cacheEnabled && c.cache != nil {
		if template, valid := c.cache.getTemplate(identifier); valid {
			return template, nil
		}
	}

	// Fetch from server
	req := &configv1.GetConfigTemplateRequest{
		Identifier: identifier,
	}

	resp, err := c.client.GetConfigTemplate(ctx, req)
	if err != nil {
		// On error, try to return stale cache
		if c.cacheEnabled && c.cache != nil {
			if stale := c.cache.getTemplateStale(identifier); stale != nil {
				return stale, nil
			}
		}
		return nil, wrapError("GetConfigTemplateCached", err)
	}

	// Update cache
	if c.cacheEnabled && c.cache != nil {
		c.cache.setTemplate(identifier, resp)
	}

	return resp, nil
}

/*
ApplyConfigTemplate applies a configuration template to the service.

Example:

    template := &configv1.ConfigTemplate{
        Identifier: NewIdentifier("my-service").WithGroupID("my-group").Build(),
        ServiceLabel: "My Service",
        GroupLabel: "My Group",
        GroupDescription: "Configuration for my service",
        Fields: []*configv1.ConfigFieldTemplate{
            {
                Path: "log.level",
                Label: "Log Level",
                Description: "Application logging level",
                Type: configv1.FieldType_STRING,
                DefaultValue: "INFO",
            },
        },
    }
    result, err := client.ApplyConfigTemplate(ctx, template, "user@example.com")
*/
func (c *Client) ApplyConfigTemplate(
    ctx context.Context,
    template *configv1.ConfigTemplate,
    user string,
) (*configv1.ConfigTemplate, error) {
    req := &configv1.ApplyConfigTemplateRequest{
        Template: template,
        User:     user,
    }

    resp, err := c.client.ApplyConfigTemplate(ctx, req)
    if err != nil {
        return nil, wrapError("ApplyConfigTemplate", err)
    }

    return resp, nil
}

/*
Additional methods that can be implemented when needed:

- GetConfigByVersion: Retrieve a specific version of a configuration
- GetConfigHistory: Get version history for a configuration
- PublishVersion: Mark a version as published
- DeleteConfig: Delete a configuration
*/

// wrapError wraps gRPC errors with additional context.
func wrapError(method string, err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("%s failed: %w", method, err)
	}

	switch st.Code() {
	case codes.NotFound:
		return fmt.Errorf("%s: resource not found: %s", method, st.Message())
	case codes.InvalidArgument:
		return fmt.Errorf("%s: invalid argument: %s", method, st.Message())
	case codes.AlreadyExists:
		return fmt.Errorf("%s: resource already exists: %s", method, st.Message())
	case codes.PermissionDenied:
		return fmt.Errorf("%s: permission denied: %s", method, st.Message())
	case codes.Unavailable:
		return fmt.Errorf("%s: service unavailable: %s", method, st.Message())
	default:
		return fmt.Errorf("%s failed with %s: %s", method, st.Code(), st.Message())
	}
}

// addSyncTarget adds a config identifier to the list of targets for background sync.
func (c *Client) addSyncTarget(identifier *configv1.ConfigIdentifier) {
	c.syncMu.Lock()
	defer c.syncMu.Unlock()

	// Check if already in the list
	key := cacheKey(identifier)
	for _, target := range c.syncTargets {
		if cacheKey(target) == key {
			return
		}
	}

	c.syncTargets = append(c.syncTargets, identifier)
}

// startBackgroundSync starts a background goroutine to periodically refresh cached configs.
func (c *Client) startBackgroundSync() {
	c.syncWg.Add(1)
	go func() {
		defer c.syncWg.Done()
		ticker := time.NewTicker(c.syncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-c.syncStopChan:
				return
			case <-ticker.C:
				c.syncCachedConfigs()
			}
		}
	}()
}

// stopBackgroundSync stops the background sync goroutine.
func (c *Client) stopBackgroundSync() {
	select {
	case <-c.syncStopChan:
		// Already closed
	default:
		close(c.syncStopChan)
	}
	c.syncWg.Wait()
}

// syncCachedConfigs refreshes all cached configurations.
func (c *Client) syncCachedConfigs() {
	c.syncMu.Lock()
	targets := make([]*configv1.ConfigIdentifier, len(c.syncTargets))
	copy(targets, c.syncTargets)
	c.syncMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, identifier := range targets {
		req := &configv1.GetConfigRequest{
			Identifier: identifier,
		}

		resp, err := c.client.GetConfig(ctx, req)
		if err != nil {
			log.Printf("Background sync failed for %s/%s: %v",
				identifier.ServiceName, identifier.GroupId, err)
			continue
		}

		if c.cache != nil {
			c.cache.set(identifier, resp)
		}
	}
}

// InvalidateCache invalidates the cache for a specific identifier.
func (c *Client) InvalidateCache(identifier *configv1.ConfigIdentifier) {
	if c.cache != nil {
		c.cache.invalidate(identifier)
	}
}

// ClearCache clears all cached configurations.
func (c *Client) ClearCache() {
	if c.cache != nil {
		c.cache.clear()
	}
}

// IsCacheEnabled returns whether caching is enabled for this client.
func (c *Client) IsCacheEnabled() bool {
	return c.cacheEnabled
}