/**
 * gRPC client for the ScopeConfig service with caching support.
 *
 * Features:
 * - In-memory caching for config values by group (reduces gRPC calls)
 * - In-memory caching for templates (for default value lookups)
 * - Background sync to refresh cached configs periodically
 * - Stale cache fallback when server is unavailable
 * - GetValue extracts specific field from cached group config
 * - GetValue with inheritance and default value support
 */

import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";
import * as path from "path";
import {
  ClientOptions,
  ConfigIdentifier,
  ScopeConfig,
  ConfigServiceError,
  ConfigTemplate,
  GetValueOptions,
  Scope,
} from "./types";
import { ConfigCache } from "./cache";

/** Default cache TTL: 1 minute */
const DEFAULT_CACHE_TTL_MS = 60000;
/** Default sync interval: 30 seconds */
const DEFAULT_SYNC_INTERVAL_MS = 30000;

/**
 * ScopeConfig gRPC Client with caching support.
 *
 * @example
 * ```typescript
 * // Create a client with caching
 * const client = new ConfigClient({
 *   address: 'localhost:50051',
 *   insecure: true,
 *   cacheEnabled: true,
 *   cacheTtlMs: 60000, // 1 minute
 *   backgroundSyncEnabled: true,
 *   backgroundSyncIntervalMs: 30000, // 30 seconds
 * });
 *
 * await client.connect();
 *
 * // Get a specific config value (uses cached group config)
 * const value = await client.getValue(identifier, 'database.host', {
 *   useDefault: true,
 *   inherit: true,
 * });
 *
 * // Get full config (cached)
 * const config = await client.getConfigCached(identifier);
 *
 * // Close connection
 * await client.close();
 * ```
 */
export class ConfigClient {
  private client: any;
  private options: ClientOptions;
  private cache: ConfigCache | null = null;

  constructor(options: ClientOptions) {
    this.options = options;

    // Initialize cache if enabled
    if (options.cacheEnabled) {
      this.cache = new ConfigCache(options.cacheTtlMs || DEFAULT_CACHE_TTL_MS);
    }
  }

  /**
   * Connects to the gRPC server and starts background sync if enabled.
   */
  async connect(): Promise<void> {
    try {
      const PROTO_PATH = path.join(
        __dirname,
        "../../proto/config/v1/config.proto"
      );
      const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
        keepCase: true,
        longs: String,
        enums: String,
        defaults: true,
        oneofs: true,
      });

      const protoDescriptor = grpc.loadPackageDefinition(
        packageDefinition
      ) as any;
      const configService = protoDescriptor.vn.dsai.config.v1;

      const credentials = this.options.insecure
        ? grpc.credentials.createInsecure()
        : this.options.credentials || grpc.credentials.createSsl();

      this.client = new configService.ConfigService(
        this.options.address,
        credentials,
        this.options.channelOptions
      );

