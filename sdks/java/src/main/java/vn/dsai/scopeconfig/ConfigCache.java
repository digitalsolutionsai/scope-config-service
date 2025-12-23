package vn.dsai.scopeconfig;

import java.time.Duration;
import java.time.Instant;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.Map;
import java.util.List;
import java.util.ArrayList;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;
import java.util.logging.Level;
import java.util.logging.Logger;

import vn.dsai.config.v1.ConfigIdentifier;
import vn.dsai.config.v1.ScopeConfig;
import vn.dsai.config.v1.ConfigTemplate;

/**
 * In-memory cache for configuration values and templates.
 * 
 * Features:
 * - Config values are cached by group to reduce gRPC calls
 * - Templates are cached for default value lookups
 * - Thread-safe implementation using ConcurrentHashMap
 * - TTL-based expiration
 * - Background sync support
 */
public class ConfigCache {
    
    private static final Logger logger = Logger.getLogger(ConfigCache.class.getName());
    
    private final ConcurrentMap<String, CacheEntry<ScopeConfig>> configs;
    private final ConcurrentMap<String, CacheEntry<ConfigTemplate>> templates;
    private final Duration ttl;
    
    private ScheduledExecutorService syncExecutor;
    private ScheduledFuture<?> syncFuture;
    
    /**
     * Creates a new cache with the specified TTL.
     * 
     * @param ttl Time-to-live for cache entries
     */
    public ConfigCache(Duration ttl) {
        this.configs = new ConcurrentHashMap<>();
        this.templates = new ConcurrentHashMap<>();
        this.ttl = ttl;
    }
    
    /**
     * Creates a new cache with the default TTL of 1 minute.
     */
    public ConfigCache() {
        this(Duration.ofMinutes(1));
    }
    
    /**
     * Generates a unique cache key for a config identifier.
     */
    private String configKey(ConfigIdentifier identifier) {
        return identifier.getServiceName() + ":" +
               identifier.getGroupId() + ":" +
               identifier.getScopeValue() + ":" +
               identifier.getProjectId() + ":" +
               identifier.getStoreId() + ":" +
               identifier.getUserId();
    }
    
    /**
     * Generates a unique cache key for a template identifier.
     */
    private String templateKey(ConfigIdentifier identifier) {
        return "template:" + identifier.getServiceName() + ":" + identifier.getGroupId();
    }
    
    /**
     * Gets a config from cache.
     * 
     * @param identifier The config identifier
     * @return CacheResult containing the config and validity status
     */
    public CacheResult<ScopeConfig> get(ConfigIdentifier identifier) {
        String key = configKey(identifier);
        CacheEntry<ScopeConfig> entry = configs.get(key);
        
        if (entry == null) {
            return new CacheResult<>(null, false);
        }
        
        return new CacheResult<>(entry.data, entry.isValid());
    }
    
    /**
     * Gets a config from cache even if expired (stale).
     */
    public ScopeConfig getStale(ConfigIdentifier identifier) {
        String key = configKey(identifier);
        CacheEntry<ScopeConfig> entry = configs.get(key);
        return entry != null ? entry.data : null;
    }
    
    /**
     * Stores a config in the cache.
     */
    public void set(ConfigIdentifier identifier, ScopeConfig config) {
        String key = configKey(identifier);
        configs.put(key, new CacheEntry<>(config, Instant.now().plus(ttl)));
    }
    
    /**
     * Gets a template from cache.
     */
    public CacheResult<ConfigTemplate> getTemplate(ConfigIdentifier identifier) {
        String key = templateKey(identifier);
        CacheEntry<ConfigTemplate> entry = templates.get(key);
        
        if (entry == null) {
            return new CacheResult<>(null, false);
        }
        
        return new CacheResult<>(entry.data, entry.isValid());
    }
    
    /**
     * Gets a template from cache even if expired (stale).
     */
    public ConfigTemplate getTemplateStale(ConfigIdentifier identifier) {
        String key = templateKey(identifier);
        CacheEntry<ConfigTemplate> entry = templates.get(key);
        return entry != null ? entry.data : null;
    }
    
