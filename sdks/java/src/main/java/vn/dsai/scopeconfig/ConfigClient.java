package vn.dsai.scopeconfig;

import io.grpc.Channel;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.StatusRuntimeException;
import io.grpc.netty.shaded.io.grpc.netty.GrpcSslContexts;
import io.grpc.netty.shaded.io.grpc.netty.NettyChannelBuilder;
import io.grpc.netty.shaded.io.netty.handler.ssl.SslContext;
import vn.dsai.config.v1.*;

import javax.net.ssl.SSLException;
import java.io.File;
import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.TimeUnit;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Java client for the ScopeConfig gRPC service with caching support.
 * 
 * Features:
 * - In-memory caching for config values by group (reduces gRPC calls)
 * - In-memory caching for templates (for default value lookups)
 * - Background sync to refresh cached configs periodically
 * - Stale cache fallback when server is unavailable
 * - GetValue with inheritance and default value support
 * - Environment variable support for configuration
 *
 * Environment Variables:
 * - GRPC_SCOPE_CONFIG_HOST: Server host (default: localhost)
 * - GRPC_SCOPE_CONFIG_PORT: Server port (default: 50051)
 * - GRPC_SCOPE_CONFIG_USE_TLS: Enable TLS (default: false)
 *
 * Example:
 * <pre>{@code
 * // Using environment variables
 * try (ConfigClient client = ConfigClient.fromEnvironment().build()) {
 *     ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
 *             .scope(Scope.SYSTEM)
 *             .groupId("database")
 *             .build();
 *
 *     // Get specific value with inheritance
 *     Optional<String> value = client.getValue(identifier, "database.host",
 *             GetValueOptions.withInheritanceAndDefaults());
 * }
 * 
 * // With explicit configuration
 * try (ConfigClient client = ConfigClient.builder()
 *         .host("localhost")
 *         .port(50051)
 *         .insecure()
 *         .cacheEnabled(true)
 *         .cacheTtl(Duration.ofMinutes(1))
 *         .build()) {
 *     // ...
 * }
 * }</pre>
 */
public class ConfigClient implements AutoCloseable {
    
    private static final Logger logger = Logger.getLogger(ConfigClient.class.getName());
    
    // Environment variable names
    public static final String ENV_HOST = "GRPC_SCOPE_CONFIG_HOST";
    public static final String ENV_PORT = "GRPC_SCOPE_CONFIG_PORT";
    public static final String ENV_USE_TLS = "GRPC_SCOPE_CONFIG_USE_TLS";
    
    // Default values
    public static final String DEFAULT_HOST = "localhost";
    public static final int DEFAULT_PORT = 50051;
    public static final Duration DEFAULT_CACHE_TTL = Duration.ofMinutes(1);
    public static final Duration DEFAULT_SYNC_INTERVAL = Duration.ofSeconds(30);

    private final ManagedChannel channel;
    private final ConfigServiceGrpc.ConfigServiceBlockingStub blockingStub;
    private final ConfigCache cache;
    private final boolean cacheEnabled;

    private ConfigClient(ManagedChannel channel, ConfigCache cache, boolean cacheEnabled) {
        this.channel = channel;
        this.blockingStub = ConfigServiceGrpc.newBlockingStub(channel);
        this.cache = cache;
        this.cacheEnabled = cacheEnabled;
    }

    /**
     * Creates a new builder for constructing a ConfigClient.
     */
    public static Builder builder() {
        return new Builder();
    }
    
    /**
     * Creates a builder pre-configured from environment variables.
     */
    public static Builder fromEnvironment() {
        String host = System.getenv(ENV_HOST);
        if (host == null || host.isEmpty()) {
            host = DEFAULT_HOST;
        }
        
        int port = DEFAULT_PORT;
        String portStr = System.getenv(ENV_PORT);
        if (portStr != null && !portStr.isEmpty()) {
            try {
                port = Integer.parseInt(portStr);
            } catch (NumberFormatException e) {
                logger.warning("Invalid port in " + ENV_PORT + ": " + portStr + ", using default");
            }
        }
        
        boolean useTls = false;
        String tlsStr = System.getenv(ENV_USE_TLS);
        if (tlsStr != null) {
            useTls = tlsStr.equalsIgnoreCase("true") || 
                     tlsStr.equals("1") || 
                     tlsStr.equalsIgnoreCase("yes");
        }
        
        Builder builder = new Builder()
            .host(host)
            .port(port);
        
        if (!useTls) {
            builder.insecure();
        }
        
        return builder;
    }

