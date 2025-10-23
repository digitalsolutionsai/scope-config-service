# Java SDK for ScopeConfig Service

A modern, type-safe Java client for interacting with the ScopeConfig gRPC service.

## Prerequisites

- **Java 22+** - This SDK leverages modern Java features
- **Maven 3.9+** - Build tool
- **Buf CLI** - For protobuf code generation ([installation guide](https://buf.build/docs/installation))

### Install Protobuf Plugins

For manual protobuf generation (when not using buf):

```bash
# These are Maven plugins and will be automatically downloaded when building
# No manual installation required if using Maven
```

## Installation

### 1. Copy the SDK to Your Project

```bash
# Copy the entire sdks/java directory to your project
cp -r sdks/java /path/to/your/project/scopeconfig-sdk
cd /path/to/your/project/scopeconfig-sdk
```

### 2. Update Maven Configuration

Edit `pom.xml` and update the `groupId` and `artifactId` to match your project:

```xml
<groupId>com.yourcompany</groupId>
<artifactId>scopeconfig-sdk</artifactId>
```

### 3. Copy Proto Files

```bash
# Copy proto files to the SDK directory
mkdir -p proto
cp -r /path/to/scope-config-service/proto/config proto/
```

### 4. Generate Protobuf Code

Using Buf (recommended):

```bash
# Generate gRPC client code using buf
buf generate ../../proto
```

Or using Maven:

```bash
# Maven will automatically generate code during compile phase
mvn clean compile
```

This will generate the gRPC client code in `src/main/java/vn/dsai/config/v1/` (which is gitignored).

### 5. Build the SDK

```bash
mvn clean install
```

## Usage

### Basic Example

```java
import com.dsai.scopeconfig.*;
import vn.dsai.config.v1.*;

public class Example {
    public static void main(String[] args) throws Exception {
        // Create a client with try-with-resources for automatic cleanup
        try (ConfigClient client = ConfigClient.builder()
                .address("localhost:50051")
                .insecure()  // Use .tls() in production
                .build()) {

            // Build a configuration identifier
            ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
                    .scope(Scope.SYSTEM)
                    .groupId("database")
                    .build();

            // Get published configuration
            ScopeConfig config = client.getConfig(identifier);
            System.out.println("Configuration for: " +
                config.getVersionInfo().getIdentifier().getServiceName());

            // Print all fields
            for (ConfigField field : config.getFieldsList()) {
                System.out.printf("  %s = %s (type: %s)%n",
                    field.getPath(),
                    field.getValue(),
                    field.getType());
            }
        }
    }
}
```

### Update Configuration

```java
try (ConfigClient client = ConfigClient.builder()
        .address("localhost:50051")
        .insecure()
        .build()) {

    ConfigIdentifier identifier = ConfigIdentifierBuilder.create("payment-service")
            .scope(Scope.PROJECT)
            .groupId("api")
            .projectId("proj-123")
            .build();

    // Create configuration fields using builder
    List<ConfigField> fields = List.of(
        ConfigFieldBuilder.create("api.timeout").intValue(30).build(),
        ConfigFieldBuilder.create("api.retry_count").intValue(3).build(),
        ConfigFieldBuilder.create("api.base_url").stringValue("https://api.example.com").build()
    );

    // Update configuration
    ScopeConfig updatedConfig = client.updateConfig(
        identifier,
        fields,
        "admin@example.com"
    );

    System.out.printf("Updated to version %d%n", updatedConfig.getCurrentVersion());
}
```

### Apply Configuration Template

```java
try (ConfigClient client = ConfigClient.builder()
        .address("localhost:50051")
        .insecure()
        .build()) {

    ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
            .groupId("logging")
            .build();

    // Create a configuration template
    ConfigTemplate template = ConfigTemplate.newBuilder()
            .setIdentifier(identifier)
            .setServiceLabel("My Service")
            .setGroupLabel("Logging Configuration")
            .setGroupDescription("Controls logging behavior for the application")
            .addFields(ConfigFieldTemplate.newBuilder()
                    .setPath("log.level")
                    .setLabel("Log Level")
                    .setDescription("Application logging level")
                    .setType(FieldType.STRING)
                    .setDefaultValue("INFO")
                    .addDisplayOn(Scope.SYSTEM)
                    .addDisplayOn(Scope.PROJECT)
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
                    .build())
            .build();

    // Apply the template
    ConfigTemplate applied = client.applyConfigTemplate(template, "admin@example.com");
    System.out.printf("Applied template: %s%n", applied.getGroupLabel());
}
```

### Get Configuration Template

```java
try (ConfigClient client = ConfigClient.builder()
        .address("localhost:50051")
        .insecure()
        .build()) {

    ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
            .groupId("logging")
            .build();

    ConfigTemplate template = client.getConfigTemplate(identifier);

    System.out.printf("Template: %s%n", template.getGroupLabel());
    System.out.printf("Description: %s%n", template.getGroupDescription());
    System.out.println("Fields:");

    for (ConfigFieldTemplate field : template.getFieldsList()) {
        System.out.printf("  - %s (%s): %s%n",
            field.getLabel(),
            field.getPath(),
            field.getDescription());
    }
}
```

### Version Management

```java
try (ConfigClient client = ConfigClient.builder()
        .address("localhost:50051")
        .insecure()
        .build()) {

    ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
            .scope(Scope.SYSTEM)
            .groupId("feature-flags")
            .build();

    // Get configuration by specific version
    ScopeConfig historicalConfig = client.getConfigByVersion(identifier, 5);
    System.out.printf("Configuration at version %d%n",
        historicalConfig.getCurrentVersion());

    // Get version history
    GetConfigHistoryResponse history = client.getConfigHistory(identifier, 10);
    System.out.println("Version history:");
    for (VersionHistoryEntry entry : history.getHistoryList()) {
        System.out.printf("  Version %d by %s at %s%n",
            entry.getVersion(),
            entry.getCreatedBy(),
            entry.getCreatedAt());
    }

    // Publish a specific version
    ConfigVersion published = client.publishVersion(identifier, 7, "admin@example.com");
    System.out.printf("Published version %d%n", published.getPublishedVersion());
}
```

### Using TLS in Production

```java
import java.io.File;

try (ConfigClient client = ConfigClient.builder()
        .address("config-service.example.com:443")
        .tls(new File("/path/to/ca-cert.pem"))
        .build()) {

    // Use the client as normal
    ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
            .scope(Scope.SYSTEM)
            .groupId("database")
            .build();

    ScopeConfig config = client.getConfig(identifier);
    // ...
}
```

### Error Handling

```java
try (ConfigClient client = ConfigClient.builder()
        .address("localhost:50051")
        .insecure()
        .build()) {

    ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
            .scope(Scope.SYSTEM)
            .groupId("database")
            .build();

    try {
        ScopeConfig config = client.getConfig(identifier);
        // Process config...
    } catch (ConfigNotFoundException e) {
        System.err.println("Configuration not found: " + e.getMessage());
    } catch (InvalidConfigException e) {
        System.err.println("Invalid configuration: " + e.getMessage());
    } catch (ConfigServiceException e) {
        System.err.printf("Service error (%s): %s%n",
            e.getStatusCode(),
            e.getMessage());
    }
}
```

## API Reference

### ConfigClient

Main client class for interacting with the ScopeConfig service.

#### Methods

- `getConfig(ConfigIdentifier identifier)` - Get published configuration
- `getLatestConfig(ConfigIdentifier identifier)` - Get latest configuration (published or not)
- `getConfigByVersion(ConfigIdentifier identifier, int version)` - Get configuration by version
- `getConfigHistory(ConfigIdentifier identifier, int limit)` - Get version history
- `updateConfig(ConfigIdentifier identifier, List<ConfigField> fields, String user)` - Update configuration
- `publishVersion(ConfigIdentifier identifier, int versionToPublish, String user)` - Publish a version
- `deleteConfig(ConfigIdentifier identifier)` - Delete configuration
- `getConfigTemplate(ConfigIdentifier identifier)` - Get configuration template
- `applyConfigTemplate(ConfigTemplate template, String user)` - Apply configuration template

#### Builder Options

- `address(String address)` - Set server address
- `insecure()` - Use insecure connection (development only)
- `tls(File certChainFile)` - Use TLS with certificate
- `sslContext(SslContext sslContext)` - Use custom SSL context

### ConfigIdentifierBuilder

Builder for creating `ConfigIdentifier` objects.

#### Methods

- `create(String serviceName)` - Create builder with service name (required)
- `scope(Scope scope)` - Set scope
- `groupId(String groupId)` - Set group ID
- `projectId(String projectId)` - Set project ID (max 20 chars)
- `storeId(String storeId)` - Set store ID (max 20 chars)
- `userId(String userId)` - Set user ID (max 36 chars)
- `build()` - Build the ConfigIdentifier

### ConfigFieldBuilder

Builder for creating `ConfigField` objects with proper type handling.

#### Methods

- `create(String path)` - Create builder with field path
- `stringValue(String value)` - Set string value
- `intValue(int value)` - Set integer value
- `longValue(long value)` - Set long value
- `floatValue(float value)` - Set float value
- `doubleValue(double value)` - Set double value
- `booleanValue(boolean value)` - Set boolean value
- `jsonValue(String jsonValue)` - Set JSON value
- `value(String value, FieldType type)` - Set raw value with explicit type
- `build()` - Build the ConfigField

### Scope Enum

Available in `vn.dsai.config.v1.Scope`:

- `SCOPE_UNSPECIFIED` (0)
- `SYSTEM` (1) - System-wide default configuration
- `PROJECT` (2) - Project-specific configuration
- `STORE` (3) - Store-specific configuration
- `USER` (4) - User-specific configuration

### FieldType Enum

Available in `vn.dsai.config.v1.FieldType`:

- `FIELD_TYPE_UNSPECIFIED` (0)
- `STRING` (1)
- `INT` (2)
- `FLOAT` (3)
- `BOOLEAN` (4)
- `JSON` (5)
- `ARRAY_STRING` (6)

## Testing

Run the tests (requires a running ScopeConfig service):

```bash
# Start the ScopeConfig service first on localhost:50051
# Then remove @Disabled annotation from test class and run:
mvn test
```

## Integration with Your Java Application

### Maven Dependency

If you publish this SDK to a Maven repository, add it to your `pom.xml`:

```xml
<dependency>
    <groupId>com.dsai</groupId>
    <artifactId>scopeconfig-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Required gRPC Dependencies

If you're integrating this SDK directly into your project without publishing it to a repository, you need to add these gRPC dependencies to your `pom.xml`:

```xml
<properties>
    <grpc.version>1.65.1</grpc.version>
    <protobuf.version>3.25.3</protobuf.version>
</properties>

<dependencies>
    <!-- gRPC runtime -->
    <dependency>
        <groupId>io.grpc</groupId>
        <artifactId>grpc-netty-shaded</artifactId>
        <version>${grpc.version}</version>
        <scope>runtime</scope>
    </dependency>

    <!-- gRPC protobuf support -->
    <dependency>
        <groupId>io.grpc</groupId>
        <artifactId>grpc-protobuf</artifactId>
        <version>${grpc.version}</version>
    </dependency>

    <!-- gRPC stub support -->
    <dependency>
        <groupId>io.grpc</groupId>
        <artifactId>grpc-stub</artifactId>
        <version>${grpc.version}</version>
    </dependency>

    <!-- Protobuf Java -->
    <dependency>
        <groupId>com.google.protobuf</groupId>
        <artifactId>protobuf-java</artifactId>
        <version>${protobuf.version}</version>
    </dependency>

    <!-- Annotations -->
    <dependency>
        <groupId>javax.annotation</groupId>
        <artifactId>javax.annotation-api</artifactId>
        <version>1.3.2</version>
    </dependency>
</dependencies>
```

### Spring Boot Integration

Example Spring configuration:

```java
@Configuration
public class ScopeConfigConfiguration {

    @Bean
    public ConfigClient configClient(
            @Value("${scopeconfig.address}") String address,
            @Value("${scopeconfig.tls.enabled}") boolean tlsEnabled,
            @Value("${scopeconfig.tls.cert:}") String certPath) throws Exception {

        ConfigClient.Builder builder = ConfigClient.builder()
                .address(address);

        if (tlsEnabled && !certPath.isEmpty()) {
            builder.tls(new File(certPath));
        } else {
            builder.insecure();
        }

        return builder.build();
    }
}
```

## Building from Source

```bash
# Clone the repository
git clone <repository-url>
cd scope-config-service/sdks/java

# Generate protobuf code
buf generate ../../proto

# Build the project
mvn clean install

# Run tests (requires running service)
mvn test
```

## Troubleshooting

### Protobuf Generation Issues

If `buf generate` fails, ensure:
- Buf CLI is installed: `buf --version`
- Proto files are in the correct location: `../../proto/config/v1/config.proto`
- Protobuf plugins are accessible

### Connection Issues

If you get connection errors:
- Verify the service is running: `grpc_cli ls localhost:50051`
- Check firewall settings
- Ensure you're using the correct address and port

### Java Version Issues

This SDK requires Java 22+. Check your version:
```bash
java -version
```

If you need to use an older Java version, edit `pom.xml` and change:
- `<maven.compiler.source>22</maven.compiler.source>`
- `<maven.compiler.target>22</maven.compiler.target>`
- Remove `<arg>--enable-preview</arg>` from compiler configuration

## License

See the main project LICENSE file.
