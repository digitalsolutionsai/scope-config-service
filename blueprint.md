# System Design Document: ScopeConfig Service

- **Version:** 1.1
    
- **Last Updated:** September 14, 2025
    
- **Author:** Gemini
    
- **Status:** Draft
    

## 1\. Overview and Project Purpose

### 1.1. Project Name

**ScopeConfig Service**

### 1.2. Purpose

The **ScopeConfig Service** is a centralized, high-performance microservice responsible for providing a versioned and scoped configuration management system for all other services within the ecosystem.

This project solves the following challenges:

- **Eliminate Distributed Configuration:** Instead of storing configuration files (`.yml`, `.json`, `.env`) in each service, all configurations will be managed in a single, central location.
    
- **Version Management:** Easily track change history, publish new versions, or roll back to a previous version in case of issues.
    
- **Scoped Configuration:** Provide different sets of configurations based on the environment (`production`, `staging`), service, project, or any other defined context.
    
- **Consistency and Safety:** Ensure that services always receive the correct and most recently "published" configuration.
    

### 1.3. Target Technology Stack

- **Language:** Go (Golang)
    
- **Communication:** gRPC (using Protocol Buffers 3)
    
- **Database:** PostgreSQL
    
- **Caching:** In-memory cache (implemented within the service)
    
- **Deployment:** Docker
    

## 2\. System Architecture Design

### 2.1. Design Principles

- **gRPC-First:** All interactions with the service will be through gRPC. A RESTful/HTTP interface, if needed, will be generated via a gRPC-Gateway.
    
- **Stateless Service:** The service will not store any request state. All state resides in the database, making it easy to scale out (add more instances).
    
- **Clean Architecture:** A clear separation of layers: Interface (gRPC), Business Logic (Service/Usecase), and Data Access (Repository).
    
- **High Performance:** Optimized for configuration read operations by using an in-memory cache.
    

### 2.2. Architecture Diagram

```
+--------------------------+         +----------------------------+
|   Client Microservice A  | ------> |                            |
+--------------------------+  gRPC   |                            |
                                    |     ScopeConfig Service    |         +-------------------+
+--------------------------+         |        (Golang)            | ------> |    PostgreSQL     |
|   Client Microservice B  | ------> |                            |         |     Database      |
+--------------------------+         |    +------------------+    |         +-------------------+
                                    |    | In-Memory Cache  |    |
                                    |    +------------------+    |
+--------------------------+         |                            |
|   Admin/Dev Tool (CLI)   | ------> |                            |
+--------------------------+         +----------------------------+
```

### 2.3. Main Workflows

#### Get Configuration Flow

1.  A **Client Service** sends a `GetConfig` gRPC request to the **ScopeConfig Service**, including a `ConfigIdentifier` (serviceName, projectId, etc.).
    
2.  The **ScopeConfig Service** first generates a *cache key* from the `ConfigIdentifier`.
    
3.  The service checks the **In-Memory Cache**:
    
    - **Cache Hit:** If the configuration is found in the cache, it is returned immediately to the client.
        
    - **Cache Miss:** If not found: a. The service queries the **PostgreSQL Database** to retrieve the published configuration version (or the latest version if none is published). b. The service aggregates the data and creates a `ScopeConfig` object. c. The service stores the `ScopeConfig` object in the **In-Memory Cache** with the generated cache key. d. The service returns the result to the client.
        

#### Update/Publish Configuration Flow

1.  An **Admin/Tool** sends an `UpdateConfig` or `PublishVersion` gRPC request.
    
2.  The **ScopeConfig Service** performs the write operations (INSERT/UPDATE) on the **PostgreSQL Database**.
    
3.  **Important:** After a successful write, the service **must invalidate (delete)** the corresponding key in the **In-Memory Cache**. The next read for this configuration will be a cache-miss, ensuring the data is always fresh.
    

## 3\. API Design (gRPC Proto)

The `proto/config/v1/config.proto` file will define the entire service interface. Compared to the original proto file, this version is improved for clarity and adherence to gRPC best practices.

