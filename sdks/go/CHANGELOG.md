# Changelog

All notable changes to the Go SDK for ScopeConfig Service will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-01-03

### Added

- **Core Client Features**
  - `NewClient` with configurable options for address, TLS, caching, and background sync
  - `GetConfig` - Get published configuration
  - `GetConfigCached` - Get configuration with caching support and stale fallback
  - `GetLatestConfig` - Get latest configuration version (published or not)
  - `GetConfigByVersion` - Get configuration by specific version number
  - `GetConfigHistory` - Get version history for a configuration
  - `UpdateConfig` - Create or update configuration values
  - `PublishVersion` - Mark a specific version as published
  - `DeleteConfig` - Delete a configuration and all its versions

- **Template Management**
  - `ApplyConfigTemplate` - Apply a configuration template
  - `GetConfigTemplate` - Get a configuration template
  - `GetConfigTemplateCached` - Get template with caching support
  - `ListConfigTemplates` - List all templates for a service
  - `LoadTemplatesFromDir` - Load and apply templates from YAML files
  - `LoadTemplateFromBytes` - Load and apply template from YAML bytes

- **Value Helpers**
  - `GetValue` - Get specific config value with inheritance and default value support
  - `GetValueString` - Convenience method returning empty string instead of nil
  - `MustGetValue` - Get value or panic on error

- **Caching**
  - In-memory caching for config values by group
  - Template caching for default value lookups
  - Background sync to refresh cached configs periodically
  - Stale cache fallback when server is unavailable
  - `InvalidateCache` - Invalidate cache for specific identifier
  - `ClearCache` - Clear all cached configurations

- **Identifier Builder**
  - Fluent API for building `ConfigIdentifier` objects
  - Support for SYSTEM, PROJECT, STORE, and USER scopes

- **Error Handling**
  - Custom error types: `ConfigError`, `ErrConfigNotFound`, `ErrTemplateNotFound`, etc.
  - Error inspection helpers: `IsNotFound`, `IsServerUnavailable`, `IsInvalidArgument`, etc.
  - `GRPCCode` - Extract gRPC status code from errors

- **Configuration Options**
  - `WithAddress` - Set server address
  - `WithInsecure` - Use insecure connection
  - `WithTLS` - Use TLS connection with custom config
  - `WithDialOptions` - Add custom gRPC dial options
  - `WithCache` - Enable caching with specified TTL
  - `WithBackgroundSync` - Enable background synchronization
  - `FromEnvironment` - Create client options from environment variables

- **Testing**
  - Comprehensive unit tests for all components
  - Integration tests using testcontainers-go
  - Test fixtures for template loading tests
  - Makefile with test targets

- **Documentation**
  - Complete README with installation and usage examples
  - GoDoc comments for all exported functions
  - Example programs in `examples/` directory

### Environment Variables

- `GRPC_SCOPE_CONFIG_HOST` - Server host (default: localhost)
- `GRPC_SCOPE_CONFIG_PORT` - Server port (default: 50051)
- `GRPC_SCOPE_CONFIG_USE_TLS` - Enable TLS (default: false)

## [Unreleased]

### Planned

- Retry logic with exponential backoff
- Prometheus metrics integration
- OpenTelemetry tracing support
