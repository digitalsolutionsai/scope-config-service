package com.dsai.scopeconfig;

import vn.dsai.config.v1.ConfigIdentifier;
import vn.dsai.config.v1.Scope;

/**
 * Fluent builder for constructing ConfigIdentifier objects.
 *
 * Example:
 * <pre>{@code
 * ConfigIdentifier identifier = ConfigIdentifierBuilder.create("payment-service")
 *         .scope(Scope.PROJECT)
 *         .groupId("api")
 *         .projectId("proj-123")
 *         .build();
 * }</pre>
 */
public class ConfigIdentifierBuilder {

    private final ConfigIdentifier.Builder builder;

    private ConfigIdentifierBuilder(String serviceName) {
        this.builder = ConfigIdentifier.newBuilder()
                .setServiceName(serviceName)
                .setScope(Scope.SCOPE_UNSPECIFIED);
    }

    // Creates a new builder with the required service name
    public static ConfigIdentifierBuilder create(String serviceName) {
        if (serviceName == null || serviceName.isEmpty()) {
            throw new IllegalArgumentException("Service name is required");
        }
        return new ConfigIdentifierBuilder(serviceName);
    }

    // Sets the scope for the configuration
    public ConfigIdentifierBuilder scope(Scope scope) {
        if (scope != null) {
            builder.setScope(scope);
        }
        return this;
    }

    // Sets the group ID
    public ConfigIdentifierBuilder groupId(String groupId) {
        if (groupId != null && !groupId.isEmpty()) {
            builder.setGroupId(groupId);
        }
        return this;
    }

    // Sets the project ID (max 20 characters)
    public ConfigIdentifierBuilder projectId(String projectId) {
        if (projectId != null && !projectId.isEmpty()) {
            builder.setProjectId(projectId);
        }
        return this;
    }

    // Sets the store ID (max 20 characters)
    public ConfigIdentifierBuilder storeId(String storeId) {
        if (storeId != null && !storeId.isEmpty()) {
            builder.setStoreId(storeId);
        }
        return this;
    }

    // Sets the user ID (max 36 characters, e.g., UUID)
    public ConfigIdentifierBuilder userId(String userId) {
        if (userId != null && !userId.isEmpty()) {
            builder.setUserId(userId);
        }
        return this;
    }

    // Builds and returns the ConfigIdentifier
    public ConfigIdentifier build() {
        return builder.build();
    }
}
