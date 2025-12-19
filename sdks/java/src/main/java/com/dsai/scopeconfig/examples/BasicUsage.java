/*
 * Example usage of the ScopeConfig Java SDK.
 *
 * This example demonstrates:
 * - Creating a client using environment variables
 * - Building config identifiers
 * - Getting config values with caching
 * - Using inheritance and default values
 * - Applying configuration templates
 *
 * Prerequisites:
 * 1. Build the SDK: mvn clean install
 * 2. Generate proto files: buf generate (done during build)
 * 3. Set environment variables (optional):
 *    - GRPC_SCOPE_CONFIG_HOST (default: localhost)
 *    - GRPC_SCOPE_CONFIG_PORT (default: 50051)
 *    - GRPC_SCOPE_CONFIG_USE_TLS (default: false)
 *
 * Run:
 *     mvn exec:java -Dexec.mainClass="com.dsai.scopeconfig.examples.BasicUsage"
 */
package com.dsai.scopeconfig.examples;

import com.dsai.scopeconfig.*;
import vn.dsai.config.v1.*;

import java.time.Duration;
import java.util.Arrays;
import java.util.Optional;

public class BasicUsage {

    public static void main(String[] args) {
        System.out.println("=== ScopeConfig Java SDK Example ===\n");

        // Example 1: Show environment variable configuration
        System.out.println("=== Example 1: Environment Variables ===");
        System.out.println("GRPC_SCOPE_CONFIG_HOST: " + 
            System.getenv().getOrDefault("GRPC_SCOPE_CONFIG_HOST", "localhost (default)"));
        System.out.println("GRPC_SCOPE_CONFIG_PORT: " + 
            System.getenv().getOrDefault("GRPC_SCOPE_CONFIG_PORT", "50051 (default)"));
        System.out.println("GRPC_SCOPE_CONFIG_USE_TLS: " + 
            System.getenv().getOrDefault("GRPC_SCOPE_CONFIG_USE_TLS", "false (default)"));

        // Example 2: Create client using environment variables
        System.out.println("\n=== Example 2: Using Environment Variables ===");
        try (ConfigClient clientEnv = ConfigClient.fromEnvironment()
                .cacheEnabled(true)
                .cacheTtl(Duration.ofMinutes(1))
                .backgroundSyncEnabled(true)
                .backgroundSyncInterval(Duration.ofSeconds(30))
                .build()) {
            
            System.out.println("Client created successfully using environment variables");
            runExamples(clientEnv);
            
        } catch (Exception e) {
            System.out.println("Failed to create client from env: " + e.getMessage());
            System.out.println("\nNote: To run this example with a live server, start the ScopeConfig service first.");
            demonstrateIdentifierBuilding();
        }
    }

