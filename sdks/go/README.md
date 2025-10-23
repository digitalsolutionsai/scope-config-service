# Go SDK for ScopeConfig Service

A lightweight, idiomatic Go client for interacting with the ScopeConfig gRPC service.

## Prerequisites

- Go 1.24 or later
- `buf` CLI for protobuf generation ([installation guide](https://buf.build/docs/installation))
- `protoc-gen-go` and `protoc-gen-go-grpc` plugins

Install the required Go plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Installation

### 1. Copy the SDK to your project

```bash
# Copy the entire sdks/go directory to your project
cp -r sdks/go /path/to/your/project/scopeconfig-sdk
cd /path/to/your/project/scopeconfig-sdk
```

### 2. Update the module path

Edit `go.mod` and change the module name to match your project:

```go
module github.com/your-org/your-project/scopeconfig-sdk
```

Also update the import paths in:
- `identifier.go`
- `client.go`

Change:
```go
configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
```

To:
```go
configv1 "github.com/your-org/your-project/scopeconfig-sdk/gen/config/v1"
```

### 3. Copy the proto files

```bash
# Copy proto files to the SDK directory
mkdir -p proto
cp -r /path/to/scope-config-service/proto/config proto/
```

### 4. Generate the protobuf code

```bash
# Ensure protoc-gen-go and protoc-gen-go-grpc are in your PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# Generate the code using buf
buf generate
```

This will generate the gRPC client code in the `gen/` directory (which is gitignored).

### 5. Install dependencies

```bash
go mod tidy
```

## Usage

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    scopeconfig "github.com/your-org/your-project/scopeconfig-sdk"
    configv1 "github.com/your-org/your-project/scopeconfig-sdk/gen/config/v1"
)

func main() {
    // Create a client
    client, err := scopeconfig.NewClient(
        scopeconfig.WithAddress("localhost:50051"),
        scopeconfig.WithInsecure(), // Use WithTLS() in production
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

    // Get configuration
    config, err := client.GetConfig(ctx, identifier)
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
- `GetConfig(ctx, identifier) (*ScopeConfig, error)` - Get published configuration
- `GetLatestConfig(ctx, identifier) (*ScopeConfig, error)` - Get latest configuration
- `UpdateConfig(ctx, identifier, fields, user) (*ScopeConfig, error)` - Update configuration
- `GetConfigTemplate(ctx, identifier) (*ConfigTemplate, error)` - Get configuration template
- `ApplyConfigTemplate(ctx, template, user) (*ConfigTemplate, error)` - Apply configuration template

### Client Options

- `WithAddress(address string)` - Set server address
- `WithInsecure()` - Use insecure connection (development only)
- `WithTLS(tlsConfig *tls.Config)` - Use TLS connection
- `WithDialOptions(opts ...grpc.DialOption)` - Add custom gRPC dial options

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

## Testing

Run the example test:

```bash
# Start the ScopeConfig service first
# Then run:
go test -v
```

## Extending the SDK

The SDK currently implements the core methods. Additional methods can be added:

- `GetConfigByVersion()` - Retrieve a specific version
- `GetConfigHistory()` - Get version history
- `PublishVersion()` - Publish a specific version
- `DeleteConfig()` - Delete a configuration

See comments in `client.go` for implementation guidance.

## License

See the main project LICENSE file.
