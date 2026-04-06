# Scope Config Service - Technical Proposal

## Executive Summary

**Scope Config Service** is a centralized configuration management system designed for microservices ecosystems. It provides high-throughput, scalable, and version-controlled configuration management with schema-driven validation, multi-scope support, and real-time distribution capabilities.

---

## 1. Problem Statement

In modern microservices architectures, managing configurations across dozens or hundreds of services presents significant challenges:

| Challenge | Impact |
|-----------|--------|
| **Configuration Sprawl** | Each service maintains its own configuration files, leading to inconsistency and duplication |
| **Environment Management** | Handling configurations across dev, staging, and production environments becomes error-prone |
| **Audit & Compliance** | Tracking who changed what, when, and why is difficult without centralized control |
| **Scalability** | Traditional file-based configurations don't scale well with growing microservices |
| **Real-time Updates** | Rolling out configuration changes requires service restarts or redeployments |
| **Multi-tenancy** | Supporting different configurations for different projects, stores, or users is complex |

---

## 2. Solution Overview

Scope Config Service addresses these challenges by providing:

- **Centralized Configuration Hub**: Single source of truth for all service configurations
- **Schema-Driven Validation**: YAML templates define configuration structure, types, and defaults
- **Multi-Scope Hierarchy**: Support for SYSTEM, PROJECT, STORE, and USER level configurations
- **Version Control**: Immutable versions with complete audit trail
- **High-Throughput API**: gRPC for internal services + HTTP REST for external access
- **Real-time Distribution**: Configurations can be fetched on-demand with minimal latency

---

## 3. System Architecture

### 3.1 High-Level Architecture

```mermaid
flowchart TB
    subgraph Clients["Client Applications"]
        WebUI["Admin Web UI"]
        CLI["Config CLI"]
        MS1["Microservice A"]
        MS2["Microservice B"]
        MS3["Microservice N"]
    end

    subgraph Gateway["API Gateway Layer"]
        APIGateway["API Gateway<br/>(Spring/Kong/Nginx)"]
    end

    subgraph ConfigService["Scope Config Service"]
        HTTP["HTTP Gateway<br/>:8080"]
        GRPC["gRPC Server<br/>:50051"]
        Service["Config Service<br/>Logic"]
        SeedLoader["Seed Loader"]
    end

    subgraph Database["Data Layer"]
        PG[(PostgreSQL)]
    end

    WebUI -->|REST/JSON| APIGateway
    CLI -->|gRPC| GRPC
    MS1 -->|gRPC| GRPC
    MS2 -->|gRPC| GRPC
    MS3 -->|gRPC| GRPC
    
    APIGateway -->|Forward| HTTP
    
    HTTP --> Service
    GRPC --> Service
    Service --> PG
    SeedLoader -->|Load Templates| Service
```

### 3.2 Service Container Architecture

```mermaid
flowchart TB
    subgraph Docker["Docker Container: config-service"]
        direction TB
        
        subgraph Entry["Entrypoint"]
            Main["main.go"]
        end
        
        subgraph Servers["Server Layer"]
            GRPCServer["gRPC Server<br/>Port: 50051"]
            HTTPServer["HTTP Server<br/>Port: 8080"]
        end
        
        subgraph Services["Service Layer"]
            ConfigSvc["ConfigService"]
            TemplateSvc["Template Handler"]
            ConfigHandler["Config Handler"]
        end
        
        subgraph Middleware["Middleware"]
            Logger["Logger"]
            Recovery["Recovery"]
            Auth["Auth (Optional)"]
            Swagger["Swagger UI"]
        end
        
        subgraph Data["Data Access"]
            Repo["Repository"]
            Migrations["Migrations"]
        end
        
        Main --> GRPCServer
        Main --> HTTPServer
        
        GRPCServer --> ConfigSvc
        HTTPServer --> Middleware
        Middleware --> ConfigSvc
        
        ConfigSvc --> TemplateSvc
        ConfigSvc --> ConfigHandler
        TemplateSvc --> Repo
        ConfigHandler --> Repo
        Repo --> Migrations
    end
    
    subgraph External["External"]
        PostgreSQL[(PostgreSQL<br/>:5432)]
        Templates["templates/*.yaml"]
    end
    
    Migrations --> PostgreSQL
    Repo --> PostgreSQL
    Main -->|"Load on Startup"| Templates
```

### 3.3 Data Flow Architecture