    // === Config Methods ===

    /**
     * Retrieves the published configuration (always fetches from server).
     */
    public ScopeConfig getConfig(ConfigIdentifier identifier) throws ConfigServiceException {
        GetConfigRequest request = GetConfigRequest.newBuilder()
                .setIdentifier(identifier)
                .build();

        try {
            ScopeConfig config = blockingStub.getConfig(request);
            
            // Update cache if enabled
            if (cacheEnabled && cache != null) {
                cache.set(identifier, config);
            }
            
            return config;
        } catch (StatusRuntimeException e) {
            throw ConfigServiceException.fromGrpcStatus("GetConfig", e);
        }
    }
    
    /**
     * Retrieves configuration with caching support.
     * Returns cached value if valid, falls back to stale cache on error.
     */
    public ScopeConfig getConfigCached(ConfigIdentifier identifier) throws ConfigServiceException {
        // Try cache first
        if (cacheEnabled && cache != null) {
            ConfigCache.CacheResult<ScopeConfig> result = cache.get(identifier);
            if (result.hasData() && result.isValid()) {
                return result.getData();
            }
        }
        
        // Fetch from server
        try {
            return getConfig(identifier);
        } catch (ConfigServiceException e) {
            // On error, try stale cache
            if (cacheEnabled && cache != null) {
                ScopeConfig stale = cache.getStale(identifier);
                if (stale != null) {
                    logger.warning("Using stale cache for " + 
                        identifier.getServiceName() + "/" + identifier.getGroupId());
                    return stale;
                }
            }
            throw e;
        }
    }

    /**
     * Retrieves the latest configuration (published or not).
     */
    public ScopeConfig getLatestConfig(ConfigIdentifier identifier) throws ConfigServiceException {
        GetConfigRequest request = GetConfigRequest.newBuilder()
                .setIdentifier(identifier)
                .build();

        try {
            return blockingStub.getLatestConfig(request);
        } catch (StatusRuntimeException e) {
            throw ConfigServiceException.fromGrpcStatus("GetLatestConfig", e);
        }
    }

    /**
     * Retrieves a configuration by a specific version number.
     */
    public ScopeConfig getConfigByVersion(ConfigIdentifier identifier, int version) throws ConfigServiceException {
        GetConfigByVersionRequest request = GetConfigByVersionRequest.newBuilder()
                .setIdentifier(identifier)
                .setVersion(version)
                .build();

        try {
            return blockingStub.getConfigByVersion(request);
        } catch (StatusRuntimeException e) {
            throw ConfigServiceException.fromGrpcStatus("GetConfigByVersion", e);
        }
    }

    /**
     * Retrieves the version history for a configuration.
     */
    public GetConfigHistoryResponse getConfigHistory(ConfigIdentifier identifier, int limit) throws ConfigServiceException {
        GetConfigHistoryRequest request = GetConfigHistoryRequest.newBuilder()
                .setIdentifier(identifier)
                .setLimit(limit)
                .build();

        try {
            return blockingStub.getConfigHistory(request);
        } catch (StatusRuntimeException e) {
            throw ConfigServiceException.fromGrpcStatus("GetConfigHistory", e);
        }
    }

    /**
     * Updates or creates a configuration with the provided fields.
     */
    public ScopeConfig updateConfig(ConfigIdentifier identifier, List<ConfigField> fields, String user)
            throws ConfigServiceException {
        UpdateConfigRequest request = UpdateConfigRequest.newBuilder()
                .setIdentifier(identifier)
                .addAllFields(fields)
                .setUser(user)
                .build();

        try {
            return blockingStub.updateConfig(request);
        } catch (StatusRuntimeException e) {
            throw ConfigServiceException.fromGrpcStatus("UpdateConfig", e);
        }
    }

