# Scope Configuration Service

This is a gRPC service for managing and retrieving versioned configurations for your applications. It provides a flexible and scalable way to handle configurations for different services, projects, and environments.

## Features

- **Versioned Configurations**: Every change to a configuration creates a new version, allowing you to track changes and roll back to previous versions.
- **Published Versions**: You can mark a specific version of a configuration as "published", which is what your client services will consume.
- **gRPC Interface**: The service uses a gRPC interface for high-performance, language-agnostic communication.
- **Command-Line Interface (CLI)**: A CLI is provided for easy interaction with the service.

## Getting Started

### Prerequisites

- Go 1.18 or later
- Protocol Buffers (protoc)
- Docker and Docker Compose

### Building the Service

You can build the gRPC server and the command-line interface (CLI) using the following commands:

```bash
# Build the server
go build -o bin/server ./cmd/server

# Build the CLI
go build -o bin/config ./cmd/cli
```

### Running with Docker Compose

To run the service and the PostgreSQL database together, you can use Docker Compose. There are two files provided:

- `compose.postgres.yml`: Defines the PostgreSQL service.
- `compose.yml`: Defines the configuration service.

Before running the service, you need to create a `.env` file from the `.env.example` file and update the values if necessary:

```bash
cp .env.example .env
```

To run both services, use the following command:

```bash
docker compose -f compose.postgres.yml -f compose.yml up -d --build
```

### Using the CLI

The CLI provides several commands for interacting with the service. Here are a few examples:

*   **Get a configuration**:

    ```bash
    ./bin/config get --service=config-service
    ```

*   **Set a configuration**:

    ```bash
    ./bin/config set database.host localhost --service=config-service
    ```

### Advanced CLI Examples

Once the service is running in Docker, you can use `docker exec` to run commands with the built-in `config` CLI. This allows you to manage configurations for different projects and services.

*   **Set a Gemini API Key for a Translation Service**

    This command sets the `GEMINI_API_KEY` for a service named `translate` within the `test` project, using `0` as a unique store identifier.

    ```bash
    docker exec -it config_service config set "GEMINI_API_KEY" "your-secret-gemini-api-key" \
      --project="test" \
      --service="translate" \
      --store="0"
    ```

*   **Set PayPal Merchant Configuration for a Payment Service**

    This example shows how to set multiple keys for a PayPal merchant configuration under the `payment` service in an `e-commerce` project.

    ```bash
    # Set the Client ID
    docker exec -it config_service config set "paypal.client_id" "your-paypal-client-id" \
      --project="e-commerce" \
      --service="payment" \
      --store="merchant"

    # Set the Client Secret
    docker exec -it config_service config set "paypal.client_secret" "your-paypal-client-secret" \
      --project="e-commerce" \
      --service="payment" \
      --store="merchant"
    ```