```mermaid
sequenceDiagram
    participant Client as Microservice Client
    participant GRPC as gRPC Server
    participant Service as Config Service
    participant DB as PostgreSQL

    Note over Client,DB: Get Configuration Flow
    Client->>GRPC: GetConfig(identifier)
    GRPC->>Service: Process Request
    Service->>DB: Query config_version
    DB-->>Service: Version Info
    Service->>DB: Query config_field (by version)
    DB-->>Service: Field Values
    Service->>DB: Query config_template_field (defaults)
    DB-->>Service: Template Defaults
    Service-->>Service: Merge Values + Defaults
    Service-->>GRPC: ScopeConfig Response
    GRPC-->>Client: Configuration Data

    Note over Client,DB: Update Configuration Flow
    Client->>GRPC: UpdateConfig(identifier, fields)
    GRPC->>Service: Process Update
    Service->>DB: Get/Create config_version
    DB-->>Service: Version Record
    Service->>Service: Increment Version
    Service->>DB: Insert config_field (new version)
    Service->>DB: Insert config_version_history
    DB-->>Service: Success
    Service-->>GRPC: Updated ScopeConfig
    GRPC-->>Client: New Configuration
```

---

## 4. Microservices Ecosystem Integration

### 4.1 Service Integration Pattern

```mermaid
flowchart LR
    subgraph ConfigManagement["Configuration Management"]
        SCS["Scope Config Service"]
        PG[(PostgreSQL)]
        SCS --> PG
    end

    subgraph MicroservicesCluster["Microservices Cluster"]
        subgraph PaymentMS["Payment Service"]
            PaymentApp["App Logic"]
            PaymentSDK["Config SDK"]
            PaymentApp --> PaymentSDK
        end
        
        subgraph NotificationMS["Notification Service"]
            NotifApp["App Logic"]
            NotifSDK["Config SDK"]
            NotifApp --> NotifSDK
        end
        
        subgraph AIGateway["AI Gateway Service"]
            AIApp["App Logic"]
            AISDK["Config SDK"]
            AIApp --> AISDK
        end
        
        subgraph ChatbotMS["Chatbot Service"]
            ChatApp["App Logic"]
            ChatSDK["Config SDK"]
            ChatApp --> ChatSDK
        end
    end

    PaymentSDK -->|gRPC| SCS
    NotifSDK -->|gRPC| SCS
    AISDK -->|gRPC| SCS
    ChatSDK -->|gRPC| SCS
```

### 4.2 Multi-Language SDK Support

```mermaid
flowchart TB
    subgraph ProtoDefinition["Protocol Buffers Definition"]
        Proto["config.proto<br/>Service Contract"]
    end

    subgraph CodeGeneration["buf generate"]
        BufGen["Code Generation Pipeline"]
    end

    subgraph SDKs["Generated SDKs"]
        GoSDK["Go SDK<br/>sdks/go/"]
        TSSDK["TypeScript SDK<br/>sdks/typescript/"]
        JavaSDK["Java SDK<br/>sdks/java/"]
        PythonSDK["Python SDK<br/>sdks/python/"]
    end

    subgraph Services["Services Using SDKs"]
        GoSvc["Go Services"]
        NodeSvc["Node.js Services"]
        JavaSvc["Java Services"]
        PySvc["Python Services"]
    end

    Proto --> BufGen
    BufGen --> GoSDK
    BufGen --> TSSDK
    BufGen --> JavaSDK
    BufGen --> PythonSDK

    GoSDK --> GoSvc
    TSSDK --> NodeSvc
    JavaSDK --> JavaSvc
    PythonSDK --> PySvc
```

---

## 5. Infrastructure Architecture

### 5.1 Docker Compose Deployment

```mermaid
flowchart TB
    subgraph DockerCompose["Docker Compose Stack"]
        subgraph Network1["postgres_network"]
            PG["PostgreSQL<br/>:5432"]
            PGAdmin["pgAdmin<br/>:80"]
            ConfigSvc["Config Service<br/>gRPC :50051<br/>HTTP :8080"]
        end

        subgraph Network2["gateway_shared_network"]
            APIGateway["API Gateway"]
            ConfigSvc
        end

        subgraph Volumes["Persistent Volumes"]
            PGData[("postgres_data")]
        end
    end

    PG --> PGData
    PGAdmin -->|Admin Access| PG
    ConfigSvc -->|DATABASE_URL| PG
    APIGateway -->|Route Requests| ConfigSvc

    subgraph External["External Access"]
        Admin["Admin Users"]
        Clients["Client Services"]
    end

    Admin -->|":8888"| PGAdmin
    Admin -->|":8080/swagger"| ConfigSvc
    Clients -->|":50051"| ConfigSvc
```

