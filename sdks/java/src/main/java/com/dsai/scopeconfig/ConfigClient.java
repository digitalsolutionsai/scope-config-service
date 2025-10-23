package com.dsai.scopeconfig;

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
import java.util.List;
import java.util.concurrent.TimeUnit;

/**
 * Java client for the ScopeConfig gRPC service.
 * Supports both secure (TLS) and insecure connections.
 *
 * Example:
 * <pre>{@code
 * try (ConfigClient client = ConfigClient.builder()
 *         .address("localhost:50051")
 *         .insecure()
 *         .build()) {
 *
 *     ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
 *             .scope(Scope.SYSTEM)
 *             .groupId("database")
 *             .build();
 *
 *     ScopeConfig config = client.getConfig(identifier);
 * }
 * }</pre>
 */
public class ConfigClient implements AutoCloseable {

    private final ManagedChannel channel;
    private final ConfigServiceGrpc.ConfigServiceBlockingStub blockingStub;

    private ConfigClient(ManagedChannel channel) {
        this.channel = channel;
        this.blockingStub = ConfigServiceGrpc.newBlockingStub(channel);
    }

    // Creates a new builder for constructing a ConfigClient
    public static Builder builder() {
        return new Builder();
    }

    // Retrieves the published configuration
    public ScopeConfig getConfig(ConfigIdentifier identifier) throws ConfigServiceException {
        GetConfigRequest request = GetConfigRequest.newBuilder()
                .setIdentifier(identifier)
                .build();

        try {
            return blockingStub.getConfig(request);
        } catch (StatusRuntimeException e) {
            throw ConfigServiceException.fromGrpcStatus("GetConfig", e);
        }
    }

    // Retrieves the latest configuration (published or not)
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

    // Retrieves a configuration by a specific version number
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

    // Retrieves the version history for a configuration
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

    // Updates or creates a configuration with the provided fields
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

    // Marks a specific version as published for client consumption
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

    // Deletes a configuration set and all of its associated versions
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

    // Retrieves the configuration template
    public ConfigTemplate getConfigTemplate(ConfigIdentifier identifier) throws ConfigServiceException {
        GetConfigTemplateRequest request = GetConfigTemplateRequest.newBuilder()
                .setIdentifier(identifier)
                .build();

        try {
            return blockingStub.getConfigTemplate(request);
        } catch (StatusRuntimeException e) {
            throw ConfigServiceException.fromGrpcStatus("GetConfigTemplate", e);
        }
    }

    // Applies a configuration template (schema) to a config identifier
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

    // Closes the underlying gRPC channel and releases all resources
    @Override
    public void close() throws InterruptedException {
        if (channel != null && !channel.isShutdown()) {
            channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
        }
    }

    // Builder for constructing ConfigClient instances with a fluent API
    public static class Builder {
        private String address;
        private boolean insecure = false;
        private SslContext sslContext;

        private Builder() {}

        // Sets the server address (e.g., "localhost:50051")
        public Builder address(String address) {
            this.address = address;
            return this;
        }

        // Configures the client to use an insecure connection (no TLS) - development only
        public Builder insecure() {
            this.insecure = true;
            return this;
        }

        // Configures the client to use TLS with the specified certificate
        public Builder tls(File certChainFile) throws SSLException {
            this.sslContext = GrpcSslContexts.forClient()
                    .trustManager(certChainFile)
                    .build();
            return this;
        }

        // Configures the client to use TLS with a custom SSL context
        public Builder sslContext(SslContext sslContext) {
            this.sslContext = sslContext;
            return this;
        }

        // Builds and returns a new ConfigClient instance
        public ConfigClient build() {
            if (address == null || address.isEmpty()) {
                throw new IllegalStateException("Server address is required");
            }

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

            return new ConfigClient(channel);
        }
    }
}