```
syntax = "proto3";

package vn.dsai.config.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

option go_package = "[github.com/vn-dsai/scope-config-service/gen/go/config/v1;configv1](https://github.com/vn-dsai/scope-config-service/gen/go/config/v1;configv1)";

// ConfigService is the main service for managing and retrieving configurations.
service ConfigService {
  // Retrieves the configuration for the published version or the latest version.
  // This is the most frequently used RPC by client services.
  rpc GetConfig(GetConfigRequest) returns (ScopeConfig);

  // Retrieves a configuration by a specific version number.
  rpc GetConfigByVersion(GetConfigByVersionRequest) returns (ScopeConfig);

  // Retrieves the version history for a configuration.
  rpc GetConfigHistory(GetConfigHistoryRequest) returns (GetConfigHistoryResponse);

  // Updates or creates a new configuration set.
  // This action creates a new version (latest_version + 1).
  rpc UpdateConfig(UpdateConfigRequest) returns (ScopeConfig);

  // Marks a specific version as "published" for client consumption.
  rpc PublishVersion(PublishVersionRequest) returns (ConfigVersion);

  // Deletes a configuration set and all of its associated versions.
  rpc DeleteConfig(DeleteConfigRequest) returns (google.protobuf.Empty);
}

// === ENUMS ===

enum Scope {
  SCOPE_UNSPECIFIED = 0;
  DEFAULT = 1;
  SYSTEM = 2;
  SERVICE = 3;
  PROJECT = 4;
  STORE = 5;
}

enum FieldType {
  FIELD_TYPE_UNSPECIFIED = 0;
  STRING = 1;
  INT = 2;
  FLOAT = 3;
  BOOLEAN = 4;
  JSON = 5;
  ARRAY_STRING = 6; // Example for an array type
}

// === MODELS ===

// A unique identifier for a configuration set.
message ConfigIdentifier {
  string service_name = 1; // Name of the service owning the config (e.g., "payment-service")
  string project_id = 2;   // (Optional) Project ID
  string store_id = 3;     // (Optional) Store ID
  string group_id = 4;     // (Optional) Config group (e.g., "database", "api-keys")
  Scope scope = 5;         // The scope of the configuration
}

// Represents a configuration version.
message ConfigVersion {
  int32 id = 1;
  ConfigIdentifier identifier = 2;
  int32 latest_version = 3;
  int32 published_version = 4;
  google.protobuf.Timestamp created_at = 5;
  string created_by = 6;
  google.protobuf.Timestamp updated_at = 7;
  string updated_by = 8;
}

// A specific configuration field.
message ConfigField {
  string path = 1;          // The key of the config (e.g., "database.postgres.connection_string")
  string value = 2;         // The value, stored as a string
  FieldType type = 3;       // Data type to help clients parse the value
  string default_value = 4; // (Optional)
  string description = 5;   // (Optional)
}

// Represents a complete configuration set at a specific version.
message ScopeConfig {
  ConfigVersion version_info = 1;
  int32 current_version = 2; // The version number of the accompanying fields
  repeated ConfigField fields = 3;
}

// === REQUESTS & RESPONSES ===

message GetConfigRequest {
  ConfigIdentifier identifier = 1;
}

message GetConfigByVersionRequest {
  ConfigIdentifier identifier = 1;
  int32 version = 2;
}

message UpdateConfigRequest {
  ConfigIdentifier identifier = 1;
  repeated ConfigField fields = 2;
  string user = 3; // The user making the change
}

message PublishVersionRequest {
  ConfigIdentifier identifier = 1;
  int32 version_to_publish = 2;
  string user = 3;
}

message DeleteConfigRequest {
  ConfigIdentifier identifier = 1;
}

message GetConfigHistoryRequest {
  ConfigIdentifier identifier = 1;
}

message GetConfigHistoryResponse {
  repeated ConfigVersion versions = 1;
}
```

## 4\. Data Model (Database Schema)

Based on the existing Liquibase schema, we will maintain the two primary tables.

#### Table: `config_versions`

Stores information about each configuration set and its versions.

| Column Name | Data Type | Constraints | Notes |
| --- | --- | --- | --- |
| `id` | `SERIAL` | `PRIMARY KEY` | Auto-incrementing ID |
| `service_name` | `VARCHAR(50)` | `NOT NULL` | Part of the unique key |
| `project_id` | `VARCHAR(50)` |     |     |
| `store_id` | `VARCHAR(50)` |     |     |
| `group_id` | `VARCHAR(50)` |     |     |
| `scope` | `VARCHAR(20)` | `NOT NULL` |     |
| `latest_version` | `INTEGER` | `NOT NULL, DEFAULT 0` | The latest version, auto-incremented |
| `published_version` | `INTEGER` |     | The version currently published to clients |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL, DEFAULT NOW()` |     |
| `created_by` | `VARCHAR(50)` |     |     |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL, DEFAULT NOW()` |     |
| `updated_by` | `VARCHAR(50)` |     |     |
|     |     | `UNIQUE` (service_name, project_id, store_id, group_id, scope) | Ensures each config set is unique |

#### Table: `config_fields`

Stores each key-value pair for a specific configuration version.

| Column Name | Data Type | Constraints | Notes |
| --- | --- | --- | --- |
| `id` | `SERIAL` | `PRIMARY KEY` | Auto-incrementing ID |
| `config_version_id` | `INTEGER` | `NOT NULL, FOREIGN KEY` | Links to `config_versions.id` |
| `version` | `INTEGER` | `NOT NULL` | The version this field belongs to |
| `path` | `VARCHAR(255)` | `NOT NULL` | The configuration key |
| `value` | `TEXT` | `NOT NULL` | The configuration value |
| `type` | `VARCHAR(20)` | `NOT NULL` | Data type (STRING, INT, JSON, etc.) |
| `default_value` | `TEXT` |     |     |
| `description` | `TEXT` |     |     |
| `is_active` | `BOOLEAN` | `DEFAULT true` |     |
|     |     | `UNIQUE` (config_version_id, version, path) | Ensures each key is unique within a version |

