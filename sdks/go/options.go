package scopeconfig

import (
	"crypto/tls"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*clientConfig)

// clientConfig holds the configuration for the Client.
type clientConfig struct {
	address     string
	dialOptions []grpc.DialOption

	// Cache settings
	cacheEnabled bool
	cacheTTL     time.Duration

	// Background sync settings
	syncEnabled  bool
	syncInterval time.Duration
}

// WithAddress sets the server address for the client.
// Example: "localhost:50051"
func WithAddress(address string) ClientOption {
	return func(c *clientConfig) {
		c.address = address
	}
}

// WithInsecure configures the client to use an insecure connection (no TLS).
// This should only be used for development and testing.
func WithInsecure() ClientOption {
	return func(c *clientConfig) {
		c.dialOptions = append(c.dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
}

// WithTLS configures the client to use TLS with the provided configuration.
func WithTLS(tlsConfig *tls.Config) ClientOption {
	return func(c *clientConfig) {
		c.dialOptions = append(c.dialOptions, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}
}

// WithDialOptions adds custom gRPC dial options.
func WithDialOptions(opts ...grpc.DialOption) ClientOption {
	return func(c *clientConfig) {
		c.dialOptions = append(c.dialOptions, opts...)
	}
}

// WithCache enables in-memory caching with the specified TTL.
// When caching is enabled, GetConfigCached and GetValue will use cached values
// when available, and fall back to stale cache data if the server is unavailable.
//
// Example:
//
//	client, err := NewClient(
//	    WithAddress("localhost:50051"),
//	    WithInsecure(),
//	    WithCache(time.Minute),
//	)
func WithCache(ttl time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.cacheEnabled = true
		c.cacheTTL = ttl
	}
}

// WithBackgroundSync enables background synchronization of cached configurations.
// This should be used together with WithCache.
// The sync interval determines how often the client refreshes cached configurations.
//
// Example:
//
//	client, err := NewClient(
//	    WithAddress("localhost:50051"),
//	    WithInsecure(),
//	    WithCache(time.Minute),
//	    WithBackgroundSync(30*time.Second),
//	)
func WithBackgroundSync(interval time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.syncEnabled = true
		c.syncInterval = interval
	}
}
