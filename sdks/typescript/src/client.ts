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
 * - Environment variable support for configuration
 *
 * Environment Variables:
 * - GRPC_SCOPE_CONFIG_HOST: Server host (default: localhost)
 * - GRPC_SCOPE_CONFIG_PORT: Server port (default: 50051)
 * - GRPC_SCOPE_CONFIG_USE_TLS: Enable TLS (default: false)
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
  ENV_HOST,
  ENV_PORT,
  ENV_USE_TLS,
  DEFAULT_HOST,
  DEFAULT_PORT,
} from "./types";
import { ConfigCache } from "./cache";

/** Default cache TTL: 1 minute */
const DEFAULT_CACHE_TTL_MS = 60000;
/** Default sync interval: 30 seconds */
const DEFAULT_SYNC_INTERVAL_MS = 30000;

/**
 * Creates client options from environment variables.
 */
export function createOptionsFromEnv(
  overrides?: Partial<ClientOptions>
): ClientOptions {
  const host = process.env[ENV_HOST] || DEFAULT_HOST;
  const port = parseInt(process.env[ENV_PORT] || String(DEFAULT_PORT), 10);
  const useTlsEnv = process.env[ENV_USE_TLS]?.toLowerCase();
  const useTls = useTlsEnv === "true" || useTlsEnv === "1" || useTlsEnv === "yes";

  return {
    address: `${host}:${port}`,
    host,
    port,
    insecure: !useTls,
    cacheEnabled: true,
    ...overrides,
  };
}

