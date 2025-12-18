/**
 * NestJS Integration Example for ScopeConfig TypeScript SDK.
 *
 * This example demonstrates how to integrate the ScopeConfig SDK
 * with a NestJS application using the Module/Service pattern.
 *
 * Integration steps:
 * 1. Copy the SDK to your project's libs folder
 * 2. Configure tsconfig.json paths
 * 3. Create a NestJS module and service
 * 4. Inject and use the service
 *
 * Prerequisites:
 * - NestJS application
 * - Environment variables (optional):
 *   - GRPC_SCOPE_CONFIG_HOST (default: localhost)
 *   - GRPC_SCOPE_CONFIG_PORT (default: 50051)
 *   - GRPC_SCOPE_CONFIG_USE_TLS (default: false)
 */

// =============================================================================
// Step 1: Copy SDK to libs folder
// =============================================================================
// Copy the sdks/typescript folder to your project:
//   cp -r sdks/typescript your-nestjs-app/libs/scopeconfig

// =============================================================================
// Step 2: Configure tsconfig.json
// =============================================================================
// Add path alias in tsconfig.json:
/*
{
  "compilerOptions": {
    "paths": {
      "@scopeconfig": ["libs/scopeconfig/src"],
      "@scopeconfig/*": ["libs/scopeconfig/src/*"]
    }
  }
}
*/

// =============================================================================
// Step 3: Create ScopeConfig Module (libs/scopeconfig/scopeconfig.module.ts)
// =============================================================================
/*
import { Module, Global, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
import { ConfigClient, createOptionsFromEnv, ClientOptions } from './src';

export const SCOPE_CONFIG_CLIENT = 'SCOPE_CONFIG_CLIENT';

@Global()
@Module({
  providers: [
    {
      provide: SCOPE_CONFIG_CLIENT,
      useFactory: async (): Promise<ConfigClient> => {
        const options: ClientOptions = createOptionsFromEnv({
          cacheEnabled: true,
          cacheTtlMs: 60000,
          backgroundSyncEnabled: true,
          backgroundSyncIntervalMs: 30000,
        });
        const client = new ConfigClient(options);
        await client.connect();
        return client;
      },
    },
  ],
  exports: [SCOPE_CONFIG_CLIENT],
})
export class ScopeConfigModule implements OnModuleDestroy {
  constructor(
    @Inject(SCOPE_CONFIG_CLIENT) private readonly client: ConfigClient,
  ) {}

  async onModuleDestroy() {
    await this.client.close();
  }
}
*/

// =============================================================================
// Step 4: Create ScopeConfig Service (libs/scopeconfig/scopeconfig.service.ts)
// =============================================================================
/*
import { Injectable, Inject } from '@nestjs/common';
import {
  ConfigClient,
  ConfigIdentifier,
  ScopeConfig,
  ConfigTemplate,
  GetValueOptions,
  createIdentifier,
  Scope,
} from './src';
import { SCOPE_CONFIG_CLIENT } from './scopeconfig.module';

@Injectable()
export class ScopeConfigService {
  constructor(
    @Inject(SCOPE_CONFIG_CLIENT) private readonly client: ConfigClient,
  ) {}

  // Get config with caching
  async getConfig(identifier: ConfigIdentifier): Promise<ScopeConfig> {
    return this.client.getConfigCached(identifier);
  }

  // Get specific value with options
  async getValue(
    identifier: ConfigIdentifier,
    path: string,
    options?: GetValueOptions,
  ): Promise<string | null> {
    return this.client.getValue(identifier, path, options);
  }

  // Helper to build identifiers
  buildIdentifier(serviceName: string) {
    return createIdentifier(serviceName);
  }

  // Helper to get project config value
  async getProjectValue(
    serviceName: string,
    groupId: string,
    projectId: string,
    path: string,
    useDefault = true,
    inherit = true,
  ): Promise<string | null> {
    const identifier = createIdentifier(serviceName)
      .withScope(Scope.PROJECT)
      .withGroupId(groupId)
      .withProjectId(projectId)
      .build();

    return this.getValue(identifier, path, { useDefault, inherit });
  }

  // Helper to get store config value
  async getStoreValue(
    serviceName: string,
    groupId: string,
    projectId: string,
    storeId: string,
    path: string,
    useDefault = true,
    inherit = true,
  ): Promise<string | null> {
    const identifier = createIdentifier(serviceName)
      .withScope(Scope.STORE)
      .withGroupId(groupId)
      .withProjectId(projectId)
      .withStoreId(storeId)
      .build();

    return this.getValue(identifier, path, { useDefault, inherit });
  }

  // Apply configuration template
  async applyTemplate(
    template: ConfigTemplate,
    user: string,
  ): Promise<ConfigTemplate> {
    return this.client.applyConfigTemplate(template, user);
  }

  // Invalidate cache for specific config
  invalidateCache(identifier: ConfigIdentifier): void {
    this.client.invalidateCache(identifier);
  }

  // Clear all cache
  clearCache(): void {
    this.client.clearCache();
  }
}
*/

