package scopeconfig

import (
	"crypto/tls"

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
