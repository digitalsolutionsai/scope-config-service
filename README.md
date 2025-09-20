# Scope Configuration Service

This is a gRPC service for managing and retrieving versioned configurations for your applications. It provides a flexible and scalable way to handle configurations for different services, projects, and environments.

## Features

- **Flexible Scoping**: Identify configurations using a flexible `scope` (SYSTEM, PROJECT, STORE, USER, GROUP) and a corresponding ID, allowing for granular control.
- **Versioned Configurations**: Every change to a configuration creates a new version, allowing you to track changes and roll back to previous versions.
- **Published Versions**: You can mark a specific version of a configuration as "published", which is what your client services will consume by default.
- **gRPC Interface**: The service uses a gRPC interface for high-performance, language-agnostic communication.
- **Command-Line Interface (CLI)**: A CLI (`config-cli`) is provided for easy interaction with the service.

## Getting Started

### Prerequisites

- Go 1.18 or later
- Docker and Docker Compose
- For API changes: `buf`, `protoc-gen-go`, and `protoc-gen-go-grpc`

### Building the Service

You can build the gRPC server and the command-line interface (CLI) using the provided `Makefile`:

```bash
# Build the server
make build-server

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

To run the service and its PostgreSQL database, use Docker Compose.

First, create a `.env` file:

```bash
cp .env.example .env
```

Then, run both services:

```bash
docker compose -f compose.postgres.yml -f compose.yml up -d --build
```

## User Guide: Managing Configurations

This guide walks you through managing configurations using the `config-cli`. All commands require the `--service-name` flag.

### 1. Set Configuration Values

The `set` command creates a new, unpublished version of a configuration with the specified key-value pairs.

In this example, we'''ll create a configuration for a service named `billing-service` within the `project-123` project and the `stripe` group.

```bash
docker compose exec config-service config-cli set \
    --service-name=billing-service \
    --scope=PROJECT \
    --project-id=project-123 \
    --group-id=stripe \
    --user-name="John Doe" \
    stripe.apiKey=sk_test_... \
    stripe.apiVersion=2023-10-16
```

This creates a new version of the configuration. It is not yet published.

### 2. View Latest and Published Configurations

The `show` command displays a summary of the latest and published configurations, allowing you to review changes before publishing them.

```bash
docker compose exec config-service config-cli show \
    --service-name=billing-service \
    --scope=PROJECT \
    --project-id=project-123 \
    --group-id=stripe
```

### 3. Get a Specific Configuration Version

The `get` command retrieves the full details of a configuration in a readable format.
- By default, it fetches the **published** version.
- Use `--latest` to get the most recent (possibly unpublished) version.
- Use `--version` to get a specific version number.
- Use `--path` to get a single key from the configuration.

To see the changes you just made, use `get --latest`:

```bash
docker compose exec config-service config-cli get --latest \
    --service-name=billing-service \
    --scope=PROJECT \
    --project-id=project-123 \
    --group-id=stripe
```

To get a single key:

```bash
docker compose exec config-service config-cli get --latest \
    --service-name=billing-service \
    --scope=PROJECT \
    --project-id=project-123 \
    --group-id=stripe \
    --path=stripe.apiKey
```

This will return the value of the key, or `null` if it doesn'''t exist.

### 4. Publish a New Configuration

The `publish` command makes a specific version the "published" one. Clients requesting the configuration without specifying a version will receive this one.

Let'''s say the `set` command created version `2`. We can now publish it:

```bash
docker compose exec config-service config-cli publish 2 \
    --service-name=billing-service \
    --scope=PROJECT \
    --project-id=project-123 \
    --group-id=stripe \
    --user-name="John Doe"
```

Now, running `get` without any flags will show that version 2 is the published version.

### 5. Show Version History

To see the history of changes for a configuration, use `show --history`.

```bash
docker compose exec config-service config-cli show --history \
    --service-name=billing-service \
    --scope=PROJECT \
    --project-id=project-123 \
    --group-id=stripe
```

This will display a table of all versions, who created them, and when. By default, it shows the last 100 versions. You can change this with the `--limit` flag:

```bash
docker compose exec config-service config-cli show --history --limit=5 \
    --service-name=billing-service \
    --scope=PROJECT \
    --project-id=project-123 \
    --group-id=stripe
```