// =============================================================================
// Step 5: Use in your application
// =============================================================================
/*
// app.module.ts
import { Module } from '@nestjs/common';
import { ScopeConfigModule, ScopeConfigService } from '@scopeconfig';
import { PaymentService } from './payment/payment.service';

@Module({
  imports: [ScopeConfigModule],
  providers: [ScopeConfigService, PaymentService],
})
export class AppModule {}

// payment/payment.service.ts
import { Injectable } from '@nestjs/common';
import { ScopeConfigService, Scope, createIdentifier } from '@scopeconfig';

@Injectable()
export class PaymentService {
  constructor(private readonly scopeConfig: ScopeConfigService) {}

  async getPaymentGatewayUrl(projectId: string, storeId: string): Promise<string> {
    // Get store-level config with inheritance (falls back to project, then system)
    const url = await this.scopeConfig.getStoreValue(
      'payment-service',
      'gateway',
      projectId,
      storeId,
      'gateway.url',
      true,  // useDefault
      true,  // inherit
    );

    return url || 'https://default-gateway.example.com';
  }

  async isFeatureEnabled(
    projectId: string,
    featureName: string,
  ): Promise<boolean> {
    const value = await this.scopeConfig.getProjectValue(
      'payment-service',
      'features',
      projectId,
      `feature.${featureName}.enabled`,
      true,
      true,
    );

    return value === 'true';
  }

  async getUserPreference(
    userId: string,
    preferencePath: string,
  ): Promise<string | null> {
    const identifier = createIdentifier('payment-service')
      .withScope(Scope.USER)
      .withGroupId('preferences')
      .withUserId(userId)
      .build();

    return this.scopeConfig.getValue(identifier, preferencePath, {
      useDefault: true,
      inherit: true,
    });
  }
}
*/

// =============================================================================
// Example: Direct usage without NestJS module
// =============================================================================

import {
  ConfigClient,
  createOptionsFromEnv,
  createIdentifier,
  Scope,
} from '../src';

async function directUsageExample() {
  console.log('=== Direct Usage Example (without NestJS module) ===\n');

  // Create client from environment variables
  const client = new ConfigClient(createOptionsFromEnv({
    cacheEnabled: true,
    cacheTtlMs: 60000,
    backgroundSyncEnabled: true,
  }));

  try {
    await client.connect();
    console.log('Connected to ScopeConfig service');

    // Get project-level database config
    const dbIdentifier = createIdentifier('my-nestjs-app')
      .withScope(Scope.PROJECT)
      .withGroupId('database')
      .withProjectId('proj-123')
      .build();

    const dbHost = await client.getValue(dbIdentifier, 'database.host', {
      useDefault: true,
      inherit: true,
    });
    console.log(`Database host: ${dbHost}`);

    // Get store-level feature flags
    const featureIdentifier = createIdentifier('my-nestjs-app')
      .withScope(Scope.STORE)
      .withGroupId('features')
      .withProjectId('proj-123')
      .withStoreId('store-456')
      .build();

    const featureEnabled = await client.getValue(featureIdentifier, 'feature.newCheckout.enabled', {
      useDefault: true,
      inherit: true,
    });
    console.log(`New checkout enabled: ${featureEnabled}`);

  } catch (error) {
    console.log(`Error: ${error}`);
    console.log('\nNote: This example requires a running ScopeConfig service.');
  } finally {
    await client.close();
  }
}

// Run the example
directUsageExample().catch(console.error);
