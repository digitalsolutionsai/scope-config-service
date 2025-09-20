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

This project uses Protocol Buffers for its gRPC API. The generated Go code for the protobuf client (`*.pb.go`) is committed directly to the repository. This is standard practice in Go projects, as it means consumers of your repository do not need to install any tools to build and run the project.

You only need to regenerate the client code if you make a change to the API definition in `proto/config/v1/config.proto`.

**1. Install Generation Tools (First-Time Setup):**

If you don't have them, install the necessary Go plugins for protobuf generation:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**2. Regenerate the Client:**

After modifying the `.proto` file, run the following command to regenerate the Go client:

```bash
make proto
```

This will run `buf generate` and update the necessary `*.pb.go` files, which you should then commit along with your changes to the `.proto` file.

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

## User Guide: Managing Configurations

This guide will walk you through the common workflow of creating, viewing, and publishing configurations using the CLI.

### 1. Create a New, Unpublished Configuration

The `set` command creates a new version of a configuration. If you provide new key-value pairs, they will be added to the configuration.

In this example, we'''ll create a new configuration for a service named `test-service` in the `test` project. This will create version 2 of the configuration, but it will not be published yet.

```bash
docker compose -f compose.yml exec config-service config set --service=test-service --scope=SYSTEM --project=test db.host=postgres db.port=5432 --user=gemini
```

### 2. View the Latest, Unpublished Configuration

The `show` command displays the latest version of a configuration, including unpublished changes. This is useful for reviewing a configuration before publishing it.

```bash
docker compose -f compose.yml exec config-service config show --service=test-service --scope=SYSTEM --project=test
```

You should see an output similar to this:

```
Latest Version: 2
Published Version: 1
Fields:
  db.host: postgres
  db.port: 5432
```

### 3. Publish the New Configuration

The `publish` command makes a specific version of a configuration the "published" version. This is the version that client services will consume when they request the configuration.

```bash
docker compose -f compose.yml exec config-service config publish 2 --service=test-service --scope=SYSTEM --project=test --user=gemini
```

### 4. Get the Published Configuration

The `get` command retrieves a single value from the *published* configuration.

```bash
docker compose -f compose.yml exec config-service config get db.host --service=test-service --scope=SYSTEM --project=test
```

This will return the value of the `db.host` key, which is `postgres`.

To view the entire published configuration, you can use the `show` command again. Since version 2 is now published, the output will reflect this:

```bash
docker compose -f compose.yml exec config-service config show --service=test-service --scope=SYSTEM --project=test
```

```
Latest Version: 2
Published Version: 2
Fields:
  db.host: postgres
  db.port: 5432
```