    /**
     * Marks a specific version as published for client consumption.
     */
    public ConfigVersion publishVersion(ConfigIdentifier identifier, int versionToPublish, String user)
            throws ConfigServiceException {
        PublishVersionRequest request = PublishVersionRequest.newBuilder()
                .setIdentifier(identifier)
                .setVersionToPublish(versionToPublish)
                .setUser(user)
                .build();

        try {
            return blockingStub.publishVersion(request);
        } catch (StatusRuntimeException e) {
            throw ConfigServiceException.fromGrpcStatus("PublishVersion", e);
        }
    }

    /**
     * Deletes a configuration set and all of its associated versions.
     */
    public void deleteConfig(ConfigIdentifier identifier) throws ConfigServiceException {
        DeleteConfigRequest request = DeleteConfigRequest.newBuilder()
                .setIdentifier(identifier)
                .build();

        try {
            blockingStub.deleteConfig(request);
        } catch (StatusRuntimeException e) {
            throw ConfigServiceException.fromGrpcStatus("DeleteConfig", e);
        }
    }

    // === Template Methods ===

    /**
     * Retrieves the configuration template (always fetches from server).
     */
    public ConfigTemplate getConfigTemplate(ConfigIdentifier identifier) throws ConfigServiceException {
        GetConfigTemplateRequest request = GetConfigTemplateRequest.newBuilder()
                .setIdentifier(identifier)
                .build();

        try {
            ConfigTemplate template = blockingStub.getConfigTemplate(request);
            
            // Update cache if enabled
            if (cacheEnabled && cache != null) {
                cache.setTemplate(identifier, template);
            }
            
            return template;
        } catch (StatusRuntimeException e) {
            throw ConfigServiceException.fromGrpcStatus("GetConfigTemplate", e);
        }
    }
    
    /**
     * Retrieves configuration template with caching support.
     */
    public ConfigTemplate getConfigTemplateCached(ConfigIdentifier identifier) throws ConfigServiceException {
        // Try cache first
        if (cacheEnabled && cache != null) {
            ConfigCache.CacheResult<ConfigTemplate> result = cache.getTemplate(identifier);
            if (result.hasData() && result.isValid()) {
                return result.getData();
            }
        }
        
        // Fetch from server
        try {
            return getConfigTemplate(identifier);
        } catch (ConfigServiceException e) {
            // On error, try stale cache
            if (cacheEnabled && cache != null) {
                ConfigTemplate stale = cache.getTemplateStale(identifier);
                if (stale != null) {
                    return stale;
                }
            }
            throw e;
        }
    }

    /**
     * Applies a configuration template (schema) to a config identifier.
     */
    public ConfigTemplate applyConfigTemplate(ConfigTemplate template, String user) throws ConfigServiceException {
        ApplyConfigTemplateRequest request = ApplyConfigTemplateRequest.newBuilder()
                .setTemplate(template)
                .setUser(user)
                .build();

        try {
            return blockingStub.applyConfigTemplate(request);
        } catch (StatusRuntimeException e) {
            throw ConfigServiceException.fromGrpcStatus("ApplyConfigTemplate", e);
        }
    }

    // === GetValue Methods ===

    /**
     * Gets a specific configuration value by path.
     * 
     * This method is optimized to reduce gRPC calls:
     * - Config values are fetched by group and cached
     * - Templates are cached for default value lookups
     * 
     * @param identifier Config identifier
     * @param path Field path (e.g., "database.host")
     * @param options GetValue options (useDefault, inherit)
     * @return The value as a string, or empty if not found
     */
    public Optional<String> getValue(ConfigIdentifier identifier, String path, GetValueOptions options) {
        if (options == null) {
            options = GetValueOptions.defaults();
        }
        
        // Try to get value from current scope
        Optional<String> value = getValueFromScope(identifier, path);
        if (value.isPresent()) {
            return value;
        }
        
        // If inherit is enabled, try parent scopes
        if (options.isInherit()) {
            List<ConfigIdentifier> parentIdentifiers = getParentIdentifiers(identifier);
            for (ConfigIdentifier parentId : parentIdentifiers) {
                value = getValueFromScope(parentId, path);
                if (value.isPresent()) {
                    return value;
                }
            }
        }
        
        // If useDefault is enabled, try to get default from template
        if (options.isUseDefault()) {
            Optional<String> defaultValue = getDefaultValue(identifier, path);
            if (defaultValue.isPresent()) {
                return defaultValue;
            }
        }
        
        return Optional.empty();
    }
    
