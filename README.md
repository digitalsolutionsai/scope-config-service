# Scope Configuration Service

This is a gRPC service for managing and retrieving versioned, schema-driven configurations for your applications. It provides a flexible and scalable way to handle configurations for different services, projects, and environments.

-----

## Core Concepts

Before using the service, it's helpful to understand two key concepts:

1.  **Templates (The Schema):** A template defines the *structure* of your configuration. It specifies all the possible keys (`path`), their data types, default values, and descriptive text for UIs (`label`, `description`). Templates are defined in YAML files and applied to the service once.

2.  **Configurations (The Values):** A configuration is an *instance* of a template. It stores the actual values for a specific scope (e.g., for `project-123`). When you request a configuration, the service uses the template to provide default values for any keys that haven't been explicitly set.

-----

## Features

  - **Schema-Driven Configurations**: Define a clear schema for your configurations using YAML templates, complete with types, default values, and descriptions.
  - **Flexible Scoping**: Apply configurations at different levels: `SYSTEM`, `PROJECT`, `STORE`, or `USER`.
  - **Versioned Configurations**: Every change to a configuration creates a new, auditable version, allowing you to track changes and roll back if needed.
  - **Published Versions**: Mark a specific version as "published" to ensure stability for client consumption, while still being able to work on a newer, unpublished version.
  - **gRPC Interface**: A high-performance, language-agnostic gRPC interface.
  - **HTTP REST API Gateway**: A lightweight HTTP/JSON wrapper for easy frontend integration. See [HTTP Gateway Documentation](docs/HTTP_GATEWAY.md).
  - **Swagger UI**: Interactive API documentation at `/swagger/index.html` for easy API exploration and testing.
  - **Command-Line Interface (CLI)**: A powerful CLI (`config-cli`) for easy interaction with the service.

-----

## Getting Started

### Prerequisites

- Go 1.18 or later
- Docker and Docker Compose
- For API changes: `buf`, `protoc-gen-go`, and `protoc-gen-go-grpc`

### Building the Service

You can build the gRPC server, HTTP gateway, and the command-line interface (CLI) using the provided `Makefile`:

```bash
# Build the gRPC server
make build-server

# Build the HTTP gateway
make build-httpgateway

# Build the CLI
make build-cli
```

The generated binaries will be placed in the `bin/` directory.

### Protobuf & API Changes

This project uses Protocol Buffers for its gRPC API. If you change the API definition in `proto/config/v1/config.proto`, you must regenerate the Go client code:

**1. Install Generation Tools (First-Time Setup):**

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**2. Regenerate the Client:**

```bash
make proto
```

This will run `buf generate` and update the necessary `*.pb.go` files, which you should then commit.

### SDK Development & Publishing

This project provides client SDKs in multiple languages located in the `sdks/` directory. When you update the proto files, you need to regenerate the SDK code and publish new versions.

#### Available SDKs

- **TypeScript** (`sdks/typescript/`) - Published to GitHub Packages as `@digitalsolutionsai/scopeconfig`
- **Go** (`sdks/go/`) - Go module
- **Python** (`sdks/python/`) - Python package
- **Java** (`sdks/java/`) - Maven package

#### Updating SDKs After Proto Changes

**1. Regenerate Proto Code for All SDKs:**

```bash
# From project root
make proto

# Or regenerate for specific SDK
cd sdks/typescript
npm run generate  # Runs buf generate
```

**2. Update SDK Version:**

```bash
cd sdks/typescript
npm version patch  # For bug fixes
npm version minor  # For new features
npm version major  # For breaking changes
```

**3. Build and Test:**

```bash
npm run build
npm test  # If tests are available
```

#### Publishing TypeScript SDK to GitHub Packages

**Prerequisites:**
- GitHub Personal Access Token with `write:packages` scope
- Access to the `digitalsolutionsai` organization

**Steps:**

**1. Login to GitHub Packages:**

```bash
cd sdks/typescript
npm login --registry=https://npm.pkg.github.com
# Username: your-github-username
# Password: <paste your GitHub token>
# Email: your-email@example.com
```

**2. Publish:**

```bash
npm publish
```

The package will be published to: `@digitalsolutionsai/scopeconfig@<version>`

**3. Verify:**

Visit: https://github.com/digitalsolutionsai/scope-config-service/packages

#### Using the TypeScript SDK

For detailed usage instructions, see [`sdks/typescript/README.md`](sdks/typescript/README.md).

**Quick Install:**

```bash
# Configure npm to use GitHub Packages
echo "@digitalsolutionsai:registry=https://npm.pkg.github.com" >> .npmrc

# Install
npm install @digitalsolutionsai/scopeconfig
```

**Quick Usage:**

```typescript
import { ConfigClient, createOptionsFromEnv, createIdentifier, Scope } from '@digitalsolutionsai/scopeconfig';

const client = new ConfigClient(createOptionsFromEnv());
await client.connect();

const identifier = createIdentifier('my-service')
  .withScope(Scope.PROJECT)
  .withGroupId('database')
  .withProjectId('proj-123')
  .build();

const value = await client.getValue(identifier, 'database.host', {
  useDefault: true,
  inherit: true,
});
```

#### Publishing Java SDK to GitHub Packages

**Prerequisites:**
- GitHub Personal Access Token with `write:packages` scope
- Access to the `digitalsolutionsai` organization

**Steps:**

**1. Configure Maven credentials in `~/.m2/settings.xml`:**

```xml
<settings>
  <servers>
    <server>
      <id>github</id>
      <username>your-github-username</username>
      <password>YOUR_GITHUB_TOKEN</password>
    </server>
  </servers>
</settings>
```

**2. Generate proto files and publish:**

