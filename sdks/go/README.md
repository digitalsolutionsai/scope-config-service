# Go SDK for ScopeConfig Service

A lightweight, idiomatic Go client for interacting with the ScopeConfig gRPC service with caching support.

## Features

- **In-memory caching** for config values by group (reduces gRPC calls)
- **Template caching** for default value lookups
- **Background sync** to refresh cached configs periodically
- **Stale cache fallback** when server is unavailable
- **GetValue** with inheritance and default value support
- **Environment variable support** for configuration
- **Automatic template loading** from YAML files

## Prerequisites

- Go 1.24 or later

## Installation

### Install via `go get` (Recommended)

The SDK can be installed directly from GitHub with generated proto files included:

```bash
go get github.com/digitalsolutionsai/scope-config-service/sdks/go@latest
```

Or install a specific version:

```bash
go get github.com/digitalsolutionsai/scope-config-service/sdks/go@v1.0.0
```

### Private Repository Access

If this is a private repository, configure Git to use SSH:

```bash
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

Or set the GOPRIVATE environment variable:

```bash
export GOPRIVATE=github.com/digitalsolutionsai/*
```

### Alternative: Manual Installation

If you prefer to copy the SDK manually:

1. Copy the entire `sdks/go` directory to your project
2. Update the module path in `go.mod` to match your project
3. Update import paths in all `.go` files
4. Run `go mod tidy`

## Usage

### Using Environment Variables

```go
package main

import (
    "context"
    "log"

    scopeconfig "github.com/digitalsolutionsai/scope-config-service/sdks/go"
    configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
)

func main() {
    // Create a client using environment variables:
    // GRPC_SCOPE_CONFIG_HOST (default: localhost)
    // GRPC_SCOPE_CONFIG_PORT (default: 50051)
    // GRPC_SCOPE_CONFIG_USE_TLS (default: false)
    client, err := scopeconfig.NewClient(scopeconfig.FromEnvironment()...)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    identifier := scopeconfig.NewIdentifier("my-service").
        WithScope(configv1.Scope_PROJECT).
        WithGroupID("database").
        WithProjectID("proj-123").
        Build()

    // Get a specific config value with inheritance
    value, err := client.GetValue(ctx, identifier, "database.host", &scopeconfig.GetValueOptions{
        UseDefault: true,
        Inherit:    true,
    })
    if err != nil {
        log.Fatal(err)
    }
    if value != nil {
        log.Printf("Database host: %s", *value)
    }
}
```

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    scopeconfig "github.com/digitalsolutionsai/scope-config-service/sdks/go"
    configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
)

func main() {
    // Create a client
    client, err := scopeconfig.NewClient(
        scopeconfig.WithAddress("localhost:50051"),
        scopeconfig.WithInsecure(), // Use WithTLS() in production
        scopeconfig.WithCache(time.Minute),
        scopeconfig.WithBackgroundSync(30*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Build a config identifier
    identifier := scopeconfig.NewIdentifier("my-service").
        WithScope(configv1.Scope_SYSTEM).
        WithGroupID("database").
        Build()

    // Get configuration with caching
    config, err := client.GetConfigCached(ctx, identifier)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Configuration for %s:\n", config.VersionInfo.Identifier.ServiceName)
    for _, field := range config.Fields {
        fmt.Printf("  %s = %s\n", field.Path, field.Value)
    }
}
```

### Update Configuration

```go
import (
    scopeconfig "github.com/digitalsolutionsai/scope-config-service/sdks/go"
    configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
)

// Update config fields
identifier := scopeconfig.NewIdentifier("payment-service").
    WithScope(configv1.Scope_PROJECT).
    WithGroupID("api").
    WithProjectID("proj-123").
    Build()

fields := []*configv1.ConfigField{
    {
        Path:  "api.timeout",
        Value: "30s",
        Type:  configv1.FieldType_STRING,
    },
    {
        Path:  "api.retry_count",
        Value: "3",
        Type:  configv1.FieldType_INT,
    },
}

config, err := client.UpdateConfig(ctx, identifier, fields, "admin@example.com")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Updated config to version %d\n", config.CurrentVersion)
```

### Apply Configuration Template

```go
// Create a configuration template
template := &configv1.ConfigTemplate{
    Identifier: scopeconfig.NewIdentifier("my-service").
        WithGroupID("logging").
        Build(),
    ServiceLabel:     "My Service",
    GroupLabel:       "Logging Configuration",
    GroupDescription: "Controls logging behavior for the application",
    Fields: []*configv1.ConfigFieldTemplate{
        {
            Path:         "log.level",
            Label:        "Log Level",
            Description:  "Application logging level",
            Type:         configv1.FieldType_STRING,
            DefaultValue: "INFO",
            Options: []*configv1.ValueOption{
                {Value: "DEBUG", Label: "Debug"},
                {Value: "INFO", Label: "Info"},
                {Value: "WARN", Label: "Warning"},
                {Value: "ERROR", Label: "Error"},
            },
        },
    },
}

result, err := client.ApplyConfigTemplate(ctx, template, "admin@example.com")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Applied template: %s\n", result.GroupLabel)
```

### Get Configuration Template

```go
identifier := scopeconfig.NewIdentifier("my-service").
    WithGroupID("logging").
    Build()

template, err := client.GetConfigTemplate(ctx, identifier)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Template: %s\n", template.GroupLabel)
fmt.Printf("Description: %s\n", template.GroupDescription)
fmt.Printf("Fields:\n")
for _, field := range template.Fields {
    fmt.Printf("  - %s (%s): %s\n", field.Label, field.Path, field.Description)
}
```

### Using Caching

```go
// Create a client with caching enabled
client, err := scopeconfig.NewClient(
    scopeconfig.WithAddress("localhost:50051"),
    scopeconfig.WithInsecure(),
    scopeconfig.WithCache(time.Minute),           // Cache TTL: 1 minute
    scopeconfig.WithBackgroundSync(30*time.Second), // Sync every 30 seconds
)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Get config with caching (reduces gRPC calls)
config, err := client.GetConfigCached(ctx, identifier)
```

### Get Specific Config Value

```go
// Get a specific value with options
value, err := client.GetValue(ctx, identifier, "database.host", &scopeconfig.GetValueOptions{
    UseDefault: true,  // Use template default if not set
    Inherit:    true,  // Check parent scopes if not found
})
if err != nil {
    log.Fatal(err)
}
if value != nil {
    fmt.Printf("Database host: %s\n", *value)
}

// Or use GetValueString for convenience (returns empty string if not found)
host, err := client.GetValueString(ctx, identifier, "database.host", &scopeconfig.GetValueOptions{
    UseDefault: true,
})
```

## Automatic Template Loading

The SDK supports automatic loading of configuration templates from YAML files. Simply place your template files in a `templates` directory and the SDK will load them automatically.

### Quick Start

1. Create a `templates` directory in your project root
2. Add your YAML template files (`.yaml` or `.yml`)
3. Call `LoadTemplatesFromDir` on client initialization

```go
// Initialize client and auto-load templates
client, err := scopeconfig.NewClient(scopeconfig.FromEnvironment()...)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Auto-load all templates from the templates directory
err = client.LoadTemplatesFromDir(ctx, "./templates", "system")
if err != nil {
    log.Fatal(err)
}
```

### Template File Format

Create YAML files in your `templates` directory following this structure:

```yaml
# templates/my-service.yaml
service:
  id: "my-service"
  label: "My Service"

groups:
  - id: "database"
    label: "Database Configuration"
    description: "Database connection settings"
    sortOrder: 100000
    fields:
      - path: "host"
        label: "Database Host"
        description: "The database server hostname"
        type: "STRING"
        defaultValue: "localhost"
        sortOrder: 100000
        displayOn:
          - "PROJECT"
          - "STORE"
      - path: "port"
        label: "Database Port"
        type: "INT"
        defaultValue: "5432"
        sortOrder: 200000
        displayOn:
          - "PROJECT"
      - path: "ssl-enabled"
        label: "Enable SSL"
        type: "BOOLEAN"
        defaultValue: "false"
        sortOrder: 300000
        displayOn:
          - "PROJECT"
```

### Field Types

| Type | Description | Example |
|------|-------------|---------|
| `STRING` | Text value | `"localhost"` |
| `INT` | Integer number | `"5432"` |
| `FLOAT` | Decimal number | `"0.7"` |
| `BOOLEAN` | True/false | `"true"` |
| `JSON` | JSON object/array | `'["a", "b"]'` |
| `ARRAY_STRING` | String array | |
| `SECRET` | Sensitive value | API keys, passwords |

### Display Scopes

The `displayOn` field controls which scopes the field is visible/editable:
- `SYSTEM` - System-wide settings
- `PROJECT` - Project-level settings
- `STORE` - Store-level settings
- `USER` - User-level settings

### Options (Dropdowns)

Define selectable options for a field:

```yaml
- path: "log-level"
  label: "Log Level"
  type: "STRING"
  defaultValue: "INFO"
  options:
    - value: "DEBUG"
      label: "Debug"
    - value: "INFO"
      label: "Info"
    - value: "WARN"
      label: "Warning"
    - value: "ERROR"
      label: "Error"
```

### Load Templates from Directory

```go
// Load and apply all YAML templates from a directory
err := client.LoadTemplatesFromDir(ctx, "./templates", "system")
if err != nil {
    log.Fatal(err)
}
```

### Using TLS in Production

```go
import (
    "crypto/tls"
    "crypto/x509"
    "os"
)

// Load CA certificate
caCert, err := os.ReadFile("ca-cert.pem")
if err != nil {
    log.Fatal(err)
}

certPool := x509.NewCertPool()
certPool.AppendCertsFromPEM(caCert)

tlsConfig := &tls.Config{
    RootCAs: certPool,
}

client, err := scopeconfig.NewClient(
    scopeconfig.WithAddress("config-service.example.com:443"),
    scopeconfig.WithTLS(tlsConfig),
)
```

## API Reference

### Client Methods

- `NewClient(opts ...ClientOption) (*Client, error)` - Create a new client
- `Close() error` - Close the client connection
- `GetConfig(ctx, identifier) (*ScopeConfig, error)` - Get published configuration (always fetches from server)
- `GetConfigCached(ctx, identifier) (*ScopeConfig, error)` - Get configuration with caching support
- `GetLatestConfig(ctx, identifier) (*ScopeConfig, error)` - Get latest configuration
- `GetConfigByVersion(ctx, identifier, version) (*ScopeConfig, error)` - Get configuration by specific version
- `GetConfigHistory(ctx, identifier, limit) (*GetConfigHistoryResponse, error)` - Get version history
- `UpdateConfig(ctx, identifier, fields, user) (*ScopeConfig, error)` - Update configuration
- `PublishVersion(ctx, identifier, version, user) (*ConfigVersion, error)` - Mark version as published
- `DeleteConfig(ctx, identifier) error` - Delete configuration and all versions
- `GetConfigTemplate(ctx, identifier) (*ConfigTemplate, error)` - Get configuration template
- `GetConfigTemplateCached(ctx, identifier) (*ConfigTemplate, error)` - Get template with caching (for default values)
- `ApplyConfigTemplate(ctx, template, user) (*ConfigTemplate, error)` - Apply configuration template
- `ListConfigTemplates(ctx, serviceName, isActive) (*ListConfigTemplatesResponse, error)` - List templates
- `GetValue(ctx, identifier, path, opts) (*string, error)` - Get specific config value with options
- `GetValueString(ctx, identifier, path, opts) (string, error)` - Get value as string (empty if not found)
- `MustGetValue(ctx, identifier, path, opts) string` - Get value or panic on error
- `LoadTemplatesFromDir(ctx, dir, user) error` - Load and apply templates from directory
- `InvalidateCache(identifier)` - Invalidate cache for specific config
- `ClearCache()` - Clear all cached configs
- `IsCacheEnabled() bool` - Check if caching is enabled

### Client Options

- `WithAddress(address string)` - Set server address
- `WithInsecure()` - Use insecure connection (development only)
- `WithTLS(tlsConfig *tls.Config)` - Use TLS connection
- `WithDialOptions(opts ...grpc.DialOption)` - Add custom gRPC dial options
- `WithCache(ttl time.Duration)` - Enable caching with specified TTL (default: 1 minute)
- `WithBackgroundSync(interval time.Duration)` - Enable background sync (default: 30 seconds)

### GetValue Options

```go
type GetValueOptions struct {
    UseDefault bool  // Use default value from template if not set
    Inherit    bool  // Traverse parent scopes (STORE → PROJECT → SYSTEM, USER → SYSTEM)
}
```

### Caching Behavior

- **Config values** are cached by group to reduce gRPC calls
- **Templates** are cached for default value lookups
- **Stale cache fallback**: If server is unavailable, returns stale cached data
- **Background sync**: Periodically refreshes cached configs in the background

### Identifier Builder

- `NewIdentifier(serviceName string)` - Create builder with service name
- `WithScope(scope Scope)` - Set scope
- `WithGroupID(groupID string)` - Set group ID
- `WithProjectID(projectID string)` - Set project ID
- `WithStoreID(storeID string)` - Set store ID
- `WithUserID(userID string)` - Set user ID
- `Build()` - Build the ConfigIdentifier

### Scope Constants

Available in `configv1.Scope`:
- `Scope_SCOPE_UNSPECIFIED`
- `Scope_SYSTEM`
- `Scope_PROJECT`
- `Scope_STORE`
- `Scope_USER`

### Error Handling

The SDK provides custom error types and helper functions for error inspection:

```go
import scopeconfig "github.com/digitalsolutionsai/scope-config-service/sdks/go"

config, err := client.GetConfig(ctx, identifier)
if err != nil {
    if scopeconfig.IsNotFound(err) {
        // Handle not found
        log.Println("Configuration not found")
    } else if scopeconfig.IsServerUnavailable(err) {
        // Handle server unavailable
        log.Println("Server is unavailable")
    } else if scopeconfig.IsInvalidArgument(err) {
        // Handle invalid argument
        log.Println("Invalid request")
    } else {
        // Handle other errors
        log.Printf("Error: %v", err)
    }
}
```

Available error helpers:
- `IsNotFound(err)` - Check if resource was not found
- `IsServerUnavailable(err)` - Check if server is unavailable
- `IsInvalidArgument(err)` - Check if argument was invalid
- `IsPermissionDenied(err)` - Check if permission was denied
- `IsAlreadyExists(err)` - Check if resource already exists
- `GRPCCode(err)` - Get the gRPC status code from the error

Sentinel errors:
- `ErrConfigNotFound`
- `ErrTemplateNotFound`
- `ErrServerUnavailable`
- `ErrInvalidArgument`
- `ErrPermissionDenied`
- `ErrAlreadyExists`

## Examples

See the `examples/` directory for complete working examples:

- `examples/main.go` - Comprehensive example demonstrating all SDK features

Run the example:

```bash
# Run the example (proto files are already generated)
go run examples/main.go
```

## Testing

### Unit Tests

Run the unit tests:

```bash
# Run unit tests only
make test-unit

# Or directly:
go test -v -short ./...
```

### Integration Tests

Integration tests use testcontainers-go to spin up PostgreSQL and the ScopeConfig service:

```bash
# Run integration tests (requires Docker)
make test-integration

# Or directly:
go test -v -tags=integration -timeout=10m ./tests/...

# With a pre-built image:
SCOPE_CONFIG_IMAGE=scope-config-service:test go test -v -tags=integration ./tests/...
```

### Test Coverage

```bash
make test-coverage
```

## Makefile Targets

```bash
make build            # Build the SDK
make test             # Run all tests
make test-unit        # Run unit tests only
make test-integration # Run integration tests (requires Docker)
make test-coverage    # Run tests with coverage report
make lint             # Run linter
make fmt              # Format code
make clean            # Clean build artifacts
make deps             # Install dependencies
make help             # Show all targets
```

## License

See the main project LICENSE file.