    /**
     * Convenience method that returns empty string instead of Optional.empty().
     */
    public String getValueString(ConfigIdentifier identifier, String path, GetValueOptions options) {
        return getValue(identifier, path, options).orElse("");
    }
    
    private Optional<String> getValueFromScope(ConfigIdentifier identifier, String path) {
        try {
            ScopeConfig config = getConfigCached(identifier);
            for (ConfigField field : config.getFieldsList()) {
                if (field.getPath().equals(path)) {
                    return Optional.of(field.getValue());
                }
            }
            return Optional.empty();
        } catch (Exception e) {
            return Optional.empty();
        }
    }
    
    private Optional<String> getDefaultValue(ConfigIdentifier identifier, String path) {
        try {
            ConfigTemplate template = getConfigTemplateCached(identifier);
            for (ConfigFieldTemplate field : template.getFieldsList()) {
                if (field.getPath().equals(path)) {
                    return Optional.of(field.getDefaultValue());
                }
            }
            return Optional.empty();
        } catch (Exception e) {
            return Optional.empty();
        }
    }
    
    /**
     * Gets parent scope identifiers for inheritance.
     * 
     * The inheritance hierarchy is:
     *   SYSTEM
     *   ├── PROJECT → STORE
     *   └── USER
     *
     * So: STORE → PROJECT → SYSTEM, USER → SYSTEM, PROJECT → SYSTEM
     */
    private List<ConfigIdentifier> getParentIdentifiers(ConfigIdentifier identifier) {
        List<ConfigIdentifier> parents = new ArrayList<>();
        
        switch (identifier.getScope()) {
            case USER:
                // User -> System (USER is at same level as PROJECT, not under STORE)
                parents.add(ConfigIdentifier.newBuilder()
                    .setServiceName(identifier.getServiceName())
                    .setGroupId(identifier.getGroupId())
                    .setScope(Scope.SYSTEM)
                    .build());
                break;
                
            case STORE:
                // Store -> Project -> System
                if (!identifier.getProjectId().isEmpty()) {
                    parents.add(ConfigIdentifier.newBuilder()
                        .setServiceName(identifier.getServiceName())
                        .setGroupId(identifier.getGroupId())
                        .setScope(Scope.PROJECT)
                        .setProjectId(identifier.getProjectId())
                        .build());
                }
                parents.add(ConfigIdentifier.newBuilder()
                    .setServiceName(identifier.getServiceName())
                    .setGroupId(identifier.getGroupId())
                    .setScope(Scope.SYSTEM)
                    .build());
                break;
                
            case PROJECT:
                // Project -> System
                parents.add(ConfigIdentifier.newBuilder()
                    .setServiceName(identifier.getServiceName())
                    .setGroupId(identifier.getGroupId())
                    .setScope(Scope.SYSTEM)
                    .build());
                break;
                
            case SYSTEM:
                // System has no parent
                break;
                
            default:
                break;
        }
        
        return parents;
    }

    // === Cache Management ===

    /**
     * Invalidates the cache for a specific identifier.
     */
    public void invalidateCache(ConfigIdentifier identifier) {
        if (cache != null) {
            cache.invalidate(identifier);
        }
    }

    /**
     * Clears all cached configurations.
     */
    public void clearCache() {
        if (cache != null) {
            cache.clear();
        }
    }

    /**
     * Returns whether caching is enabled.
     */
    public boolean isCacheEnabled() {
        return cacheEnabled;
    }

