# Java SDK for ScopeConfig Service

A Java client for the ScopeConfig gRPC service with caching support.

## Features

- **In-memory caching** for config values by group (reduces gRPC calls)
- **Template caching** for default value lookups
- **Background sync** to refresh cached configs periodically
- **Stale cache fallback** when server is unavailable
- **GetValue** with inheritance and default value support
- **Environment variable support** for configuration
- **Automatic template loading** from YAML files

## Prerequisites

- Java 22+
- Maven 3.9+
- Buf CLI for protobuf generation

## Installation

```bash
# Copy proto files
mkdir -p proto
cp -r ../../proto/config proto/

# Generate gRPC code
buf generate proto

# Build the SDK
mvn clean install
```

## Quick Start

### Using Environment Variables

```java
import com.dsai.scopeconfig.*;
import vn.dsai.config.v1.*;

// Environment variables:
// GRPC_SCOPE_CONFIG_HOST (default: localhost)
// GRPC_SCOPE_CONFIG_PORT (default: 50051)
// GRPC_SCOPE_CONFIG_USE_TLS (default: false)

try (ConfigClient client = ConfigClient.fromEnvironment().build()) {
    ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
            .scope(Scope.PROJECT)
            .groupId("database")
            .projectId("proj-123")
            .build();

    // Get specific value with inheritance
    Optional<String> value = client.getValue(identifier, "database.host",
            GetValueOptions.withInheritanceAndDefaults());
    
    value.ifPresent(v -> System.out.println("Database host: " + v));
}
```

### With Explicit Configuration

```java
try (ConfigClient client = ConfigClient.builder()
        .host("localhost")
        .port(50051)
        .insecure()
        .cacheEnabled(true)
        .cacheTtl(Duration.ofMinutes(1))
        .backgroundSyncEnabled(true)
        .backgroundSyncInterval(Duration.ofSeconds(30))
        .build()) {

    // Get config with caching
    ScopeConfig config = client.getConfigCached(identifier);
    for (ConfigField field : config.getFieldsList()) {
        System.out.printf("%s = %s%n", field.getPath(), field.getValue());
    }
}
```

## Client Options

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `host` | `GRPC_SCOPE_CONFIG_HOST` | `localhost` | Server host |
| `port` | `GRPC_SCOPE_CONFIG_PORT` | `50051` | Server port |
| `useTls` | `GRPC_SCOPE_CONFIG_USE_TLS` | `false` | Enable TLS |
| `cacheEnabled` | - | `true` | Enable caching |
| `cacheTtl` | - | `1 minute` | Cache TTL |
| `backgroundSyncEnabled` | - | `false` | Enable background sync |
| `backgroundSyncInterval` | - | `30 seconds` | Sync interval |

## API Reference

### Client Methods

- `getConfig(identifier)` - Get config (always fetches from server)
- `getConfigCached(identifier)` - Get config with caching support
- `getLatestConfig(identifier)` - Get latest config (unpublished)
- `getConfigByVersion(identifier, version)` - Get config by version
- `getConfigHistory(identifier, limit)` - Get version history
- `updateConfig(identifier, fields, user)` - Update configuration
- `publishVersion(identifier, version, user)` - Publish a version
- `deleteConfig(identifier)` - Delete configuration
- `getConfigTemplate(identifier)` - Get template (always fetches from server)
- `getConfigTemplateCached(identifier)` - Get template with caching
- `applyConfigTemplate(template, user)` - Apply configuration template
- `getValue(identifier, path, options)` - Get specific value with options
- `getValueString(identifier, path, options)` - Get value as string (empty if not found)
- `invalidateCache(identifier)` - Invalidate cache for specific config
- `clearCache()` - Clear all cached configs
- `isCacheEnabled()` - Check if caching is enabled

### GetValueOptions

```java
// Default options
GetValueOptions options = GetValueOptions.defaults();

// With inheritance and defaults
GetValueOptions options = GetValueOptions.withInheritanceAndDefaults();

// Custom options
GetValueOptions options = GetValueOptions.builder()
    .useDefault(true)
    .inherit(true)
    .build();
```

### Identifier Builder

```java
ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
    .scope(Scope.PROJECT)
    .groupId("database")
    .projectId("proj-123")
    .storeId("store-456")
    .userId("user-789")
    .build();
```

### Scope Hierarchy

```
SYSTEM
├── PROJECT → STORE
└── USER
```

Inheritance:
- **STORE** → PROJECT → SYSTEM
- **USER** → SYSTEM
- **PROJECT** → SYSTEM

## Automatic Template Loading

The SDK supports automatic loading of configuration templates from YAML files. Simply place your template files in a `templates` directory and the SDK will load them automatically.

### Quick Start

1. Create a `templates` directory in your project root
2. Add your YAML template files (`.yaml` or `.yml`)
3. Load templates on client initialization

```java
import com.dsai.scopeconfig.ConfigClient;
import com.dsai.scopeconfig.TemplateLoader;

try (ConfigClient client = ConfigClient.fromEnvironment().build()) {
    // Auto-load all templates from the templates directory
    TemplateLoader.loadFromDir(client, "./templates", "system");
    
    // Now use the client as normal
    Optional<String> value = client.getValue(identifier, "database.host",
            GetValueOptions.withInheritanceAndDefaults());
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

## Using TLS

```java
try (ConfigClient client = ConfigClient.builder()
        .host("config-service.example.com")
        .port(443)
        .tls(new File("/path/to/ca-cert.pem"))
        .build()) {
    // ...
}
```

## Error Handling

```java
try {
    ScopeConfig config = client.getConfig(identifier);
} catch (ConfigNotFoundException e) {
    System.err.println("Configuration not found: " + e.getMessage());
} catch (InvalidConfigException e) {
    System.err.println("Invalid configuration: " + e.getMessage());
} catch (ConfigServiceException e) {
    System.err.printf("Service error (%s): %s%n",
        e.getStatusCode(), e.getMessage());
}
```

## Examples

See the `src/main/java/com/dsai/scopeconfig/examples/` directory for complete working examples:

- `BasicUsage.java` - Comprehensive example demonstrating all SDK features

Run the example:

```bash
# Build the SDK first
mvn clean package

# Run the example
mvn exec:java -Dexec.mainClass="com.dsai.scopeconfig.examples.BasicUsage"
```

### Spring Boot Integration

For Spring Boot applications, you can create a configuration class:

```java
@Configuration
public class ScopeConfigConfiguration {

    @Bean
    public ConfigClient scopeConfigClient() {
        return ConfigClient.fromEnvironment()
                .cacheEnabled(true)
                .cacheTtl(Duration.ofMinutes(1))
                .backgroundSyncEnabled(true)
                .backgroundSyncInterval(Duration.ofSeconds(30))
                .build();
    }
}

@Service
public class MyService {
    private final ConfigClient configClient;

    public MyService(ConfigClient configClient) {
        this.configClient = configClient;
    }

    public String getDatabaseHost(String projectId) {
        ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
                .scope(Scope.PROJECT)
                .groupId("database")
                .projectId(projectId)
                .build();

        return configClient.getValue(identifier, "database.host",
                GetValueOptions.withInheritanceAndDefaults())
                .orElse("localhost");
    }
}
```

## Proto Generation

Generate the proto files using buf:

```bash
# Install buf (https://buf.build/docs/installation)

# Copy proto files
mkdir -p proto
cp -r ../../proto/config proto/

# Generate Java code
buf generate proto
```

## License

See the main project LICENSE file.
