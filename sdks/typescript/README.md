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