    /**
     * Closes the underlying gRPC channel and releases all resources.
     */
    @Override
    public void close() throws InterruptedException {
        if (cache != null) {
            cache.stopBackgroundSync();
        }
        
        if (channel != null && !channel.isShutdown()) {
            channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
        }
    }

    /**
     * Builder for constructing ConfigClient instances with a fluent API.
     */
    public static class Builder {
        private String host = DEFAULT_HOST;
        private int port = DEFAULT_PORT;
        private boolean insecure = false;
        private SslContext sslContext;
        private boolean cacheEnabled = true;
        private Duration cacheTtl = DEFAULT_CACHE_TTL;
        private boolean backgroundSyncEnabled = false;
        private Duration backgroundSyncInterval = DEFAULT_SYNC_INTERVAL;

        private Builder() {}

        /**
         * Sets the server host.
         */
        public Builder host(String host) {
            this.host = host;
            return this;
        }

        /**
         * Sets the server port.
         */
        public Builder port(int port) {
            this.port = port;
            return this;
        }

        /**
         * Sets the server address (e.g., "localhost:50051").
         * @deprecated Use host() and port() instead for consistency with environment variables.
         */
        @Deprecated
        public Builder address(String address) {
            if (address != null && address.contains(":")) {
                String[] parts = address.split(":");
                this.host = parts[0];
                if (parts.length > 1) {
                    try {
                        this.port = Integer.parseInt(parts[1]);
                    } catch (NumberFormatException e) {
                        // Keep default port
                    }
                }
            } else {
                this.host = address;
            }
            return this;
        }

        /**
         * Configures the client to use an insecure connection (no TLS) - development only.
         */
        public Builder insecure() {
            this.insecure = true;
            return this;
        }

        /**
         * Configures the client to use TLS with the specified certificate.
         */
        public Builder tls(File certChainFile) throws SSLException {
            this.sslContext = GrpcSslContexts.forClient()
                    .trustManager(certChainFile)
                    .build();
            return this;
        }

        /**
         * Configures the client to use TLS with a custom SSL context.
         */
        public Builder sslContext(SslContext sslContext) {
            this.sslContext = sslContext;
            return this;
        }

        /**
         * Enables or disables caching.
         */
        public Builder cacheEnabled(boolean enabled) {
            this.cacheEnabled = enabled;
            return this;
        }

        /**
         * Sets the cache TTL.
         */
        public Builder cacheTtl(Duration ttl) {
            this.cacheTtl = ttl;
            return this;
        }

        /**
         * Enables background sync.
         */
        public Builder backgroundSyncEnabled(boolean enabled) {
            this.backgroundSyncEnabled = enabled;
            return this;
        }

        /**
         * Sets the background sync interval.
         */
        public Builder backgroundSyncInterval(Duration interval) {
            this.backgroundSyncInterval = interval;
            return this;
        }

        /**
         * Builds and returns a new ConfigClient instance.
         */
        public ConfigClient build() {
            String address = host + ":" + port;
            
            ManagedChannel channel;

            if (insecure) {
                channel = ManagedChannelBuilder
                        .forTarget(address)
                        .usePlaintext()
                        .build();
            } else if (sslContext != null) {
                channel = NettyChannelBuilder
                        .forTarget(address)
                        .sslContext(sslContext)
                        .build();
            } else {
                // Default to TLS without custom cert (uses system trust store)
                channel = ManagedChannelBuilder
                        .forTarget(address)
                        .build();
            }

            ConfigCache cache = null;
            if (cacheEnabled) {
                cache = new ConfigCache(cacheTtl);
            }

            ConfigClient client = new ConfigClient(channel, cache, cacheEnabled);
            
            // Start background sync after client is created to avoid circular reference
            if (cacheEnabled && backgroundSyncEnabled && cache != null) {
                final ConfigClient syncClient = client;
                cache.startBackgroundSync(backgroundSyncInterval, identifier -> {
                    try {
                        syncClient.getConfig(identifier);
                    } catch (Exception e) {
                        // Silently fail - stale cache will be used
                    }
                });
            }

            return client;
        }
    }
}