```bash
cd sdks/java
mkdir -p proto && cp -r ../../proto/config proto/
buf generate proto
mvn clean deploy
```

The package will be published to: `vn.dsai:scopeconfig-sdk:<version>`

**3. Verify:**

Visit: https://github.com/digitalsolutionsai/scope-config-service/packages

#### Using the Java SDK

For detailed usage instructions, see [`sdks/java/README.md`](sdks/java/README.md).

**Quick Install (Maven):**

Add to your `pom.xml`:

```xml
<repositories>
    <repository>
        <id>github</id>
        <url>https://maven.pkg.github.com/digitalsolutionsai/scope-config-service</url>
    </repository>
</repositories>

<dependencies>
    <dependency>
        <groupId>vn.dsai</groupId>
        <artifactId>scopeconfig-sdk</artifactId>
        <version>1.0.0</version>
    </dependency>
</dependencies>
```

**Quick Usage:**

```java
import vn.dsai.scopeconfig.*;
import vn.dsai.config.v1.*;

// Create client using environment variables
try (ConfigClient client = ConfigClient.fromEnvironment().build()) {
    ConfigIdentifier identifier = ConfigIdentifierBuilder.create("my-service")
            .scope(Scope.PROJECT)
            .groupId("database")
            .projectId("proj-123")
            .build();

    Optional<String> value = client.getValue(identifier, "database.host",
            GetValueOptions.withInheritanceAndDefaults());
    
    value.ifPresent(v -> System.out.println("Database host: " + v));
}
```

### Running with Docker Compose

To run the complete stack (PostgreSQL database and config service with both gRPC and HTTP), use Docker Compose.

First, create a `.env` file:

```bash
cp .env.example .env
```

Then, run all services:

```bash
docker compose -f compose.postgres.yml -f compose.yml up -d --build
```

The config service runs both gRPC and HTTP in a single container and will be available at:
- **gRPC Service**: `localhost:50051`
- **HTTP Gateway**: `http://localhost:8080`
- **Swagger UI**: `http://localhost:8080/swagger/index.html`

For detailed HTTP API documentation and examples, see the [HTTP Gateway Documentation](docs/HTTP_GATEWAY.md).

### Exploring the API with Swagger UI

Once the service is running, you can explore and test the HTTP API interactively:

1. Open your browser and navigate to `http://localhost:8080/swagger/index.html`
2. Browse through available endpoints with detailed parameter descriptions
3. Try out API calls directly from the browser
4. View request/response schemas and examples

-----

## User Guide: A Typical Workflow

This guide walks you through the complete lifecycle of a configuration using the `config-cli`.

### Step 1: Define and Apply a Template (Admin Task)

First, define the schema for your configuration in a YAML file. This is typically done once when setting up a new service.

**Example `payment-template.yaml`:**

```yaml
service:
  id: "payment"
  label: "Payment Service"

groups:
  - id: "payment-methods"
    label: "Payment Methods"
    description: "Configuration for available payment methods."
    fields:
      - path: "methods.credit-card"
        label: "Credit Card"
        description: "Credit card payment method"
        type: "BOOLEAN"
        defaultValue: "true"
        displayOn:
          - "SYSTEM"
          - "USER"
      - path: "methods.paypal"
        label: "PayPal"
        description: "PayPal payment method"
        type: "BOOLEAN"
        defaultValue: "true"
        displayOn:
          - "SYSTEM"
          - "USER"
      - path: "methods.stripe"
        label: "Stripe"
        description: "Stripe payment method"
        type: "BOOLEAN"
        defaultValue: "false"
        displayOn:
          - "SYSTEM"
          - "USER"

  - id: "server-config"
    label: "Server Configuration"
    description: "Payment gateway server settings and API configurations."
    fields:
      - path: "api.timeout"
        label: "API Timeout"
        description: "Timeout for payment API calls in seconds"
        type: "INT"
        defaultValue: "30"
        displayOn:
          - "SYSTEM"
          - "PROJECT"
```

Next, use the `template apply` command to upload this schema to the service.

```bash
# Make sure your template file is accessible inside the container or use `docker cp`
docker compose exec config-service config-cli template apply -f /path/to/payment-template.yaml
```

### Step 2: Set Configuration Values

Now that a template exists, you can set values for a specific scope. The `set` command creates a new, unpublished version of a configuration.

```bash
docker compose exec config-service config-cli set \
    --service-name=billing-service \
    --scope=PROJECT \
    --project-id=project-123 \
    --group-id=stripe \
    --user-name="John Doe" \
    stripe.apiKey=sk_test_...
```

### Step 3: Get a Specific Configuration

The `get` command retrieves the configuration. It will merge the values you explicitly set with the default values from the template.

  - By default, it fetches the **published** version.
  - Use `--latest` to get the most recent (possibly unpublished) version.

<!-- end list -->

```bash
docker compose exec config-service config-cli get --latest \
    --service-name=billing-service \
    --scope=PROJECT \
    --project-id=project-123 \
    --group-id=stripe
```

### Step 4: Publish a New Configuration

The `publish` command makes a specific version the "live" one.

Let's say the `set` command created version `1`. We can now publish it:

```bash
docker compose exec config-service config-cli publish 1 \
    --service-name=billing-service \
    --scope=PROJECT \
    --project-id=project-123 \
    --group-id=stripe \
    --user-name="John Doe"
```

### Step 5: Show Version History

To see the audit log of changes, use the dedicated `history` command. This displays a clean record of when each version was created and by whom.

```bash
docker compose exec config-service config-cli history \
    --service-name=billing-service \
    --scope=PROJECT \
    --project-id=project-123 \
    --group-id=stripe
```