/**
 * ScopeConfig gRPC Client with caching support.
 *
 * @example
 * ```typescript
 * // Create a client using environment variables
 * const client = new ConfigClient(createOptionsFromEnv());
 *
 * // Or with explicit configuration
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
  private address: string;

  constructor(options?: ClientOptions) {
    // Use provided options or load from environment
    this.options = options || createOptionsFromEnv();

    // Resolve address
    if (this.options.address) {
      this.address = this.options.address;
    } else {
      const host = this.options.host || process.env[ENV_HOST] || DEFAULT_HOST;
      const port =
        this.options.port ||
        parseInt(process.env[ENV_PORT] || String(DEFAULT_PORT), 10);
      this.address = `${host}:${port}`;
    }

    // Initialize cache if enabled (default: true)
    if (this.options.cacheEnabled !== false) {
      this.cache = new ConfigCache(
        this.options.cacheTtlMs || DEFAULT_CACHE_TTL_MS
      );
    }
  }

  /**
   * Connects to the gRPC server and starts background sync if enabled.
   */
  async connect(): Promise<void> {
    try {
      const PROTO_PATH = path.join(
        __dirname,
        "../proto/config/v1/config.proto"
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
        this.address,
        credentials,
        this.options.channelOptions
      );

      // Start background sync if enabled
      if (
        this.options.backgroundSyncEnabled &&
        this.cache &&
        this.options.cacheEnabled !== false
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

/**
 * Load and apply all YAML templates from a directory.
 *
 * Simply place your template files in the specified directory and this function
 * will automatically load and apply them to the config service.
 *
 * @param client - The connected ConfigClient instance
 * @param dirPath - Path to the templates directory
 * @param user - The user performing the action
 *
 * @example
 * // Initialize client and auto-load templates
 * const client = new ConfigClient(createOptionsFromEnv());
 * await client.connect();
 * await loadTemplatesFromDir(client, './templates', 'system');
 */
export async function loadTemplatesFromDir(
  client: ConfigClient,
  dirPath: string,
  user: string
): Promise<void> {
  const fs = await import("fs");
  const path = await import("path");

  // Check if yaml is available
  let yaml: any;
  try {
    yaml = await import("js-yaml");
  } catch {
    throw new ConfigServiceError(
      "js-yaml is required for template loading. Install with: npm install js-yaml",
      grpc.status.INTERNAL
    );
  }

  if (!fs.existsSync(dirPath)) {
    console.log(
      `Templates directory ${dirPath} does not exist, skipping template import`
    );
    return;
  }

  const files = fs.readdirSync(dirPath);
  const yamlFiles = files.filter(
    (f: string) => f.endsWith(".yaml") || f.endsWith(".yml")
  );

  if (yamlFiles.length === 0) {
    console.log(`No template files found in ${dirPath}`);
    return;
  }

  console.log(`Found ${yamlFiles.length} template file(s) to import`);

  for (const file of yamlFiles) {
    const filePath = path.join(dirPath, file);
    await loadAndApplyTemplateFile(client, filePath, user, yaml);
  }
}

async function loadAndApplyTemplateFile(
  client: ConfigClient,
  filePath: string,
  user: string,
  yaml: any
): Promise<void> {
  const fs = await import("fs");
  const path = await import("path");

  let data: any;
  try {
    const content = fs.readFileSync(filePath, "utf-8");
    data = yaml.load(content);
  } catch (e: any) {
    throw new ConfigServiceError(
      `Failed to read template file ${filePath}: ${e.message}`,
      grpc.status.INTERNAL
    );
  }

  if (!data) {
    console.warn(`Empty template file: ${filePath}`);
    return;
  }

  // Validate required fields
  if (!data.service || !data.service.id) {
    throw new ConfigServiceError(
      `Template file ${filePath} missing 'service.id'`,
      grpc.status.INVALID_ARGUMENT
    );
  }

  const serviceName = data.service.id;
  const serviceLabel = data.service.label || serviceName;
  const groups = data.groups || [];

  if (groups.length === 0) {
    console.warn(`No groups defined in template: ${filePath}`);
    return;
  }

  for (const group of groups) {
    await applyGroupTemplate(client, serviceName, serviceLabel, group, user);
    console.log(
      `Successfully imported template: service=${serviceName}, group=${group.id} from ${path.basename(filePath)}`
    );
  }
}

async function applyGroupTemplate(
  client: ConfigClient,
  serviceName: string,
  serviceLabel: string,
  group: any,
  user: string
): Promise<void> {
  const groupId = group.id || "";
  const groupLabel = group.label || groupId;
  const groupDescription = group.description || "";
  const sortOrder = group.sortOrder || 0;

  const fields = (group.fields || []).map((f: any) => ({
    path: f.path || "",
    label: f.label || "",
    description: f.description || "",
    type: toFieldType(f.type || "STRING"),
    default_value: f.defaultValue || "",  // snake_case for proto-loader
    display_on: (f.displayOn || []).map(toScope),  // snake_case for proto-loader
    options: (f.options || []).map((o: any) => ({
      value: o.value,
      label: o.label || o.value,
    })),
    sort_order: f.sortOrder || 0,  // snake_case for proto-loader
  }));

  // Use snake_case field names for proto-loader (keepCase: true)
  const template: any = {
    identifier: {
      service_name: serviceName,  // snake_case
      group_id: groupId,          // snake_case
      scope: Scope.SCOPE_UNSPECIFIED,
    },
    service_label: serviceLabel,      // snake_case
    group_label: groupLabel,          // snake_case
    group_description: groupDescription,  // snake_case
    fields,
    sort_order: sortOrder,  // snake_case
  };

  await client.applyConfigTemplate(template, user);
}

function toScope(s: string): Scope {
  const scopeMap: Record<string, Scope> = {
    SYSTEM: Scope.SYSTEM,
    PROJECT: Scope.PROJECT,
    STORE: Scope.STORE,
    USER: Scope.USER,
  };
  return scopeMap[s.toUpperCase()] || Scope.SCOPE_UNSPECIFIED;
}

function toFieldType(t: string): number {
  const typeMap: Record<string, number> = {
    STRING: 1,
    INT: 2,
    FLOAT: 3,
    BOOLEAN: 4,
    JSON: 5,
    ARRAY_STRING: 6,
    SECRET: 7,
  };
  return typeMap[t.toUpperCase()] || 1;
}