    private static void runExamples(ConfigClient client) {
        // Example 3: Build config identifiers
        System.out.println("\n=== Example 3: Building Config Identifiers ===");
        demonstrateIdentifierBuilding();

        // Example 4: Get configuration with caching
        System.out.println("\n=== Example 4: Get Configuration with Caching ===");
        ConfigIdentifier identifier = ConfigIdentifierBuilder.create("payment-service")
                .scope(Scope.PROJECT)
                .groupId("database")
                .projectId("proj-123")
                .build();

        try {
            ScopeConfig config = client.getConfigCached(identifier);
            System.out.println("Configuration for " + 
                config.getVersionInfo().getIdentifier().getServiceName() + ":");
            for (ConfigField field : config.getFieldsList()) {
                System.out.println("  " + field.getPath() + " = " + field.getValue());
            }
        } catch (Exception e) {
            System.out.println("Failed to get config: " + e.getMessage());
        }

        // Example 5: Get specific value with inheritance
        System.out.println("\n=== Example 5: Get Value with Inheritance ===");
        try {
            Optional<String> value = client.getValue(identifier, "database.host",
                    GetValueOptions.withInheritanceAndDefaults());
            
            if (value.isPresent()) {
                System.out.println("Database host: " + value.get());
            } else {
                System.out.println("Database host not found");
            }
        } catch (Exception e) {
            System.out.println("Failed to get value: " + e.getMessage());
        }

        // Example 6: Get value as string (convenience method)
        System.out.println("\n=== Example 6: Get Value as String ===");
        try {
            String host = client.getValueString(identifier, "database.host",
                    GetValueOptions.builder().useDefault(true).build());
            System.out.println("Database host (string): '" + host + "'");
        } catch (Exception e) {
            System.out.println("Failed to get value string: " + e.getMessage());
        }

        // Example 7: Apply configuration template
        System.out.println("\n=== Example 7: Apply Configuration Template ===");
        try {
            ConfigTemplate template = ConfigTemplate.newBuilder()
                    .setIdentifier(ConfigIdentifierBuilder.create("payment-service")
                            .groupId("logging")
                            .build())
                    .setServiceLabel("Payment Service")
                    .setGroupLabel("Logging Configuration")
                    .setGroupDescription("Controls logging behavior for the payment service")
                    .addFields(ConfigFieldTemplate.newBuilder()
                            .setPath("log.level")
                            .setLabel("Log Level")
                            .setDescription("Application logging level")
                            .setType(FieldType.STRING)
                            .setDefaultValue("INFO")
                            .addAllDisplayOn(Arrays.asList(Scope.SYSTEM, Scope.PROJECT))
                            .addOptions(ValueOption.newBuilder()
                                    .setValue("DEBUG")
                                    .setLabel("Debug")
                                    .build())
                            .addOptions(ValueOption.newBuilder()
                                    .setValue("INFO")
                                    .setLabel("Info")
                                    .build())
                            .addOptions(ValueOption.newBuilder()
                                    .setValue("WARN")
                                    .setLabel("Warning")
                                    .build())
                            .addOptions(ValueOption.newBuilder()
                                    .setValue("ERROR")
                                    .setLabel("Error")
                                    .build())
                            .setSortOrder(100000)
                            .build())
                    .addFields(ConfigFieldTemplate.newBuilder()
                            .setPath("log.format")
                            .setLabel("Log Format")
                            .setDescription("Output format for log messages")
                            .setType(FieldType.STRING)
                            .setDefaultValue("json")
                            .addDisplayOn(Scope.SYSTEM)
                            .addOptions(ValueOption.newBuilder()
                                    .setValue("json")
                                    .setLabel("JSON")
                                    .build())
                            .addOptions(ValueOption.newBuilder()
                                    .setValue("text")
                                    .setLabel("Plain Text")
                                    .build())
                            .setSortOrder(100001)
                            .build())
                    .setSortOrder(100000)
                    .build();

            ConfigTemplate result = client.applyConfigTemplate(template, "admin@example.com");
            System.out.println("Applied template: " + result.getServiceLabel() + 
                " - " + result.getGroupLabel());
        } catch (Exception e) {
            System.out.println("Failed to apply template: " + e.getMessage());
        }

        // Example 8: Cache management
        System.out.println("\n=== Example 8: Cache Management ===");
        System.out.println("Cache enabled: " + client.isCacheEnabled());

        // Invalidate specific config cache
        client.invalidateCache(identifier);
        System.out.println("Cache invalidated for specific identifier");

        // Clear all cache
        client.clearCache();
        System.out.println("All cache cleared");

        System.out.println("\n=== Example Complete ===");
    }

    private static void demonstrateIdentifierBuilding() {
        // SYSTEM scope (global config)
        ConfigIdentifier systemId = ConfigIdentifierBuilder.create("my-service")
                .scope(Scope.SYSTEM)
                .groupId("database")
                .build();
        System.out.println("System identifier: service=" + systemId.getServiceName() + 
                ", group=" + systemId.getGroupId() + ", scope=" + systemId.getScope());

        // PROJECT scope
        ConfigIdentifier projectId = ConfigIdentifierBuilder.create("my-service")
                .scope(Scope.PROJECT)
                .groupId("database")
                .projectId("proj-123")
                .build();
        System.out.println("Project identifier: service=" + projectId.getServiceName() + 
                ", group=" + projectId.getGroupId() + ", project=" + projectId.getProjectId());

        // STORE scope
        ConfigIdentifier storeId = ConfigIdentifierBuilder.create("my-service")
                .scope(Scope.STORE)
                .groupId("database")
                .projectId("proj-123")
                .storeId("store-456")
                .build();
        System.out.println("Store identifier: service=" + storeId.getServiceName() + 
                ", group=" + storeId.getGroupId() + ", project=" + storeId.getProjectId() + 
                ", store=" + storeId.getStoreId());

        // USER scope
        ConfigIdentifier userId = ConfigIdentifierBuilder.create("my-service")
                .scope(Scope.USER)
                .groupId("preferences")
                .userId("user-789")
                .build();
        System.out.println("User identifier: service=" + userId.getServiceName() + 
                ", group=" + userId.getGroupId() + ", user=" + userId.getUserId());
    }
}
