/**
 * In-memory cache for configuration values and templates.
 * - Config values are cached by group to reduce gRPC calls
 * - Templates are cached for default value lookups
 */

import { ScopeConfig, ConfigIdentifier, ConfigTemplate } from "./types";

interface CacheEntry<T> {
  data: T;
  expiresAt: number;
}

/**
 * Configuration cache with TTL support and stale fallback.
 */
export class ConfigCache {
  private configs: Map<string, CacheEntry<ScopeConfig>> = new Map();
  private templates: Map<string, CacheEntry<ConfigTemplate>> = new Map();
  private ttlMs: number;
  private syncInterval: NodeJS.Timeout | null = null;

  constructor(ttlMs: number = 60000) {
    // Default: 1 minute
    this.ttlMs = ttlMs;
  }

  /**
   * Generates a unique cache key for a config identifier.
   */
  private configKey(identifier: ConfigIdentifier): string {
    return `${identifier.serviceName}:${identifier.groupId}:${identifier.scope}:${identifier.projectId || ""}:${identifier.storeId || ""}:${identifier.userId || ""}`;
  }

  /**
   * Generates a unique cache key for a template identifier.
   */
  private templateKey(identifier: ConfigIdentifier): string {
    return `template:${identifier.serviceName}:${identifier.groupId}`;
  }

  /**
   * Gets a config from cache.
   * @returns [config, isValid] - config may be stale if isValid is false
   */
  get(identifier: ConfigIdentifier): [ScopeConfig | null, boolean] {
    const key = this.configKey(identifier);
    const entry = this.configs.get(key);

    if (!entry) {
      return [null, false];
    }

    const isValid = Date.now() < entry.expiresAt;
    return [entry.data, isValid];
  }

  /**
   * Gets a stale config from cache (ignores expiration).
   */
  getStale(identifier: ConfigIdentifier): ScopeConfig | null {
    const key = this.configKey(identifier);
    const entry = this.configs.get(key);
    return entry?.data || null;
  }

  /**
   * Sets a config in the cache.
   */
  set(identifier: ConfigIdentifier, config: ScopeConfig): void {
    const key = this.configKey(identifier);
    this.configs.set(key, {
      data: config,
      expiresAt: Date.now() + this.ttlMs,
    });
  }

  /**
   * Gets a template from cache.
   * @returns [template, isValid] - template may be stale if isValid is false
   */
  getTemplate(identifier: ConfigIdentifier): [ConfigTemplate | null, boolean] {
    const key = this.templateKey(identifier);
    const entry = this.templates.get(key);

    if (!entry) {
      return [null, false];
    }

    const isValid = Date.now() < entry.expiresAt;
    return [entry.data, isValid];
  }

  /**
   * Gets a stale template from cache (ignores expiration).
   */
  getTemplateStale(identifier: ConfigIdentifier): ConfigTemplate | null {
    const key = this.templateKey(identifier);
    const entry = this.templates.get(key);
    return entry?.data || null;
  }

  /**
   * Sets a template in the cache.
   */
  setTemplate(identifier: ConfigIdentifier, template: ConfigTemplate): void {
    const key = this.templateKey(identifier);
    this.templates.set(key, {
      data: template,
      expiresAt: Date.now() + this.ttlMs,
    });
  }

  /**
   * Invalidates a specific config from the cache.
   */
  invalidate(identifier: ConfigIdentifier): void {
    const key = this.configKey(identifier);
    this.configs.delete(key);
  }

  /**
   * Invalidates a specific template from the cache.
   */
  invalidateTemplate(identifier: ConfigIdentifier): void {
    const key = this.templateKey(identifier);
    this.templates.delete(key);
  }

  /**
   * Clears all entries from the cache.
   */
  clear(): void {
    this.configs.clear();
    this.templates.clear();
  }

  /**
   * Gets all cached config identifiers (for background sync).
   */
  getCachedIdentifiers(): ConfigIdentifier[] {
    const identifiers: ConfigIdentifier[] = [];
    for (const key of this.configs.keys()) {
      const parts = key.split(":");
      if (parts.length >= 3) {
        identifiers.push({
          serviceName: parts[0],
          groupId: parts[1],
          scope: parseInt(parts[2], 10),
          projectId: parts[3] || undefined,
          storeId: parts[4] || undefined,
          userId: parts[5] || undefined,
        });
      }
    }
    return identifiers;
  }

  /**
   * Starts background sync interval.
   */
  startBackgroundSync(
    intervalMs: number,
    syncFn: (identifier: ConfigIdentifier) => Promise<void>
  ): void {
    this.stopBackgroundSync();
    this.syncInterval = setInterval(async () => {
      const identifiers = this.getCachedIdentifiers();
      for (const identifier of identifiers) {
        try {
          await syncFn(identifier);
        } catch (error) {
          console.warn(
            `Background sync failed for ${identifier.serviceName}/${identifier.groupId}:`,
            error
          );
        }
      }
    }, intervalMs);
  }

  /**
   * Stops background sync interval.
   */
  stopBackgroundSync(): void {
    if (this.syncInterval) {
      clearInterval(this.syncInterval);
      this.syncInterval = null;
    }
  }
}