    /**
     * Stores a template in the cache.
     */
    public void setTemplate(ConfigIdentifier identifier, ConfigTemplate template) {
        String key = templateKey(identifier);
        templates.put(key, new CacheEntry<>(template, Instant.now().plus(ttl)));
    }
    
    /**
     * Removes a specific config from the cache.
     */
    public void invalidate(ConfigIdentifier identifier) {
        String key = configKey(identifier);
        configs.remove(key);
    }
    
    /**
     * Removes a specific template from the cache.
     */
    public void invalidateTemplate(ConfigIdentifier identifier) {
        String key = templateKey(identifier);
        templates.remove(key);
    }
    
    /**
     * Clears all entries from the cache.
     */
    public void clear() {
        configs.clear();
        templates.clear();
    }
    
    /**
     * Gets all cached config identifiers (for background sync).
     */
    public List<ConfigIdentifier> getCachedIdentifiers() {
        List<ConfigIdentifier> identifiers = new ArrayList<>();
        
        for (String key : configs.keySet()) {
            String[] parts = key.split(":");
            if (parts.length >= 3) {
                try {
                    int scopeValue = Integer.parseInt(parts[2]);
                    ConfigIdentifier.Builder builder = ConfigIdentifier.newBuilder()
                        .setServiceName(parts[0])
                        .setGroupId(parts[1])
                        .setScopeValue(scopeValue);
                    
                    if (parts.length > 3 && !parts[3].isEmpty()) {
                        builder.setProjectId(parts[3]);
                    }
                    if (parts.length > 4 && !parts[4].isEmpty()) {
                        builder.setStoreId(parts[4]);
                    }
                    if (parts.length > 5 && !parts[5].isEmpty()) {
                        builder.setUserId(parts[5]);
                    }
                    
                    identifiers.add(builder.build());
                } catch (NumberFormatException e) {
                    // Skip invalid entries
                }
            }
        }
        
        return identifiers;
    }
    
    /**
     * Starts background sync at the specified interval.
     */
    public void startBackgroundSync(Duration interval, Consumer<ConfigIdentifier> syncFn) {
        stopBackgroundSync();
        
        syncExecutor = Executors.newSingleThreadScheduledExecutor(r -> {
            Thread t = new Thread(r, "scopeconfig-cache-sync");
            t.setDaemon(true);
            return t;
        });
        
        syncFuture = syncExecutor.scheduleAtFixedRate(() -> {
            List<ConfigIdentifier> identifiers = getCachedIdentifiers();
            for (ConfigIdentifier identifier : identifiers) {
                try {
                    syncFn.accept(identifier);
                } catch (Exception e) {
                    logger.log(Level.WARNING, 
                        "Background sync failed for " + identifier.getServiceName() + "/" + identifier.getGroupId(), e);
                }
            }
        }, interval.toMillis(), interval.toMillis(), TimeUnit.MILLISECONDS);
    }
    
    /**
     * Stops background sync.
     */
    public void stopBackgroundSync() {
        if (syncFuture != null) {
            syncFuture.cancel(false);
            syncFuture = null;
        }
        if (syncExecutor != null) {
            syncExecutor.shutdown();
            try {
                syncExecutor.awaitTermination(5, TimeUnit.SECONDS);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
            syncExecutor = null;
        }
    }
    
    /**
     * Internal cache entry with expiration.
     */
    private static class CacheEntry<T> {
        final T data;
        final Instant expiresAt;
        
        CacheEntry(T data, Instant expiresAt) {
            this.data = data;
            this.expiresAt = expiresAt;
        }
        
        boolean isValid() {
            return Instant.now().isBefore(expiresAt);
        }
    }
    
    /**
     * Result of a cache lookup.
     */
    public static class CacheResult<T> {
        private final T data;
        private final boolean valid;
        
        public CacheResult(T data, boolean valid) {
            this.data = data;
            this.valid = valid;
        }
        
        public T getData() {
            return data;
        }
        
        public boolean isValid() {
            return valid;
        }
        
        public boolean hasData() {
            return data != null;
        }
    }
}