### 5.2 Production Kubernetes Deployment

```mermaid
flowchart TB
    subgraph K8sCluster["Kubernetes Cluster"]
        subgraph Ingress["Ingress Controller"]
            NG["Nginx Ingress"]
        end

        subgraph ConfigNamespace["config-service namespace"]
            subgraph Deployment["Deployment: config-service"]
                Pod1["Pod 1<br/>gRPC + HTTP"]
                Pod2["Pod 2<br/>gRPC + HTTP"]
                Pod3["Pod N<br/>gRPC + HTTP"]
            end

            HPA["Horizontal Pod<br/>Autoscaler"]
            SvcGRPC["Service: grpc<br/>ClusterIP :50051"]
            SvcHTTP["Service: http<br/>ClusterIP :8080"]
            CM["ConfigMap<br/>Environment"]
            Secret["Secret<br/>DB Credentials"]
        end

        subgraph DBNamespace["database namespace"]
            PG[(PostgreSQL<br/>StatefulSet)]
            PGSvc["Service: postgres<br/>ClusterIP :5432"]
        end
    end

    NG -->|HTTP Traffic| SvcHTTP
    SvcGRPC --> Pod1 & Pod2 & Pod3
    SvcHTTP --> Pod1 & Pod2 & Pod3
    HPA --> Deployment
    Pod1 & Pod2 & Pod3 -->|DB Connection| PGSvc
    PGSvc --> PG
    CM --> Pod1 & Pod2 & Pod3
    Secret --> Pod1 & Pod2 & Pod3
```

---

## 6. Database Schema

### 6.1 Entity Relationship Diagram

```mermaid
erDiagram
    config_template {
        serial id PK
        varchar service_name UK
        varchar service_label
        varchar group_id UK
        varchar group_label
        text group_description
        boolean is_active
        int sort_order
        timestamptz created_at
        timestamptz updated_at
    }

    config_template_field {
        serial id PK
        int config_template_id FK
        varchar path UK
        varchar label
        text description
        varchar type
        text default_value
        scope_enum[] display_on
        jsonb options
        int sort_order
    }

    config_version {
        serial id PK
        varchar service_name
        scope_enum scope
        varchar scope_id
        varchar group_id
        int latest_version
        int published_version
        timestamptz created_at
        varchar created_by
        timestamptz updated_at
        varchar updated_by
    }

    config_version_history {
        serial id PK
        int config_version_id FK
        int version UK
        timestamptz created_at
        varchar created_by
    }

    config_field {
        serial id PK
        int config_version_id FK
        int version
        varchar path
        text value
        varchar type
        boolean is_active
    }

    config_template ||--o{ config_template_field : "has fields"
    config_version ||--o{ config_version_history : "has history"
    config_version ||--o{ config_field : "has fields"
```

---

## 7. Key Features

### 7.1 Configuration Scoping

```mermaid
flowchart TB
    subgraph ScopeHierarchy["Configuration Scope Hierarchy"]
        SYSTEM["SYSTEM Scope<br/>Global Defaults<br/>scope_id: 'default'"]
        PROJECT["PROJECT Scope<br/>Project Override<br/>scope_id: project_id"]
        STORE["STORE Scope<br/>Store Override<br/>scope_id: store_id"]
        USER["USER Scope<br/>User Preferences<br/>scope_id: user_id"]
    end

    SYSTEM -->|"Inherited by"| PROJECT
    PROJECT -->|"Inherited by"| STORE
    STORE -->|"Inherited by"| USER

    subgraph Example["Example: Payment Config"]
        SYS["SYSTEM<br/>stripe.enabled=true<br/>paypal.enabled=false"]
        PROJ["PROJECT A<br/>paypal.enabled=true"]
        STOR["STORE 123<br/>stripe.fee=2.5%"]
        USR["USER john<br/>(uses inherited)"]
    end
```

### 7.2 Version Control & Publishing

