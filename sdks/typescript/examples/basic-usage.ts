/**
 * Example usage of the ScopeConfig TypeScript SDK.
 *
 * This example demonstrates:
 * - Creating a client using environment variables
 * - Building config identifiers
 * - Getting config values with caching
 * - Using inheritance and default values
 * - Applying configuration templates
 *
 * Prerequisites:
 * 1. Install dependencies: npm install
 * 2. Generate proto files: npx buf generate (optional)
 * 3. Set environment variables (optional):
 *    - GRPC_SCOPE_CONFIG_HOST (default: localhost)
 *    - GRPC_SCOPE_CONFIG_PORT (default: 50051)
 *    - GRPC_SCOPE_CONFIG_USE_TLS (default: false)
 *
 * Run:
 *   npx ts-node examples/basic-usage.ts
 */

import {
  ConfigClient,
  createOptionsFromEnv,
  createIdentifier,
  Scope,
  FieldType,
  ConfigTemplate,
} from '../src';

async function main() {
  console.log('=== ScopeConfig TypeScript SDK Example ===\n');

  // Example 1: Create client using environment variables
  console.log('=== Example 1: Using Environment Variables ===');
  const envOptions = createOptionsFromEnv();
  console.log(`Host: ${envOptions.host || 'localhost'}`);
  console.log(`Port: ${envOptions.port || 50051}`);
  console.log(`Insecure: ${envOptions.insecure}`);

  // Example 2: Create client with explicit configuration
  console.log('\n=== Example 2: Explicit Configuration ===');
  const client = new ConfigClient({
    address: 'localhost:50051',
    insecure: true,
    cacheEnabled: true,
    cacheTtlMs: 60000, // 1 minute
    backgroundSyncEnabled: true,
    backgroundSyncIntervalMs: 30000, // 30 seconds
  });

  try {
    await client.connect();
    console.log('Client connected successfully');
  } catch (error) {
    console.log(`Failed to connect: ${error}`);
    console.log('\nNote: To run this example with a live server, start the ScopeConfig service first.');
    demonstrateIdentifierBuilding();
    return;
  }

  try {
    // Example 3: Build config identifiers
    console.log('\n=== Example 3: Building Config Identifiers ===');
    demonstrateIdentifierBuilding();

    // Example 4: Get configuration with caching
    console.log('\n=== Example 4: Get Configuration with Caching ===');
    const identifier = createIdentifier('payment-service')
      .withScope(Scope.PROJECT)
      .withGroupId('database')
      .withProjectId('proj-123')
      .build();

    try {
      const config = await client.getConfigCached(identifier);
      console.log(`Configuration for ${config.versionInfo?.identifier?.serviceName}:`);
      for (const field of config.fields || []) {
        console.log(`  ${field.path} = ${field.value}`);
      }
    } catch (error) {
      console.log(`Failed to get config: ${error}`);
    }

    // Example 5: Get specific value with inheritance
    console.log('\n=== Example 5: Get Value with Inheritance ===');
    try {
      const value = await client.getValue(identifier, 'database.host', {
        useDefault: true, // Use template default if not set
        inherit: true, // Traverse parent scopes (STORE → PROJECT → SYSTEM)
      });
      if (value !== null) {
        console.log(`Database host: ${value}`);
      } else {
        console.log('Database host not found');
      }
    } catch (error) {
      console.log(`Failed to get value: ${error}`);
    }

    // Example 6: Apply configuration template
    console.log('\n=== Example 6: Apply Configuration Template ===');
    const template: ConfigTemplate = {
      identifier: createIdentifier('payment-service')
        .withGroupId('logging')
        .build(),
      serviceLabel: 'Payment Service',
      groupLabel: 'Logging Configuration',
      groupDescription: 'Controls logging behavior for the payment service',
      fields: [
        {
          path: 'log.level',
          label: 'Log Level',
          description: 'Application logging level',
          type: FieldType.STRING,
          defaultValue: 'INFO',
          displayOn: [Scope.SYSTEM, Scope.PROJECT],
          options: [
            { value: 'DEBUG', label: 'Debug' },
            { value: 'INFO', label: 'Info' },
            { value: 'WARN', label: 'Warning' },
            { value: 'ERROR', label: 'Error' },
          ],
          sortOrder: 100000,
        },
        {
          path: 'log.format',
          label: 'Log Format',
          description: 'Output format for log messages',
          type: FieldType.STRING,
          defaultValue: 'json',
          displayOn: [Scope.SYSTEM],
          options: [
            { value: 'json', label: 'JSON' },
            { value: 'text', label: 'Plain Text' },
          ],
          sortOrder: 100001,
        },
      ],
      sortOrder: 100000,
    };

    try {
      const result = await client.applyConfigTemplate(template, 'admin@example.com');
      console.log(`Applied template: ${result.serviceLabel} - ${result.groupLabel}`);
    } catch (error) {
      console.log(`Failed to apply template: ${error}`);
    }

    // Example 7: Cache management
    console.log('\n=== Example 7: Cache Management ===');
    console.log(`Cache enabled: ${client.isCacheEnabled()}`);

    // Invalidate specific config cache
    client.invalidateCache(identifier);
    console.log('Cache invalidated for specific identifier');

    // Clear all cache
    client.clearCache();
    console.log('All cache cleared');

  } finally {
    // Clean up
    await client.close();
    console.log('\n=== Example Complete ===');
  }
}

function demonstrateIdentifierBuilding() {
  // SYSTEM scope (global config)
  const systemId = createIdentifier('my-service')
    .withScope(Scope.SYSTEM)
    .withGroupId('database')
    .build();
  console.log(`System identifier: service=${systemId.serviceName}, group=${systemId.groupId}, scope=${systemId.scope}`);

  // PROJECT scope
  const projectId = createIdentifier('my-service')
    .withScope(Scope.PROJECT)
    .withGroupId('database')
    .withProjectId('proj-123')
    .build();
  console.log(`Project identifier: service=${projectId.serviceName}, group=${projectId.groupId}, project=${projectId.projectId}`);

  // STORE scope
  const storeId = createIdentifier('my-service')
    .withScope(Scope.STORE)
    .withGroupId('database')
    .withProjectId('proj-123')
    .withStoreId('store-456')
    .build();
  console.log(`Store identifier: service=${storeId.serviceName}, group=${storeId.groupId}, project=${storeId.projectId}, store=${storeId.storeId}`);

  // USER scope
  const userId = createIdentifier('my-service')
    .withScope(Scope.USER)
    .withGroupId('preferences')
    .withUserId('user-789')
    .build();
  console.log(`User identifier: service=${userId.serviceName}, group=${userId.groupId}, user=${userId.userId}`);
}

// Run the example
main().catch(console.error);
