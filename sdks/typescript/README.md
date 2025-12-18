# TypeScript SDK for ScopeConfig Service

A TypeScript client for the ScopeConfig gRPC service with caching support.

## Features

- **In-memory caching** for config values by group (reduces gRPC calls)
- **Template caching** for default value lookups
- **Background sync** to refresh cached configs periodically
- **Stale cache fallback** when server is unavailable
- **GetValue** with inheritance and default value support
- **TypeScript interfaces** for type safety
- **Environment variable support** for configuration
- **Automatic template loading** from YAML files

## Quick Start

### Installation

```bash
npm install @grpc/grpc-js @grpc/proto-loader
```

### Using Environment Variables

```typescript
import { ConfigClient, createOptionsFromEnv, Scope, createIdentifier } from './src';

// Environment variables:
// GRPC_SCOPE_CONFIG_HOST (default: localhost)
// GRPC_SCOPE_CONFIG_PORT (default: 50051)
// GRPC_SCOPE_CONFIG_USE_TLS (default: false)

const client = new ConfigClient(createOptionsFromEnv());
await client.connect();

const identifier = createIdentifier('my-service')
  .withScope(Scope.PROJECT)
  .withGroupId('database')
  .withProjectId('proj-123')
  .build();

const value = await client.getValue(identifier, 'database.host', {
  useDefault: true,
  inherit: true,
});
console.log('Database host:', value);

await client.close();
```

### With Explicit Configuration

```typescript
import { ConfigClient, Scope, createIdentifier } from './src';

// Create a client with caching
const client = new ConfigClient({
  address: 'localhost:50051',
  insecure: true,
  cacheEnabled: true,
  cacheTtlMs: 60000, // 1 minute
  backgroundSyncEnabled: true,
  backgroundSyncIntervalMs: 30000, // 30 seconds
});

await client.connect();

// Build an identifier
const identifier = createIdentifier('my-service')
  .withScope(Scope.PROJECT)
  .withGroupId('database')
  .withProjectId('proj-123')
  .build();

// Get a specific config value
const host = await client.getValue(identifier, 'database.host', {
  useDefault: true,
  inherit: true,
});
console.log('Database host:', host);

// Get full config (cached)
const config = await client.getConfigCached(identifier);
for (const field of config.fields) {
  console.log(`${field.path} = ${field.value}`);
}

// Close connection
await client.close();
```

## Client Options

```typescript
interface ClientOptions {
  address: string;
  insecure?: boolean;
  credentials?: grpc.ChannelCredentials;
  channelOptions?: grpc.ChannelOptions;
  cacheEnabled?: boolean;              // Enable caching (default: false)
  cacheTtlMs?: number;                 // Cache TTL in ms (default: 60000)
  backgroundSyncEnabled?: boolean;     // Enable background sync (default: false)
  backgroundSyncIntervalMs?: number;   // Sync interval in ms (default: 30000)
}
```

## API Reference

### Client Methods

- `connect()` - Connect to the gRPC server
- `close()` - Close connection and stop background sync
- `getConfig(identifier)` - Get config (always fetches from server)
- `getConfigCached(identifier)` - Get config with caching support
- `getLatestConfig(identifier)` - Get latest config (unpublished)
- `getConfigTemplate(identifier)` - Get template (always fetches from server)
- `getConfigTemplateCached(identifier)` - Get template with caching
- `applyConfigTemplate(template, user)` - Apply a config template
- `getValue(identifier, path, options?)` - Get specific value with options
- `invalidateCache(identifier)` - Invalidate cache for specific config
- `clearCache()` - Clear all cached configs
- `isCacheEnabled()` - Check if caching is enabled

### GetValue Options

```typescript
interface GetValueOptions {
  useDefault?: boolean;  // Use template default if config value not set
  inherit?: boolean;     // Traverse parent scopes (STORE → PROJECT → SYSTEM, USER → SYSTEM)
}
```

### Identifier Builder

