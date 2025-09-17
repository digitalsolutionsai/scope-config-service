# System Design Document: ScopeConfig Service

- **Version:** 1.2
    
- **Last Updated:** September 15, 2025
    
- **Author:** Gemini
    
- **Status:** In Development

## 1. Overview and Project Purpose

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

## 2. System Architecture Design

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
|   Client Microservice B  | ------> |        (Golang)            | ------> |    PostgreSQL     |
+--------------------------+         |                            |         |     Database      |
                                    |    +------------------+    |         +-------------------+
+--------------------------+         |    | In-Memory Cache  |    |
|   Admin/Dev Tool (CLI)   | ------> |    +------------------+    |
+--------------------------+         |                            |
                                    |                            |
+--------------------------+         +----------------------------+
|      Java SDK            |
+--------------------------+

+--------------------------+         
|    NestJS (Node.js) SDK  |
+--------------------------+

+--------------------------+         
|        Python SDK        |
+--------------------------+         
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

## 3. API Design (gRPC Proto)

The `proto/config/v1/config.proto` file will define the entire service interface. Compared to the original proto file, this version is improved for clarity and adherence to gRPC best practices.

```
syntax = "proto3";

package vn.dsai.config.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

option go_package = "github.com/vn-dsai/scope-config-service/gen/go/config/v1;configv1";

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

## 4. Data Model (Database Schema)

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

## 5. Current State of Implementation

The project has a solid foundation and the core service is functional.

*   **gRPC Service:** The main `ConfigService` is implemented in Go, providing all the necessary RPC methods for managing and retrieving configurations.
*   **Database Schema:** The PostgreSQL database schema is defined and managed through migration files. Indexes have been added to optimize query performance.
*   **Command-Line Interface (CLI):** A CLI tool is available for interacting with the service. It allows users to `set`, `get`, `publish`, and `show` configurations.
*   **Containerization:** The entire application is containerized using Docker, making it easy to build, deploy, and run.

## 6. Next Steps: Client SDKs

To facilitate the adoption of the **ScopeConfig Service** across different platforms, we will develop client SDKs for the following languages:

*   **Java**
*   **NestJS (Node.js/TypeScript)**
*   **Python**

### 6.1. SDK Development Strategy

The general strategy for developing these SDKs will be as follows:

1.  **Generate gRPC Client:** Use the existing `config.proto` file to generate the gRPC client code for each target language.
2.  **Create a Wrapper/Facade:** Create a user-friendly wrapper class or module that simplifies the interaction with the generated gRPC client. This wrapper will:
    *   Handle the gRPC connection and channel setup.
    *   Provide idiomatic methods that are easy to understand and use in the target language (e.g., using Promises in Node.js, native data types in Python).
    *   Abstract away the complexities of the gRPC request and response objects.
3.  **Implement Configuration Parsing:** The SDKs should include utility functions to parse the configuration values from the `ConfigField` objects into the correct data types (e.g., string, integer, boolean, JSON).
4.  **Provide Clear Documentation and Examples:** For each SDK, create a `README.md` file with clear instructions on how to install and use it. Include code examples for common use cases.

### 6.2. Java SDK

*   **Build Tool:** Maven or Gradle.
*   **gRPC Generation:** Use the `protobuf-maven-plugin` or the `protobuf-gradle-plugin` to generate the Java gRPC client.
*   **Wrapper Class:** Create a `ScopeConfigClient` class that encapsulates the gRPC client and provides methods like `getConfig()`, `publishVersion()`, etc.

### 6.3. NestJS SDK

*   **Package Manager:** npm or yarn.
*   **gRPC Generation:** Use `@grpc/grpc-js` and `ts-protoc-gen` to generate the Node.js gRPC client and TypeScript definitions.
*   **NestJS Module:** Create a `ScopeConfigModule` that can be imported into a NestJS application. This module will provide a `ScopeConfigService` that can be injected into other services.

### 6.4. Python SDK

*   **Package Manager:** pip.
*   **gRPC Generation:** Use `grpcio-tools` to generate the Python gRPC client.
*   **Wrapper Class:** Create a `ScopeConfigClient` class that provides a simple and Pythonic interface for interacting with the service.
