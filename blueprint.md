# Project Blueprint: Scope Config Service

## I. High-Level Vision

This project, the Scope Config Service, is a centralized, schema-driven, version-controlled configuration management system for a microservices environment. It allows developers to define a schema for their configurations (a "template"), manage values against that schema, and safely publish updates across different services, projects, and scopes. It will serve as the single source of truth for all application configuration.

## II. Core Components & Current Status

| Component                 | Technology      | Status & Notes                                                                                                                                                                                                                                                                                                  |
| ------------------------- | --------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **gRPC API**              | Protobuf v3     | **EVOLVED**. The API contract in `config.proto` has been updated to support a more flexible identification strategy using `scope` and `scope_id`.                                                                                                                                                              |
| **Server**                | Go, gRPC-Go     | **IN PROGRESS**. The server has been updated to use the new identification strategy. The core logic for handling configurations has been refactored to be more efficient and maintainable.                                                                                                                              |
| **CLI Client**            | Go, Cobra       | **FUNCTIONAL**. The CLI has been updated to use the new `scope` and `scope_id` flags. The `service-name` flag is now required.                                                                                                                                                                           |
| **Database**              | PostgreSQL      | **UPDATED**. The database schema has been updated to use `scope` and `scope_id` for resource identification.                                                                                                                                                                                                   |
| **Tooling & Automation**  | Makefile, Shell | **COMPLETE & ROBUST**. The `Makefile` and `scripts/gen-proto.sh` provide a solid foundation for development and code generation. This will be extended for SDK generation.                                                                                                                                        |
| **Containerization**      | Docker          | **READY**. `docker-compose.yml` files are in place for local development.                                                                                                                                                                                                                                     |

## III. Next Steps & Development Roadmap

### Phase 1: Implement Configuration Templates (Schema-Driven Config)

This is the highest priority. The goal is to allow users to define a configuration schema using YAML, which the service will then enforce. The template defines *what* can be configured, while the existing config messages define the *values*.

1.  **Database Migrations**: Create and run new database migration files to add tables for storing template data.
    *   A `config_template` table to hold the schema, uniquely identified by `service_name` and a combination of `scope` and `scope_id`.
    *   A `config_field_template` table to store the details for each field within a template (path, label, description, type, `display_on` scopes, etc.).

2.  **Implement `ApplyConfigTemplate` RPC**:
    *   The server-side logic will parse the incoming request from the CLI.
    *   It will use an **"insert on duplicate key update" (upsert)** strategy to save the template data into the new tables.
    *   The unique key for a template (and therefore the target for the upsert) is the combination of `service_name`, `scope`, and `scope_id`.

3.  **Implement `GetConfigTemplate` RPC**:
    *   This RPC will fetch the stored template definition from the database. This is critical for clients (like a future web UI) to dynamically build configuration forms.

4.  **CLI Command**: 
    *   Create a new CLI command, `scope-cli template apply -f <template.yaml>`.
    *   This command will be responsible for reading the YAML file, unmarshaling it into the `ConfigTemplate` protobuf message, and sending it to the `ApplyConfigTemplate` RPC.

5.  **Validation Logic in `UpdateConfig`**:
    *   Enhance the existing `UpdateConfig` RPC.
    *   When a user tries to save a configuration value, the RPC must first check if a template exists for that `service_name`, `scope`, and `scope_id`.
    *   If a template exists, `UpdateConfig` must validate the incoming data against the schema (e.g., is the scope allowed based on `display_on`? Is the value one of the predefined `options`?).

### Phase 2: Client SDK Generation

1.  **Create SDK Generation Scripts**: Add new targets to the `Makefile` (e.g., `make sdk-ts`, `make sdk-java`, `make sdk-python`).
2.  **Implement `protoc` Commands**: These scripts will execute `protoc` with the appropriate gRPC plugins to generate client-side SDKs for TypeScript (for NestJS), Java, and Python.
3.  **Provide Documentation**: Add a section to the project's `README.md` explaining how to generate and use the client SDKs.

## IV. Key Architectural Decisions & Patterns

*   **Schema-Driven Configuration**: By introducing `ConfigTemplate`, the service enforces a schema. This provides structure, validation, and enables auto-generation of UIs. The template is the *schema*, the `ScopeConfig` is the *data*.
*   **API-First Design**: The gRPC API (`config.proto`) serves as the central contract, ensuring clear separation between the server and any clients.
*   **Upsert for Templates**: Templates are applied idempotently using an upsert operation, simplifying template management.
*   **Versioned Values**: Configuration *values* are immutable and versioned. Every update creates a new version, allowing for easy rollbacks and a clear audit trail. Templates themselves are not versioned; applying a new template simply updates the current schema.
*   **Automation via Make & Scripts**: A `Makefile` abstracts away complex commands, ensuring a reliable and consistent build and generation process.
