package vn.dsai.scopeconfig;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.Disabled;
import vn.dsai.config.v1.*;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Integration tests for the ConfigClient.
 * Requires a running ScopeConfig service on localhost:50051.
 * Remove @Disabled and run: mvn test
 */
@Disabled("Integration tests require a running ScopeConfig service")
class ConfigClientIntegrationTest {

    private ConfigClient client;
    private ConfigIdentifier testIdentifier;

    @BeforeEach
    void setUp() {
        // Create client
        client = ConfigClient.builder()
                .address("localhost:50051")
                .insecure()
                .build();

        // Create test identifier
        testIdentifier = ConfigIdentifierBuilder.create("test-service")
                .scope(Scope.SYSTEM)
                .groupId("integration-test")
                .build();
    }

    @AfterEach
    void tearDown() throws InterruptedException {
        if (client != null) {
            client.close();
        }
    }

    @Test
    void testUpdateAndGetConfig() throws ConfigServiceException {
        // Create config fields
        List<ConfigField> fields = List.of(
                ConfigFieldBuilder.create("test.key1").stringValue("value1").build(),
                ConfigFieldBuilder.create("test.key2").intValue(42).build(),
                ConfigFieldBuilder.create("test.key3").booleanValue(true).build()
        );

        // Update configuration
        ScopeConfig updatedConfig = client.updateConfig(
                testIdentifier,
                fields,
                "test-user@example.com"
        );

        assertNotNull(updatedConfig);
        assertEquals(3, updatedConfig.getFieldsCount());
        assertTrue(updatedConfig.getCurrentVersion() > 0);

        // Get latest configuration
        ScopeConfig latestConfig = client.getLatestConfig(testIdentifier);
        assertNotNull(latestConfig);
        assertEquals(3, latestConfig.getFieldsCount());
        assertEquals(updatedConfig.getCurrentVersion(), latestConfig.getCurrentVersion());

        // Verify field values
        ConfigField field1 = latestConfig.getFields(0);
        assertEquals("test.key1", field1.getPath());
        assertEquals("value1", field1.getValue());
        assertEquals(FieldType.STRING, field1.getType());
    }

    @Test
    void testConfigTemplate() throws ConfigServiceException {
        // Create configuration template
        ConfigTemplate template = ConfigTemplate.newBuilder()
                .setIdentifier(testIdentifier)
                .setServiceLabel("Test Service")
                .setGroupLabel("Integration Test Configuration")
                .setGroupDescription("Configuration for integration tests")
                .addFields(ConfigFieldTemplate.newBuilder()
                        .setPath("log.level")
                        .setLabel("Log Level")
                        .setDescription("Application logging level")
                        .setType(FieldType.STRING)
                        .setDefaultValue("INFO")
                        .addDisplayOn(Scope.SYSTEM)
                        .addOptions(ValueOption.newBuilder()
                                .setValue("DEBUG")
                                .setLabel("Debug")
                                .build())
                        .addOptions(ValueOption.newBuilder()
                                .setValue("INFO")
                                .setLabel("Info")
                                .build())
                        .addOptions(ValueOption.newBuilder()
                                .setValue("ERROR")
                                .setLabel("Error")
                                .build())
                        .build())
                .build();

        // Apply template
        ConfigTemplate appliedTemplate = client.applyConfigTemplate(
                template,
                "test-user@example.com"
        );

        assertNotNull(appliedTemplate);
        assertEquals("Test Service", appliedTemplate.getServiceLabel());
        assertEquals("Integration Test Configuration", appliedTemplate.getGroupLabel());
        assertEquals(1, appliedTemplate.getFieldsCount());

        // Get template
        ConfigTemplate retrievedTemplate = client.getConfigTemplate(testIdentifier);
        assertNotNull(retrievedTemplate);
        assertEquals("Test Service", retrievedTemplate.getServiceLabel());
    }

    @Test
    void testConfigVersioning() throws ConfigServiceException {
        // Create initial version
        List<ConfigField> fields1 = List.of(
                ConfigFieldBuilder.create("version.test").stringValue("v1").build()
        );
        ScopeConfig config1 = client.updateConfig(testIdentifier, fields1, "test-user");
        int version1 = config1.getCurrentVersion();

        // Create second version
        List<ConfigField> fields2 = List.of(
                ConfigFieldBuilder.create("version.test").stringValue("v2").build()
        );
        ScopeConfig config2 = client.updateConfig(testIdentifier, fields2, "test-user");
        int version2 = config2.getCurrentVersion();

        // Verify version numbers increased
        assertTrue(version2 > version1);

        // Get specific version
        ScopeConfig historicalConfig = client.getConfigByVersion(testIdentifier, version1);
        assertNotNull(historicalConfig);
        assertEquals(version1, historicalConfig.getCurrentVersion());
        assertEquals("v1", historicalConfig.getFields(0).getValue());

        // Get config history
        GetConfigHistoryResponse history = client.getConfigHistory(testIdentifier, 10);
        assertNotNull(history);
        assertTrue(history.getHistoryCount() >= 2);
    }

    @Test
    void testPublishVersion() throws ConfigServiceException {
        // Create a configuration
        List<ConfigField> fields = List.of(
                ConfigFieldBuilder.create("publish.test").stringValue("published").build()
        );
        ScopeConfig config = client.updateConfig(testIdentifier, fields, "test-user");
        int version = config.getCurrentVersion();

        // Publish the version
        ConfigVersion publishedVersion = client.publishVersion(
                testIdentifier,
                version,
                "test-user"
        );

        assertNotNull(publishedVersion);
        assertEquals(version, publishedVersion.getPublishedVersion());

        // Get published config
        ScopeConfig publishedConfig = client.getConfig(testIdentifier);
        assertNotNull(publishedConfig);
        assertEquals(version, publishedConfig.getCurrentVersion());
    }

    @Test
    void testConfigNotFound() {
        ConfigIdentifier nonExistentIdentifier = ConfigIdentifierBuilder.create("non-existent-service")
                .scope(Scope.SYSTEM)
                .groupId("non-existent")
                .build();

        assertThrows(ConfigNotFoundException.class, () -> {
            client.getConfig(nonExistentIdentifier);
        });
    }

    @Test
    void testInvalidConfig() {
        ConfigIdentifier invalidIdentifier = ConfigIdentifierBuilder.create("")
                .build();

        assertThrows(IllegalArgumentException.class, () -> {
            ConfigIdentifierBuilder.create("");
        });
    }
}