      // Start background sync if enabled
      if (
        this.options.backgroundSyncEnabled &&
        this.cache &&
        this.options.cacheEnabled
      ) {
        this.cache.startBackgroundSync(
          this.options.backgroundSyncIntervalMs || DEFAULT_SYNC_INTERVAL_MS,
          async (identifier) => {
            try {
              const config = await this.getConfig(identifier);
              this.cache?.set(identifier, config);
            } catch (error) {
              // Silently fail - stale cache will be used
            }
          }
        );
      }
    } catch (error) {
      throw new Error(`Failed to connect to ConfigService: ${error}`);
    }
  }

  /**
   * Closes the client connection and stops background sync.
   */
  async close(): Promise<void> {
    if (this.cache) {
      this.cache.stopBackgroundSync();
    }
    if (this.client) {
      this.client.close();
    }
  }

  /**
   * Gets a config (always fetches from server, updates cache).
   */
  async getConfig(identifier: ConfigIdentifier): Promise<ScopeConfig> {
    const config = await this.promisify("GetConfig", { identifier });

    // Update cache if enabled
    if (this.cache) {
      this.cache.set(identifier, config);
    }

    return config;
  }

  /**
   * Gets a config with caching support.
   * Returns cached value if valid, falls back to stale cache on error.
   */
  async getConfigCached(identifier: ConfigIdentifier): Promise<ScopeConfig> {
    // Try cache first
    if (this.cache) {
      const [cached, isValid] = this.cache.get(identifier);
      if (cached && isValid) {
        return cached;
      }
    }

    // Fetch from server
    try {
      const config = await this.promisify("GetConfig", { identifier });
      if (this.cache) {
        this.cache.set(identifier, config);
      }
      return config;
    } catch (error) {
      // On error, try stale cache
      if (this.cache) {
        const stale = this.cache.getStale(identifier);
        if (stale) {
          console.warn(
            `Using stale cache for ${identifier.serviceName}/${identifier.groupId}`
          );
          return stale;
        }
      }
      throw error;
    }
  }

  /**
   * Gets the latest config (always fetches from server).
   */
  async getLatestConfig(identifier: ConfigIdentifier): Promise<ScopeConfig> {
    return this.promisify("GetLatestConfig", { identifier });
  }

  /**
   * Gets a config template (always fetches from server, updates cache).
   */
  async getConfigTemplate(
    identifier: ConfigIdentifier
  ): Promise<ConfigTemplate> {
    const template = await this.promisify("GetConfigTemplate", { identifier });

    // Update cache if enabled
    if (this.cache) {
      this.cache.setTemplate(identifier, template);
    }

    return template;
  }

  /**
   * Gets a config template with caching support.
   * Templates are cached for default value lookups.
   */
  async getConfigTemplateCached(
    identifier: ConfigIdentifier
  ): Promise<ConfigTemplate> {
    // Try cache first
    if (this.cache) {
      const [cached, isValid] = this.cache.getTemplate(identifier);
      if (cached && isValid) {
        return cached;
      }
    }

    // Fetch from server
    try {
      const template = await this.promisify("GetConfigTemplate", {
        identifier,
      });
      if (this.cache) {
        this.cache.setTemplate(identifier, template);
      }
      return template;
    } catch (error) {
      // On error, try stale cache
      if (this.cache) {
        const stale = this.cache.getTemplateStale(identifier);
        if (stale) {
          return stale;
        }
      }
      throw error;
    }
  }

  /**
   * Applies a config template.
   */
  async applyConfigTemplate(
    template: ConfigTemplate,
    user: string
  ): Promise<ConfigTemplate> {
    return this.promisify("ApplyConfigTemplate", { template, user });
  }

  /**
   * Gets a specific configuration value by path.
   * Optimized to reduce gRPC calls:
   * - Fetches config by group and extracts specific field locally
   * - Uses cached templates for default value lookups
   *
   * @param identifier - Config identifier
   * @param path - Field path (e.g., "database.host")
   * @param options - GetValue options (useDefault, inherit)
   * @returns The value as a string, or null if not found
   */
  async getValue(
    identifier: ConfigIdentifier,
    path: string,
    options?: GetValueOptions
  ): Promise<string | null> {
    const opts = options || {};

    // Try to get value from current scope (uses cached group config)
    const value = await this.getValueFromScope(identifier, path);
    if (value !== null) {
      return value;
    }

    // If inherit is enabled, try parent scopes
    if (opts.inherit) {
      const parentIdentifiers = this.getParentIdentifiers(identifier);
      for (const parentId of parentIdentifiers) {
        const parentValue = await this.getValueFromScope(parentId, path);
        if (parentValue !== null) {
          return parentValue;
        }
      }
    }

    // If useDefault is enabled, try to get default from template
    if (opts.useDefault) {
      const defaultValue = await this.getDefaultValue(identifier, path);
      if (defaultValue !== null) {
        return defaultValue;
      }
    }

    return null;
  }

  /**
   * Gets a value from a specific scope's configuration.
   */
  private async getValueFromScope(
    identifier: ConfigIdentifier,
    path: string
  ): Promise<string | null> {
    try {
      const config = await this.getConfigCached(identifier);
      const field = config.fields.find((f) => f.path === path);
      return field?.value || null;
    } catch {
      return null;
    }
  }

  /**
   * Gets the default value from the configuration template.
   */
  private async getDefaultValue(
    identifier: ConfigIdentifier,
    path: string
  ): Promise<string | null> {
    try {
      const template = await this.getConfigTemplateCached(identifier);
      const field = template.fields.find(
        (f) => f.path === path && f.defaultValue
      );
      return field?.defaultValue || null;
    } catch {
      return null;
    }
  }

  /**
   * Gets parent scope identifiers for inheritance.
   * The inheritance hierarchy is:
   *   SYSTEM
   *   ├── PROJECT → STORE
   *   └── USER
   * So: STORE → PROJECT → SYSTEM, USER → SYSTEM, PROJECT → SYSTEM
   */
  private getParentIdentifiers(
    identifier: ConfigIdentifier
  ): ConfigIdentifier[] {
    const parents: ConfigIdentifier[] = [];

    switch (identifier.scope) {
      case Scope.USER:
        // User -> System (USER is at same level as PROJECT, not under STORE)
        parents.push({
          ...identifier,
          scope: Scope.SYSTEM,
          projectId: undefined,
          storeId: undefined,
          userId: undefined,
        });
        break;

      case Scope.STORE:
        // Store -> Project -> System
        if (identifier.projectId) {
          parents.push({
            ...identifier,
            scope: Scope.PROJECT,
            storeId: undefined,
          });
        }
        parents.push({
          ...identifier,
          scope: Scope.SYSTEM,
          projectId: undefined,
          storeId: undefined,
        });
        break;

      case Scope.PROJECT:
        // Project -> System
        parents.push({
          ...identifier,
          scope: Scope.SYSTEM,
          projectId: undefined,
        });
        break;

      case Scope.SYSTEM:
        // System has no parent
        break;
    }

    return parents;
  }

  /**
   * Invalidates the cache for a specific identifier.
   */
  invalidateCache(identifier: ConfigIdentifier): void {
    this.cache?.invalidate(identifier);
  }

  /**
   * Clears all cached configurations.
   */
  clearCache(): void {
    this.cache?.clear();
  }

  /**
   * Returns whether caching is enabled.
   */
  isCacheEnabled(): boolean {
    return this.cache !== null;
  }

  /**
   * Promisifies a gRPC call and handles errors
   */
  private promisify(method: string, request: any): Promise<any> {
    return new Promise((resolve, reject) => {
      this.client[method](
        request,
        (error: grpc.ServiceError | null, response: any) => {
          if (error) {
            reject(this.wrapError(method, error));
          } else {
            resolve(response);
          }
        }
      );
    });
  }

  /**
   * Wraps gRPC errors with additional context
   */
  private wrapError(
    method: string,
    error: grpc.ServiceError
  ): ConfigServiceError {
    const statusCode = error.code;
    const message = error.details || error.message;

    switch (statusCode) {
      case grpc.status.NOT_FOUND:
        return new ConfigServiceError(
          `${method}: resource not found: ${message}`,
          statusCode,
          message
        );
      case grpc.status.INVALID_ARGUMENT:
        return new ConfigServiceError(
          `${method}: invalid argument: ${message}`,
          statusCode,
          message
        );
      case grpc.status.UNAVAILABLE:
        return new ConfigServiceError(
          `${method}: service unavailable: ${message}`,
          statusCode,
          message
        );
      default:
        return new ConfigServiceError(
          `${method} failed: ${message}`,
          statusCode,
          message
        );
    }
  }
}
