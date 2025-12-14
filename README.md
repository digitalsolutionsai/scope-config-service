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