```typescript
const identifier = createIdentifier('my-service')
  .withScope(Scope.PROJECT)
  .withGroupId('database')
  .withProjectId('proj-123')
  .withStoreId('store-456')
  .withUserId('user-789')
  .build();
```

### Scope Values

```typescript
enum Scope {
  SCOPE_UNSPECIFIED = 0,
  SYSTEM = 1,
  PROJECT = 2,
  STORE = 3,
  USER = 4,
}
```

### Field Types

```typescript
enum FieldType {
  FIELD_TYPE_UNSPECIFIED = 0,
  STRING = 1,
  INT = 2,
  FLOAT = 3,
  BOOLEAN = 4,
  JSON = 5,
  ARRAY_STRING = 6,
  SECRET = 7,
}
```

## Caching Behavior

- **Config values** are cached by group to reduce gRPC calls
- **Templates** are cached for default value lookups
- **Stale cache fallback**: If server is unavailable, returns stale cached data
- **Background sync**: Periodically refreshes cached configs in the background

## Examples

### Get Config with Caching

```typescript
// First call fetches from server and caches
const config1 = await client.getConfigCached(identifier);

// Second call returns cached value (no gRPC call)
const config2 = await client.getConfigCached(identifier);
```

### Get Value with Inheritance

```typescript
// Inheritance: STORE → PROJECT → SYSTEM, USER → SYSTEM
const value = await client.getValue(
  { ...identifier, scope: Scope.STORE, storeId: 'store-456' },
  'feature.enabled',
  { inherit: true, useDefault: true }
);
```

### Apply Configuration Template

```typescript
const template = {
  identifier: createIdentifier('my-service').withGroupId('logging').build(),
  serviceLabel: 'My Service',
  groupLabel: 'Logging',
  groupDescription: 'Logging configuration',
  fields: [
    {
      path: 'log.level',
      label: 'Log Level',
      description: 'Application log level',
      type: FieldType.STRING,
      defaultValue: 'INFO',
      displayOn: [Scope.SYSTEM, Scope.PROJECT],
      options: [
        { value: 'DEBUG', label: 'Debug' },
        { value: 'INFO', label: 'Info' },
        { value: 'ERROR', label: 'Error' },
      ],
      sortOrder: 100000,
    },
  ],
  sortOrder: 100000,
};

await client.applyConfigTemplate(template, 'admin@example.com');
```

## Automatic Template Loading

The SDK supports automatic loading of configuration templates from YAML files. Simply place your template files in a `templates` directory and the SDK will load them automatically.

### Quick Start

1. Create a `templates` directory in your project root
2. Add your YAML template files (`.yaml` or `.yml`)
3. Load templates on client initialization

```typescript
import { ConfigClient, createOptionsFromEnv, loadTemplatesFromDir } from './src';
import * as fs from 'fs';
import * as path from 'path';
import * as yaml from 'js-yaml';

const client = new ConfigClient(createOptionsFromEnv());
await client.connect();

// Auto-load all templates from the templates directory
await loadTemplatesFromDir(client, './templates', 'system');
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

- `examples/basic-usage.ts` - Basic SDK usage demonstration
- `examples/nestjs-integration.ts` - NestJS integration with dependency injection

Run the examples:

```bash
# Install dependencies
npm install

# Run basic example
npx ts-node examples/basic-usage.ts

# Run NestJS integration example
npx ts-node examples/nestjs-integration.ts
```

### NestJS Integration

For NestJS applications, see `examples/nestjs-integration.ts` for:

1. Creating a ScopeConfig module with lifespan management
2. Creating a service wrapper with dependency injection
3. Using the service in controllers and other services
4. Configuring tsconfig.json paths for custom imports

## Proto File Generation (Optional)

For production use, generate TypeScript types from proto files:

```bash
# Install buf CLI
npm install -g @bufbuild/buf

# Generate TypeScript code
buf generate
```

## License

See the main project LICENSE file.
