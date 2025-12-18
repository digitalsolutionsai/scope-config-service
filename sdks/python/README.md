# Python SDK for ScopeConfig Service

A Python client for the ScopeConfig gRPC service with caching support.

## Features

- **In-memory caching** for config values by group (reduces gRPC calls)
- **Template caching** for default value lookups
- **Background sync** to refresh cached configs periodically
- **Stale cache fallback** when server is unavailable
- **GetValue** with inheritance and default value support
- **Environment variable support** for configuration
- **Automatic template loading** from YAML files

## Installation

```bash
pip install -r requirements.txt
```

## Quick Start

### Using Environment Variables

```python
from scopeconfig import ConfigClient, Scope, GetValueOptions, create_identifier

# Environment variables (optional):
# GRPC_SCOPE_CONFIG_HOST (default: localhost)
# GRPC_SCOPE_CONFIG_PORT (default: 50051)
# GRPC_SCOPE_CONFIG_USE_TLS (default: false)

# Create client (uses environment variables)
with ConfigClient() as client:
    # Build an identifier
    identifier = (
        create_identifier("my-service")
        .with_scope(Scope.PROJECT)
        .with_group_id("database")
        .with_project_id("proj-123")
        .build()
    )
    
    # Get a specific config value
    value = client.get_value(
        identifier,
        "database.host",
        GetValueOptions(use_default=True, inherit=True)
    )
    print(f"Database host: {value}")
```

### With Explicit Configuration

```python
from scopeconfig import ConfigClient

client = ConfigClient(
    host="localhost",
    port=50051,
    use_tls=False,
    cache_enabled=True,
    cache_ttl_seconds=60.0,
    background_sync_enabled=True,
    background_sync_interval_seconds=30.0,
)

client.connect()
try:
    config = client.get_config_cached(identifier)
    for field in config.fields:
        print(f"{field.path} = {field.value}")
finally:
    client.close()
```

## Client Options

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `host` | `GRPC_SCOPE_CONFIG_HOST` | `localhost` | Server host |
| `port` | `GRPC_SCOPE_CONFIG_PORT` | `50051` | Server port |
| `use_tls` | `GRPC_SCOPE_CONFIG_USE_TLS` | `false` | Enable TLS |
| `cache_enabled` | - | `True` | Enable caching |
| `cache_ttl_seconds` | - | `60.0` | Cache TTL |
| `background_sync_enabled` | - | `False` | Enable background sync |
| `background_sync_interval_seconds` | - | `30.0` | Sync interval |

## API Reference

### Client Methods

- `connect()` - Connect to the gRPC server
- `close()` - Close connection and stop background sync
- `get_config(identifier)` - Get config (always fetches from server)
- `get_config_cached(identifier)` - Get config with caching support
- `get_latest_config(identifier)` - Get latest config (unpublished)
- `get_config_template(identifier)` - Get template (always fetches from server)
- `get_config_template_cached(identifier)` - Get template with caching
- `get_value(identifier, path, options?)` - Get specific value with options
- `get_value_string(identifier, path, options?)` - Get value as string (empty if not found)
- `invalidate_cache(identifier)` - Invalidate cache for specific config
- `clear_cache()` - Clear all cached configs
- `is_cache_enabled()` - Check if caching is enabled

### GetValueOptions

```python
from scopeconfig import GetValueOptions

options = GetValueOptions(
    use_default=True,  # Use template default if config value not set
    inherit=True,      # Traverse parent scopes (STORE → PROJECT → SYSTEM, USER → SYSTEM)
)
```

### Identifier Builder

```python
from scopeconfig import create_identifier, Scope

identifier = (
    create_identifier("my-service")
    .with_scope(Scope.PROJECT)
    .with_group_id("database")
    .with_project_id("proj-123")
    .with_store_id("store-456")
    .with_user_id("user-789")
    .build()
)
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

```python
from scopeconfig import ConfigClient, load_templates_from_dir

# Initialize client and auto-load templates
with ConfigClient() as client:
    # Auto-load all templates from the templates directory
    load_templates_from_dir(client, "./templates", "system")
    
    # Now use the client as normal
    value = client.get_value(identifier, "database.host")
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

## Examples

See the `examples/` directory for complete working examples:

- `examples/basic_usage.py` - Basic SDK usage demonstration
- `examples/fastapi_integration.py` - FastAPI integration with dependency injection

Run the examples:

```bash
# Install dependencies
pip install -r requirements.txt

# Run basic example
python examples/basic_usage.py

# Run FastAPI integration example (requires FastAPI and uvicorn)
pip install fastapi uvicorn
uvicorn examples.fastapi_integration:app --reload
```

### FastAPI Integration

For FastAPI applications, see `examples/fastapi_integration.py` for:

1. Creating a ScopeConfig service with lifespan management
2. Using dependency injection in routes
3. Creating business-logic services
4. Cache management endpoints

## Proto Generation

Generate the proto files using buf:

```bash
# Install buf (https://buf.build/docs/installation)

# Copy proto files
mkdir -p proto
cp -r ../../proto/config proto/

# Generate Python code
buf generate proto
```

## License

See the main project LICENSE file.
