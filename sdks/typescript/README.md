# TypeScript SDK for ScopeConfig Service

A simple TypeScript client for the ScopeConfig gRPC service.

## Quick Start

There are two ways to use the config service in TypeScript:

### Option 1: Using Dynamic Proto Loading 

```bash
# Install dependencies
npm install @grpc/grpc-js @grpc/proto-loader
```

```typescript
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import * as path from 'path';

// Load the proto file dynamically
const PROTO_PATH = path.join(__dirname, '../proto/config/v1/config.proto');
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const proto = grpc.loadPackageDefinition(packageDefinition) as any;
const ConfigService = proto.vn.dsai.config.v1.ConfigService;

// Create client
const client = new ConfigService(
  'localhost:50051',
  grpc.credentials.createInsecure()
);
```

### Option 2: Generate TypeScript Code with Buf (Recommended)

```bash
# Install buf CLI (https://buf.build/docs/installation)
# or use npx @bufbuild/buf

# Install dependencies
npm install @grpc/grpc-js @grpc/proto-loader

# Generate TypeScript code from proto files
buf generate ./your_proto_path

# This generates files in src/gen/ directory based on buf.gen.yaml
```

Then use the generated code:

```typescript
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';

// After buf generate, you can load from the source proto files
const PROTO_PATH = path.join(__dirname, '../proto/config/v1/config.proto');
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const proto = grpc.loadPackageDefinition(packageDefinition) as any;
const client = new proto.vn.dsai.config.v1.ConfigService(
  'localhost:50051',
  grpc.credentials.createInsecure()
);
```

## How to Use the Config Service

### Get Configuration

```typescript
// Get published configuration
client.GetConfig({
  identifier: {
    service_name: 'my-service',
    scope: 1, // SYSTEM
    group_id: 'database',
  },
}, (error, response) => {
  if (error) {
    console.error('Error:', error);
    return;
  }
  console.log('Config:', response);
  response.fields.forEach(field => {
    console.log(`${field.path} = ${field.value}`);
  });
});
```

### Update Configuration

```typescript
// Update or create configuration
client.UpdateConfig({
  identifier: {
    service_name: 'my-service',
    scope: 1, // SYSTEM
    group_id: 'database',
  },
  fields: [
    { path: 'database.host', value: 'localhost', type: 1 }, // STRING
    { path: 'database.port', value: '5432', type: 2 }, // INT
  ],
  user: 'admin@example.com',
}, (error, response) => {
  if (error) {
    console.error('Error:', error);
    return;
  }
  console.log('Updated to version:', response.current_version);
});
```

### Apply Configuration Template

```typescript
// Define a configuration template
client.ApplyConfigTemplate({
  template: {
    identifier: {
      service_name: 'my-service',
      group_id: 'logging',
    },
    service_label: 'My Service',
    group_label: 'Logging',
    group_description: 'Logging configuration',
    fields: [
      {
        path: 'log.level',
        label: 'Log Level',
        description: 'Application log level',
        type: 1, // STRING
        default_value: 'INFO',
        display_on: [1], // SYSTEM
        options: [
          { value: 'DEBUG', label: 'Debug' },
          { value: 'INFO', label: 'Info' },
          { value: 'ERROR', label: 'Error' },
        ],
      },
    ],
  },
  user: 'admin@example.com',
}, (error, response) => {
  if (error) {
    console.error('Error:', error);
    return;
  }
  console.log('Applied template:', response.group_label);
});
```

## Reference

### Scope Values
- `0` - SCOPE_UNSPECIFIED
- `1` - SYSTEM
- `2` - PROJECT
- `3` - STORE
- `4` - USER

### Field Types
- `1` - STRING
- `2` - INT
- `3` - FLOAT
- `4` - BOOLEAN
- `5` - JSON
- `6` - ARRAY_STRING

### Available Methods
- `GetConfig` - Get published configuration
- `GetLatestConfig` - Get latest configuration (published or not)
- `UpdateConfig` - Update or create configuration
- `GetConfigTemplate` - Get configuration template
- `ApplyConfigTemplate` - Apply configuration template
- `GetConfigByVersion` - Get configuration by version
- `GetConfigHistory` - Get version history
- `PublishVersion` - Publish a specific version
- `DeleteConfig` - Delete a configuration