## 5\. Caching Strategy

- **Objective:** Minimize database access for configuration read requests.
    
- **Cache Type:** In-memory, LRU (Least Recently Used) to prevent unbounded cache growth.
    
- **Recommended Library:** Start with a simple `map` and `sync.RWMutex`. For more complex needs (like TTL, size limits), consider using [ristretto](https://github.com/dgraph-io/ristretto "null").
    
- **Cache Key:** A string generated from the `ConfigIdentifier`.
    
    - **Format:** `fmt.Sprintf("%s:%s:%s:%s:%s", id.ServiceName, id.ProjectId, id.StoreId, id.GroupId, id.Scope)`
- **Cache Value:** The fully processed `ScopeConfig` object.
    
- **Invalidation Mechanism:**
    
    - After a successful call to `UpdateConfig`, `PublishVersion`, or `DeleteConfig`, the service **MUST** delete the corresponding cache key.
        
    - This ensures that the next `GetConfig` call will have to re-read from the database and cache the latest data.
        

## 6\. Project Structure (Go)

We will follow a standard Go project structure for easier management and maintenance.

```
scope-config-service/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── configs/
│   └── config.yml               # Service configuration file (DB conn, port,...)
├── db/                          #<-- NEW: Database migration files
│   └── migrations/
│       ├── 000001_init_schema.up.sql
│       └── 000001_init_schema.down.sql
├── internal/
│   ├── app/                     # Application layer, orchestrates logic
│   │   └── grpc/
│   │       └── server.go        # gRPC server implementation and RPC handlers
│   ├── domain/                  # Contains core business models and entities
│   │   ├── config.go
│   │   └── repository.go      # Interfaces for repositories
│   ├── infrastructure/
│   │   ├── cache/
│   │   │   └── memory.go        # In-memory cache implementation
│   │   └── persistence/
│   │       └── postgres.go      # Repository implementation for DB interaction
├── gen/                         # Contains generated Go code from proto files
│   └── go/
│       └── config/
│           └── v1/
│               └── config.pb.go
├── proto/                       # Contains the .proto source files
│   └── config/
│       └── v1/
│           └── config.proto
├── go.mod
├── go.sum
└── Dockerfile
```

## 7\. Database Migration Strategy

### 7.1. Tooling

We will use **`golang-migrate/migrate`**, the de-facto standard for database migrations in the Go ecosystem.

- **Migrations are written in plain SQL**, providing explicit control over the schema.
    
- **State is tracked** in a `schema_migrations` table within the database to prevent duplicate runs.
    

### 7.2. Workflow

The chosen strategy is to **run migrations programmatically on application startup**. This ensures that the deployed application instance is always in sync with the required database schema, simplifying the deployment process.

### 7.3. Implementation

1.  **Migration Files:** All SQL migration scripts will be located in the `db/migrations` directory. Each migration consists of an `up` and a `down` file (e.g., `000001_init_schema.up.sql`).
    
2.  **Startup Logic:** The migration logic will be executed in the `main()` function of the application entry point (`cmd/server/main.go`) before the gRPC server is started.
    
3.  **Code Example:** The application will use the `golang-migrate/migrate` library to connect to the database and apply any pending migrations found in the migrations directory.
    
    ```
    // Example from cmd/server/main.go
    
    import (
        "log"
        "[github.com/golang-migrate/migrate/v4](https://github.com/golang-migrate/migrate/v4)"
        _ "[github.com/golang-migrate/migrate/v4/database/postgres](https://github.com/golang-migrate/migrate/v4/database/postgres)"
        _ "[github.com/golang-migrate/migrate/v4/source/file](https://github.com/golang-migrate/migrate/v4/source/file)"
    )
    
    func main() {
        // 1. Load configuration (DB URL, migration path)
        dbURL := "postgres://user:password@host:port/dbname?sslmode=disable"
        migrationsPath := "file://db/migrations"
    
        // 2. Run migrations before starting the app
        runMigrations(dbURL, migrationsPath)
    
        // 3. Continue with application startup
        log.Println("Successfully applied migrations. Starting server...")
        // ... initialize DB pool, gRPC server, etc.
    }
    
    func runMigrations(databaseURL string, migrationsPath string) {
        m, err := migrate.New(migrationsPath, databaseURL)
        if err != nil {
            log.Fatalf("Failed to create migrate instance: %v", err)
        }
    
        if err := m.Up(); err != nil && err != migrate.ErrNoChange {
            log.Fatalf("Failed to apply migrations: %v", err)
        }
    
        log.Println("Database migrations applied successfully.")
    }
    ```
    

This approach automates schema management and reduces the risk of deployment errors caused by mismatched application code and database structure.