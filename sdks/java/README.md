# Java SDK for ScopeConfig Service

A Java client for the ScopeConfig gRPC service with caching support.

## Features

- **In-memory caching** for config values by group (reduces gRPC calls)
- **Template caching** for default value lookups
- **Background sync** to refresh cached configs periodically
- **Stale cache fallback** when server is unavailable
- **GetValue** with inheritance and default value support
- **Environment variable support** for configuration
- **Automatic template loading** from YAML files

## Prerequisites

- Java 22+
- Maven 3.9+ or Gradle 8+

## Quick Start

### Installation

This package is published to **GitHub Packages**. Follow these steps to install:

#### Option 1: Maven (Recommended)

**1. Create a GitHub Personal Access Token:**

- Go to [GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)](https://github.com/settings/tokens)
- Click **"Generate new token (classic)"**
- Give it a name (e.g., `maven-packages`)
- Select scope: **`read:packages`**
- Click **Generate token** and copy it

**2. Configure Maven to use GitHub Packages:**

Add the following to your `~/.m2/settings.xml`:

```xml
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0
                              https://maven.apache.org/xsd/settings-1.0.0.xsd">
  <servers>
    <server>
      <id>github</id>
      <username>YOUR_GITHUB_USERNAME</username>
      <password>YOUR_GITHUB_TOKEN</password>
    </server>
  </servers>
</settings>
```

> ⚠️ **Security Note**: Never commit `settings.xml` with hardcoded tokens to version control. For CI/CD pipelines, use environment variables or encrypted secrets instead.

**3. Add the repository and dependency to your `pom.xml`:**

```xml
<repositories>
    <repository>
        <id>github</id>
        <url>https://maven.pkg.github.com/digitalsolutionsai/scope-config-service</url>
    </repository>
</repositories>

<dependencies>
    <dependency>
        <groupId>com.dsai</groupId>
        <artifactId>scopeconfig-sdk</artifactId>
        <version>1.0.0</version>
    </dependency>
</dependencies>
```

#### Option 2: Gradle

**1. Configure Gradle to use GitHub Packages:**

Add to your `build.gradle`:

```groovy
repositories {
    maven {
        url = uri("https://maven.pkg.github.com/digitalsolutionsai/scope-config-service")
        credentials {
            username = project.findProperty("gpr.user") ?: System.getenv("GITHUB_USERNAME")
            password = project.findProperty("gpr.key") ?: System.getenv("GITHUB_TOKEN")
        }
    }
    mavenCentral()
}

dependencies {
    implementation 'com.dsai:scopeconfig-sdk:1.0.0'
}
```

**2. Set credentials** in `~/.gradle/gradle.properties`:

```properties
gpr.user=YOUR_GITHUB_USERNAME
gpr.key=YOUR_GITHUB_TOKEN
```

#### Option 3: Local Installation (Development)

If you want to build from source:

```bash
# Clone the repository
git clone https://github.com/digitalsolutionsai/scope-config-service.git
cd scope-config-service/sdks/java

# Copy proto files
mkdir -p proto
cp -r ../../proto/config proto/

# Generate gRPC code
buf generate proto

# Build and install to local Maven repository
mvn clean install
```

#### Verify Installation

```bash
mvn dependency:tree | grep scopeconfig
```

> **Access Requirements**: You need read access to the [digitalsolutionsai/scope-config-service](https://github.com/digitalsolutionsai/scope-config-service) repository to install this package.

### Using Environment Variables

```java
import com.dsai.scopeconfig.*;
import vn.dsai.config.v1.*;

// Environment variables:
// GRPC_SCOPE_CONFIG_HOST (default: localhost)
// GRPC_SCOPE_CONFIG_PORT (default: 50051)
// GRPC_SCOPE_CONFIG_USE_TLS (default: false)

try (ConfigClient client = ConfigClient.fromEnvironment().build()) {
    ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
            .scope(Scope.PROJECT)
            .groupId("database")
            .projectId("proj-123")
            .build();

    // Get specific value with inheritance
    Optional<String> value = client.getValue(identifier, "database.host",
            GetValueOptions.withInheritanceAndDefaults());
    
    value.ifPresent(v -> System.out.println("Database host: " + v));
}
```

### With Explicit Configuration

```java
try (ConfigClient client = ConfigClient.builder()
        .host("localhost")
        .port(50051)
        .insecure()
        .cacheEnabled(true)
        .cacheTtl(Duration.ofMinutes(1))
        .backgroundSyncEnabled(true)
        .backgroundSyncInterval(Duration.ofSeconds(30))
        .build()) {

    // Get config with caching
    ScopeConfig config = client.getConfigCached(identifier);
    for (ConfigField field : config.getFieldsList()) {
        System.out.printf("%s = %s%n", field.getPath(), field.getValue());
    }
}
```

## Client Options

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `host` | `GRPC_SCOPE_CONFIG_HOST` | `localhost` | Server host |
| `port` | `GRPC_SCOPE_CONFIG_PORT` | `50051` | Server port |
| `useTls` | `GRPC_SCOPE_CONFIG_USE_TLS` | `false` | Enable TLS |
| `cacheEnabled` | - | `true` | Enable caching |
| `cacheTtl` | - | `1 minute` | Cache TTL |
| `backgroundSyncEnabled` | - | `false` | Enable background sync |
| `backgroundSyncInterval` | - | `30 seconds` | Sync interval |

## Caching Behavior

The SDK provides built-in caching to minimize gRPC calls and improve performance.

### In-Memory Cache

- **Config values** are cached by group identifier (service name, group ID, scope, project/store/user IDs)
- **Templates** are cached for default value lookups
- Cache entries expire after the configured TTL (default: 1 minute)

### Stale Cache Fallback

When the server is unavailable, the SDK automatically falls back to cached data even if expired:

```java
// If server is down, returns stale cached data with a warning log
ScopeConfig config = client.getConfigCached(identifier);
```

### Background Sync (Auto Refresh)

Enable background sync to automatically refresh cached configs at regular intervals:

```java
try (ConfigClient client = ConfigClient.builder()
        .host("localhost")
        .port(50051)
        .insecure()
        .cacheEnabled(true)
        .cacheTtl(Duration.ofMinutes(5))
        .backgroundSyncEnabled(true)                    // Enable auto refresh
        .backgroundSyncInterval(Duration.ofSeconds(30)) // Refresh every 30 seconds
        .build()) {
    
    // Configs are automatically refreshed in the background
    // First call populates cache, subsequent calls use cached data
    ScopeConfig config = client.getConfigCached(identifier);
}
```

### Cache Management

```java
// Invalidate cache for specific config
client.invalidateCache(identifier);

// Clear all cached configs and templates
client.clearCache();

// Check if caching is enabled
boolean enabled = client.isCacheEnabled();
```

### Cache Flow

1. **First request**: Fetches from server, stores in cache
2. **Subsequent requests**: Returns cached data if not expired
3. **Expired cache + server available**: Fetches fresh data, updates cache
4. **Expired cache + server unavailable**: Returns stale cached data (fallback)
5. **Background sync**: Periodically refreshes all cached entries

## API Reference

### Client Methods

- `getConfig(identifier)` - Get config (always fetches from server)
- `getConfigCached(identifier)` - Get config with caching support
- `getLatestConfig(identifier)` - Get latest config (unpublished)
- `getConfigByVersion(identifier, version)` - Get config by version
- `getConfigHistory(identifier, limit)` - Get version history
- `updateConfig(identifier, fields, user)` - Update configuration
- `publishVersion(identifier, version, user)` - Publish a version
- `deleteConfig(identifier)` - Delete configuration
- `getConfigTemplate(identifier)` - Get template (always fetches from server)
- `getConfigTemplateCached(identifier)` - Get template with caching
- `applyConfigTemplate(template, user)` - Apply configuration template
- `getValue(identifier, path, options)` - Get specific value with options
- `getValueString(identifier, path, options)` - Get value as string (empty if not found)
- `invalidateCache(identifier)` - Invalidate cache for specific config
- `clearCache()` - Clear all cached configs
- `isCacheEnabled()` - Check if caching is enabled

### GetValueOptions

```java
// Default options
GetValueOptions options = GetValueOptions.defaults();

// With inheritance and defaults
GetValueOptions options = GetValueOptions.withInheritanceAndDefaults();

// Custom options
GetValueOptions options = GetValueOptions.builder()
    .useDefault(true)
    .inherit(true)
    .build();
```

### Identifier Builder

```java
ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
    .scope(Scope.PROJECT)
    .groupId("database")
    .projectId("proj-123")
    .storeId("store-456")
    .userId("user-789")
    .build();
```

### Scope Hierarchy

```
SYSTEM
├── PROJECT → STORE
└── USER
```

Inheritance:
- **STORE** → PROJECT → SYSTEM
- **USER** → SYSTEM
- **PROJECT** → SYSTEM

## Automatic Template Loading

The SDK supports automatic loading of configuration templates from YAML files. Simply place your template files in a `templates` directory and the SDK will load them automatically.

### Quick Start

1. Create a `templates` directory in your project root
2. Add your YAML template files (`.yaml` or `.yml`)
3. Load templates on client initialization

```java
import com.dsai.scopeconfig.ConfigClient;
import com.dsai.scopeconfig.TemplateLoader;

try (ConfigClient client = ConfigClient.fromEnvironment().build()) {
    // Auto-load all templates from the templates directory
    TemplateLoader.loadFromDir(client, "./templates", "system");
    
    // Now use the client as normal
    Optional<String> value = client.getValue(identifier, "database.host",
            GetValueOptions.withInheritanceAndDefaults());
}
```

### Template File Format

Create YAML files in your `templates` directory following this structure:

```yaml
# templates/my-service.yaml
service:
  id: "my-service"
  label: "My Service"

groups:
  - id: "database"
    label: "Database Configuration"
    description: "Database connection settings"
    sortOrder: 100000
    fields:
      - path: "host"
        label: "Database Host"
        description: "The database server hostname"
        type: "STRING"
        defaultValue: "localhost"
        sortOrder: 100000
        displayOn:
          - "PROJECT"
          - "STORE"
      - path: "port"
        label: "Database Port"
        type: "INT"
        defaultValue: "5432"
        sortOrder: 200000
        displayOn:
          - "PROJECT"
      - path: "ssl-enabled"
        label: "Enable SSL"
        type: "BOOLEAN"
        defaultValue: "false"
        sortOrder: 300000
        displayOn:
          - "PROJECT"
```

### Field Types

| Type | Description | Example |
|------|-------------|---------|
| `STRING` | Text value | `"localhost"` |
| `INT` | Integer number | `"5432"` |
| `FLOAT` | Decimal number | `"0.7"` |
| `BOOLEAN` | True/false | `"true"` |
| `JSON` | JSON object/array | `'["a", "b"]'` |
| `ARRAY_STRING` | String array | |
| `SECRET` | Sensitive value | API keys, passwords |

### Display Scopes

The `displayOn` field controls which scopes the field is visible/editable:
- `SYSTEM` - System-wide settings
- `PROJECT` - Project-level settings
- `STORE` - Store-level settings
- `USER` - User-level settings

### Options (Dropdowns)

Define selectable options for a field:

```yaml
- path: "log-level"
  label: "Log Level"
  type: "STRING"
  defaultValue: "INFO"
  options:
    - value: "DEBUG"
      label: "Debug"
    - value: "INFO"
      label: "Info"
    - value: "WARN"
      label: "Warning"
    - value: "ERROR"
      label: "Error"
```

## Using TLS

```java
try (ConfigClient client = ConfigClient.builder()
        .host("config-service.example.com")
        .port(443)
        .tls(new File("/path/to/ca-cert.pem"))
        .build()) {
    // ...
}
```

## Error Handling

```java
try {
    ScopeConfig config = client.getConfig(identifier);
} catch (ConfigNotFoundException e) {
    System.err.println("Configuration not found: " + e.getMessage());
} catch (InvalidConfigException e) {
    System.err.println("Invalid configuration: " + e.getMessage());
} catch (ConfigServiceException e) {
    System.err.printf("Service error (%s): %s%n",
        e.getStatusCode(), e.getMessage());
}
```

## Examples

See the `src/main/java/com/dsai/scopeconfig/examples/` directory for complete working examples:

- `BasicUsage.java` - Comprehensive example demonstrating all SDK features

Run the example:

```bash
# Build the SDK first
mvn clean package

# Run the example
mvn exec:java -Dexec.mainClass="com.dsai.scopeconfig.examples.BasicUsage"
```

### Spring Boot Integration

For Spring Boot applications, you can create a configuration class:

```java
@Configuration
public class ScopeConfigConfiguration {

    @Bean
    public ConfigClient scopeConfigClient() {
        return ConfigClient.fromEnvironment()
                .cacheEnabled(true)
                .cacheTtl(Duration.ofMinutes(1))
                .backgroundSyncEnabled(true)
                .backgroundSyncInterval(Duration.ofSeconds(30))
                .build();
    }
}

@Service
public class MyService {
    private final ConfigClient configClient;

    public MyService(ConfigClient configClient) {
        this.configClient = configClient;
    }

    public String getDatabaseHost(String projectId) {
        ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
                .scope(Scope.PROJECT)
                .groupId("database")
                .projectId(projectId)
                .build();

        return configClient.getValue(identifier, "database.host",
                GetValueOptions.withInheritanceAndDefaults())
                .orElse("localhost");
    }
}
```

## Building & Publishing

### Building the JAR

```bash
# Build the JAR file (includes source and javadoc JARs)
mvn clean package

# The JARs will be in the target/ directory:
# - scopeconfig-sdk-1.0.0.jar (main JAR)
# - scopeconfig-sdk-1.0.0-sources.jar (source code)
# - scopeconfig-sdk-1.0.0-javadoc.jar (documentation)
```

### Publishing to GitHub Packages

**Prerequisites:**
- GitHub Personal Access Token with `write:packages` scope
- Access to the `digitalsolutionsai` organization

**Steps:**

**1. Configure Maven for publishing:**

Ensure your `~/.m2/settings.xml` has write credentials:

```xml
<settings>
  <servers>
    <server>
      <id>github</id>
      <username>YOUR_GITHUB_USERNAME</username>
      <password>YOUR_GITHUB_TOKEN</password>
    </server>
  </servers>
</settings>
```

**2. Generate proto files (if not already done):**

```bash
mkdir -p proto
cp -r ../../proto/config proto/
buf generate proto
```

**3. Publish:**

```bash
mvn clean deploy
```

The package will be published to: `com.dsai:scopeconfig-sdk:<version>`

**4. Verify:**

Visit: https://github.com/digitalsolutionsai/scope-config-service/packages

### Version Management

Update the version in `pom.xml` before publishing:

```bash
# For bug fixes
mvn versions:set -DnewVersion=1.0.1

# For new features
mvn versions:set -DnewVersion=1.1.0

# For breaking changes
mvn versions:set -DnewVersion=2.0.0
```

## Proto Generation (Development)

Generate the proto files using buf:

```bash
# Install buf (https://buf.build/docs/installation)

# Copy proto files
mkdir -p proto
cp -r ../../proto/config proto/

# Generate Java code
buf generate proto
```

## CI/CD Build (GitHub Actions)

The SDK is automatically built and published via GitHub Actions when a tag is pushed.

### Automated Build Workflow

The workflow is defined in `.github/workflows/build-sdks.yml` and triggers on tags matching `sdks/java/v*`.

**What the workflow does:**

1. Sets up Java 22 with GitHub Packages authentication
2. Copies proto files from the repository root
3. Generates Java code from proto files using buf
4. Extracts version from tag and updates `pom.xml`
5. Builds the JAR (including source and javadoc JARs)
6. Publishes to GitHub Packages
7. Creates a GitHub Release with the SDK artifact

### Triggering a Release

To release a new version of the Java SDK:

```bash
# Create and push a version tag
git tag sdks/java/v1.0.1
git push origin sdks/java/v1.0.1
```

This will:
- Build the SDK with version `1.0.1`
- Publish `com.dsai:scopeconfig-sdk:1.0.1` to GitHub Packages
- Create a GitHub Release with the SDK tarball

### CI/CD for Consuming Projects

In your project's CI/CD pipeline, configure Maven to authenticate with GitHub Packages:

```yaml
# GitHub Actions example
- name: Set up Java
  uses: actions/setup-java@v4
  with:
    distribution: 'temurin'
    java-version: '22'
    server-id: github
    settings-path: ${{ github.workspace }}

- name: Build with Maven
  run: mvn clean package -s $GITHUB_WORKSPACE/settings.xml
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## License

See the main project LICENSE file.