```mermaid
stateDiagram-v2
    [*] --> Draft: Create Config
    Draft --> Version1: Save Changes
    Version1 --> Version2: Update
    Version2 --> Version3: Update
    Version3 --> VersionN: Continue...
    
    Version1 --> Published: Publish v1
    Version2 --> Published: Publish v2
    VersionN --> Published: Publish vN
    
    Published --> [*]: Active Config
    
    note right of Draft
        New configuration
        No version yet
    end note
    
    note right of Published
        Only published version
        is served to clients
        via GetConfig()
    end note
```

---

## 8. API Summary

### 8.1 gRPC API (Internal Services)

| RPC Method | Description |
|------------|-------------|
| `GetConfig` | Get published configuration |
| `GetLatestConfig` | Get latest version (published or draft) |
| `GetConfigByVersion` | Get specific version |
| `GetConfigHistory` | Get version history |
| `UpdateConfig` | Create/update configuration (creates new version) |
| `PublishVersion` | Publish a specific version |
| `DeleteConfig` | Delete configuration and all versions |
| `ApplyConfigTemplate` | Apply configuration schema |
| `GetConfigTemplate` | Get template schema |
| `ListConfigTemplates` | List all templates |

### 8.2 HTTP REST API (External Access)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/config/templates` | List all templates |
| `GET` | `/api/v1/config/{service}/template` | Get service template |
| `GET` | `/api/v1/config/{service}/scope/{scope}` | Get published config |
| `PUT` | `/api/v1/config/{service}/scope/{scope}` | Update config |
| `GET` | `/api/v1/config/{service}/scope/{scope}/latest` | Get latest config |
| `GET` | `/api/v1/config/{service}/scope/{scope}/history` | Get version history |
| `POST` | `/api/v1/config/{service}/scope/{scope}/publish` | Publish version |

---

## 9. Performance & Scalability

### 9.1 High Throughput Design

- **gRPC Protocol**: Binary serialization, HTTP/2 multiplexing, bidirectional streaming
- **Connection Pooling**: Database connection reuse
- **Stateless Architecture**: Horizontal scaling without session affinity
- **Caching Strategy**: Client-side caching with version-based invalidation

### 9.2 Scalability Targets

| Metric | Target |
|--------|--------|
| Read Requests/Second | 10,000+ |
| Write Requests/Second | 1,000+ |
| Configuration Count | 100,000+ |
| Response Latency (p99) | <50ms |
| Concurrent Connections | 5,000+ |

---

## 10. Security Considerations

- **Authentication**: Delegated to API Gateway (OAuth2/JWT)
- **Authorization**: Role-based access control at gateway level
- **Sensitive Data**: `SECRET` field type for API keys/credentials (masked in UI)
- **Audit Trail**: Complete version history with user attribution
- **TLS**: gRPC supports TLS for encrypted transport

---

## 11. Getting Started

### Quick Start with Docker Compose

```bash
# Clone the repository (replace with your repository URL)
git clone <repository-url>
cd scope-config-service

# Configure environment
cp .env.example .env

# Start services
docker compose -f compose.postgres.yml -f compose.yml up -d --build

# Access points:
# - gRPC: localhost:50051
# - HTTP: http://localhost:8080
# - Swagger: http://localhost:8080/swagger/index.html
# - pgAdmin: http://localhost:8888
```

### Using the CLI

```bash
# Apply a template
docker compose exec config-service config-cli template apply -f /app/templates/payment.yaml

# Set configuration
docker compose exec config-service config-cli set \
    --service-name=payment \
    --scope=PROJECT \
    --project-id=proj-123 \
    --group-id=stripe \
    stripe.enabled=true

# Get configuration
docker compose exec config-service config-cli get \
    --service-name=payment \
    --scope=PROJECT \
    --project-id=proj-123 \
    --group-id=stripe

# Publish configuration
docker compose exec config-service config-cli publish 1 \
    --service-name=payment \
    --scope=PROJECT \
    --project-id=proj-123 \
    --group-id=stripe
```

---

## 12. Conclusion

Scope Config Service provides a robust, scalable solution for centralized configuration management in microservices ecosystems. Its schema-driven approach ensures consistency, while version control and multi-scope support enable flexible, auditable configuration management across complex distributed systems.

---

## References

- [README.md](../README.md) - Project overview and setup
- [HTTP Gateway Documentation](./HTTP_GATEWAY.md) - REST API details
- [Protocol Buffers Definition](../proto/config/v1/config.proto) - gRPC contract
- [Template Examples](../templates/) - Configuration schema examples
