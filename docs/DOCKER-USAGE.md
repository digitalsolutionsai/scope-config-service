# Scope Configuration Service

The **Scope Configuration Service** is a high-performance, centralized, and schema-driven configuration management system designed for modern microservices architectures. 

It enables teams to manage application configurations as a single source of truth, with support for flexible scoping, version control, and multi-protocol access (gRPC and HTTP/REST).

## Key Features

- 🏗️ **Schema-Driven (Templates)**: Define your configuration structure using YAML templates with types, default values, and validation rules.
- 🎯 **Flexible Scoping**: Apply configurations at different levels: `SYSTEM`, `PROJECT`, `STORE`, or `USER`.
- 📜 **Version Control & Audit**: Every change creates a new, immutable version. Track who changed what and when.
- 🚀 **Stable Publishing**: Mark specific versions as "published" for production consumption while continuing to work on newer drafts.
- ⚡ **Multi-Protocol Support**:
    - **gRPC**: High-performance API for service-to-service communication.
    - **HTTP/REST Gateway**: JSON-based API for easy frontend and legacy system integration.
    - **Admin UI**: Built-in web interface for managing configuration templates visually.
    - **Swagger UI**: Interactive API documentation built-in.
- 🛠️ **CLI Tooling**: Powerful command-line interface for administrators and developers.

## Core Concepts

### 1. Templates (The Schema)
A template defines the *structure* of your configuration. It specifies fields, data types (`STRING`, `INT`, `BOOLEAN`, etc.), and default values.

### 2. Configurations (The Values)
A configuration is an *instance* of a template for a specific scope (e.g., a specific project). The service automatically merges explicit values with template defaults.

## Quick Start

The service supports two database backends: **SQLite** (built-in, zero setup) and **PostgreSQL** (for high availability).

### One-Line Docker Run

```bash
docker run -d -p 50051:50051 -p 8080:8080 -v config_data:/app/data dsailoivo/scope-config:0.1.2
```

This starts the service with SQLite (default), exposes gRPC on port `50051` and the HTTP gateway on port `8080`, and persists data to a named volume.

To enable Basic Auth, pass credentials via environment variables:

```bash
docker run -d -p 50051:50051 -p 8080:8080 -v config_data:/app/data \
  -e AUTH_USER=admin -e AUTH_PASSWORD=secret \
  dsailoivo/scope-config:0.1.2
```

To connect to PostgreSQL instead of SQLite:

```bash
docker run -d -p 50051:50051 -p 8080:8080 \
  -e DATABASE_URL=postgresql://user:password@host:5432/config_db?sslmode=disable \
  dsailoivo/scope-config:0.1.2
```

### Mode 1: SQLite (Built-in Default)
Run the service instantly without any external database dependencies. All data is persisted to a local volume.

```yaml
# compose.yml
services:
  config-service:
    image: dsailoivo/scope-config:0.1.0  # Or latest
    ports:
      - "50051:50051" # gRPC
      - "8080:8080"   # HTTP Gateway
    volumes:
      - config_data:/app/data

volumes:
  config_data:
```

### Mode 2: PostgreSQL
Connect to an external PostgreSQL database by providing the `DATABASE_URL` environment variable.

```yaml
# compose.yml
services:
  config-service:
    image: dsailoivo/scope-config:latest
    ports:
      - "50051:50051" # gRPC
      - "8080:8080"   # HTTP Gateway
    environment:
      - DATABASE_URL=postgresql://user:password@postgres:5432/config_db?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:17-alpine
    environment:
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=config_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```
## API Documentation

Once running, you can access:
- **Admin UI**: `http://localhost:8080/admin`
- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **gRPC API**: `localhost:50051`

## GitHub Repository

For full documentation, SDKs (TypeScript, Go, Java, Python), and contribution guidelines, visit our GitHub repository:
[digitalsolutionsai/scope-config-service](https://github.com/digitalsolutionsai/scope-config-service)
