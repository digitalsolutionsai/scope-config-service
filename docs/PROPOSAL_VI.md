# Scope Config Service - Đề Xuất Kỹ Thuật

## Tóm Tắt

**Scope Config Service** là hệ thống quản lý cấu hình tập trung được thiết kế cho hệ sinh thái microservices. Dịch vụ cung cấp khả năng quản lý cấu hình có thông lượng cao, khả năng mở rộng, kiểm soát phiên bản, xác thực theo schema, hỗ trợ đa phạm vi và phân phối thời gian thực.

---

## 1. Phát Biểu Vấn Đề

Trong kiến trúc microservices hiện đại, việc quản lý cấu hình trên hàng chục hoặc hàng trăm dịch vụ đặt ra nhiều thách thức đáng kể:

| Thách Thức | Tác Động |
|------------|----------|
| **Phân tán cấu hình** | Mỗi dịch vụ duy trì file cấu hình riêng, dẫn đến không nhất quán và trùng lặp |
| **Quản lý môi trường** | Xử lý cấu hình qua các môi trường dev, staging và production dễ phát sinh lỗi |
| **Kiểm toán & Tuân thủ** | Theo dõi ai đã thay đổi gì, khi nào và tại sao rất khó khăn nếu không có kiểm soát tập trung |
| **Khả năng mở rộng** | Cấu hình dựa trên file truyền thống không mở rộng tốt khi microservices tăng lên |
| **Cập nhật thời gian thực** | Triển khai thay đổi cấu hình yêu cầu khởi động lại hoặc triển khai lại dịch vụ |
| **Đa thuê bao** | Hỗ trợ cấu hình khác nhau cho các dự án, cửa hàng hoặc người dùng khác nhau rất phức tạp |

---

## 2. Tổng Quan Giải Pháp

Scope Config Service giải quyết các thách thức này bằng cách cung cấp:

- **Hub Cấu Hình Tập Trung**: Nguồn thông tin duy nhất cho tất cả cấu hình dịch vụ
- **Xác Thực Theo Schema**: Template YAML định nghĩa cấu trúc, kiểu dữ liệu và giá trị mặc định
- **Phân Cấp Đa Phạm Vi**: Hỗ trợ cấu hình cấp SYSTEM, PROJECT, STORE và USER
- **Kiểm Soát Phiên Bản**: Các phiên bản bất biến với lịch sử kiểm toán đầy đủ
- **API Thông Lượng Cao**: gRPC cho dịch vụ nội bộ + HTTP REST cho truy cập bên ngoài
- **Phân Phối Thời Gian Thực**: Cấu hình có thể được lấy theo yêu cầu với độ trễ tối thiểu

---

## 3. Kiến Trúc Hệ Thống

### 3.1 Kiến Trúc Tổng Quan

```mermaid
flowchart TB
    subgraph Clients["Ứng Dụng Client"]
        WebUI["Admin Web UI"]
        CLI["Config CLI"]
        MS1["Microservice A"]
        MS2["Microservice B"]
        MS3["Microservice N"]
    end

    subgraph Gateway["Tầng API Gateway"]
        APIGateway["API Gateway<br/>(Spring/Kong/Nginx)"]
    end

    subgraph ConfigService["Scope Config Service"]
        HTTP["HTTP Gateway<br/>:8080"]
        GRPC["gRPC Server<br/>:50051"]
        Service["Config Service<br/>Logic"]
        SeedLoader["Seed Loader"]
    end

    subgraph Database["Tầng Dữ Liệu"]
        PG[(PostgreSQL)]
    end

    WebUI -->|REST/JSON| APIGateway
    CLI -->|gRPC| GRPC
    MS1 -->|gRPC| GRPC
    MS2 -->|gRPC| GRPC
    MS3 -->|gRPC| GRPC
    
    APIGateway -->|Chuyển tiếp| HTTP
    
    HTTP --> Service
    GRPC --> Service
    Service --> PG
    SeedLoader -->|Nạp Templates| Service
```

### 3.2 Kiến Trúc Container Dịch Vụ

```mermaid
flowchart TB
    subgraph Docker["Docker Container: config-service"]
        direction TB
        
        subgraph Entry["Điểm Vào"]
            Main["main.go"]
        end
        
        subgraph Servers["Tầng Server"]
            GRPCServer["gRPC Server<br/>Cổng: 50051"]
            HTTPServer["HTTP Server<br/>Cổng: 8080"]
        end
        
        subgraph Services["Tầng Dịch Vụ"]
            ConfigSvc["ConfigService"]
            TemplateSvc["Template Handler"]
            ConfigHandler["Config Handler"]
        end
        
        subgraph Middleware["Middleware"]
            Logger["Logger"]
            Recovery["Recovery"]
            Auth["Auth (Tùy chọn)"]
            Swagger["Swagger UI"]
        end
        
        subgraph Data["Truy Cập Dữ Liệu"]
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
    
    subgraph External["Bên Ngoài"]
        PostgreSQL[(PostgreSQL<br/>:5432)]
        Templates["templates/*.yaml"]
    end
    
    Migrations --> PostgreSQL
    Repo --> PostgreSQL
    Main -->|"Nạp khi khởi động"| Templates
```

### 3.3 Kiến Trúc Luồng Dữ Liệu

```mermaid
sequenceDiagram
    participant Client as Client Microservice
    participant GRPC as gRPC Server
    participant Service as Config Service
    participant DB as PostgreSQL

    Note over Client,DB: Luồng Lấy Cấu Hình
    Client->>GRPC: GetConfig(identifier)
    GRPC->>Service: Xử lý yêu cầu
    Service->>DB: Truy vấn config_version
    DB-->>Service: Thông tin phiên bản
    Service->>DB: Truy vấn config_field (theo version)
    DB-->>Service: Giá trị các trường
    Service->>DB: Truy vấn config_template_field (mặc định)
    DB-->>Service: Giá trị mặc định Template
    Service-->>Service: Merge Giá trị + Mặc định
    Service-->>GRPC: Phản hồi ScopeConfig
    GRPC-->>Client: Dữ liệu cấu hình

    Note over Client,DB: Luồng Cập Nhật Cấu Hình
    Client->>GRPC: UpdateConfig(identifier, fields)
    GRPC->>Service: Xử lý cập nhật
    Service->>DB: Lấy/Tạo config_version
    DB-->>Service: Bản ghi Version
    Service->>Service: Tăng Version
    Service->>DB: Insert config_field (version mới)
    Service->>DB: Insert config_version_history
    DB-->>Service: Thành công
    Service-->>GRPC: ScopeConfig đã cập nhật
    GRPC-->>Client: Cấu hình mới
```

---

## 4. Tích Hợp Hệ Sinh Thái Microservices

### 4.1 Mô Hình Tích Hợp Dịch Vụ

```mermaid
flowchart LR
    subgraph ConfigManagement["Quản Lý Cấu Hình"]
        SCS["Scope Config Service"]
        PG[(PostgreSQL)]
        SCS --> PG
    end

    subgraph MicroservicesCluster["Cụm Microservices"]
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

### 4.2 Hỗ Trợ SDK Đa Ngôn Ngữ

```mermaid
flowchart TB
    subgraph ProtoDefinition["Định Nghĩa Protocol Buffers"]
        Proto["config.proto<br/>Hợp đồng dịch vụ"]
    end

    subgraph CodeGeneration["buf generate"]
        BufGen["Pipeline sinh mã"]
    end

    subgraph SDKs["SDK Được Sinh"]
        GoSDK["Go SDK<br/>sdks/go/"]
        TSSDK["TypeScript SDK<br/>sdks/typescript/"]
        JavaSDK["Java SDK<br/>sdks/java/"]
        PythonSDK["Python SDK<br/>sdks/python/"]
    end

    subgraph Services["Dịch Vụ Sử Dụng SDK"]
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

## 5. Kiến Trúc Hạ Tầng

### 5.1 Triển Khai Docker Compose

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

        subgraph Volumes["Volume Lưu Trữ"]
            PGData[("postgres_data")]
        end
    end

    PG --> PGData
    PGAdmin -->|Truy cập Admin| PG
    ConfigSvc -->|DATABASE_URL| PG
    APIGateway -->|Định tuyến yêu cầu| ConfigSvc

    subgraph External["Truy Cập Bên Ngoài"]
        Admin["Người dùng Admin"]
        Clients["Dịch vụ Client"]
    end

    Admin -->|":8888"| PGAdmin
    Admin -->|":8080/swagger"| ConfigSvc
    Clients -->|":50051"| ConfigSvc
```

### 5.2 Triển Khai Production Kubernetes

```mermaid
flowchart TB
    subgraph K8sCluster["Cụm Kubernetes"]
        subgraph Ingress["Ingress Controller"]
            NG["Nginx Ingress"]
        end

        subgraph ConfigNamespace["namespace config-service"]
            subgraph Deployment["Deployment: config-service"]
                Pod1["Pod 1<br/>gRPC + HTTP"]
                Pod2["Pod 2<br/>gRPC + HTTP"]
                Pod3["Pod N<br/>gRPC + HTTP"]
            end

            HPA["Horizontal Pod<br/>Autoscaler"]
            SvcGRPC["Service: grpc<br/>ClusterIP :50051"]
            SvcHTTP["Service: http<br/>ClusterIP :8080"]
            CM["ConfigMap<br/>Môi trường"]
            Secret["Secret<br/>Thông tin DB"]
        end

        subgraph DBNamespace["namespace database"]
            PG[(PostgreSQL<br/>StatefulSet)]
            PGSvc["Service: postgres<br/>ClusterIP :5432"]
        end
    end

    NG -->|HTTP Traffic| SvcHTTP
    SvcGRPC --> Pod1 & Pod2 & Pod3
    SvcHTTP --> Pod1 & Pod2 & Pod3
    HPA --> Deployment
    Pod1 & Pod2 & Pod3 -->|Kết nối DB| PGSvc
    PGSvc --> PG
    CM --> Pod1 & Pod2 & Pod3
    Secret --> Pod1 & Pod2 & Pod3
```

---

## 6. Schema Cơ Sở Dữ Liệu

### 6.1 Sơ Đồ Quan Hệ Thực Thể

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

    config_template ||--o{ config_template_field : "có các trường"
    config_version ||--o{ config_version_history : "có lịch sử"
    config_version ||--o{ config_field : "có các trường"
```

---

## 7. Các Tính Năng Chính

### 7.1 Phạm Vi Cấu Hình

```mermaid
flowchart TB
    subgraph ScopeHierarchy["Phân Cấp Phạm Vi Cấu Hình"]
        SYSTEM["Phạm Vi SYSTEM<br/>Mặc định toàn cục<br/>scope_id: 'default'"]
        PROJECT["Phạm Vi PROJECT<br/>Ghi đè dự án<br/>scope_id: project_id"]
        STORE["Phạm Vi STORE<br/>Ghi đè cửa hàng<br/>scope_id: store_id"]
        USER["Phạm Vi USER<br/>Tùy chọn người dùng<br/>scope_id: user_id"]
    end

    SYSTEM -->|"Kế thừa bởi"| PROJECT
    PROJECT -->|"Kế thừa bởi"| STORE
    STORE -->|"Kế thừa bởi"| USER

    subgraph Example["Ví dụ: Cấu Hình Thanh Toán"]
        SYS["SYSTEM<br/>stripe.enabled=true<br/>paypal.enabled=false"]
        PROJ["PROJECT A<br/>paypal.enabled=true"]
        STOR["STORE 123<br/>stripe.fee=2.5%"]
        USR["USER john<br/>(sử dụng kế thừa)"]
    end
```

### 7.2 Kiểm Soát Phiên Bản & Xuất Bản

```mermaid
stateDiagram-v2
    [*] --> Draft: Tạo cấu hình
    Draft --> Version1: Lưu thay đổi
    Version1 --> Version2: Cập nhật
    Version2 --> Version3: Cập nhật
    Version3 --> VersionN: Tiếp tục...
    
    Version1 --> Published: Xuất bản v1
    Version2 --> Published: Xuất bản v2
    VersionN --> Published: Xuất bản vN
    
    Published --> [*]: Cấu hình hoạt động
    
    note right of Draft
        Cấu hình mới
        Chưa có phiên bản
    end note
    
    note right of Published
        Chỉ phiên bản đã xuất bản
        được phục vụ cho clients
        qua GetConfig()
    end note
```

---

## 8. Tóm Tắt API

### 8.1 API gRPC (Dịch Vụ Nội Bộ)

| Phương Thức RPC | Mô Tả |
|-----------------|-------|
| `GetConfig` | Lấy cấu hình đã xuất bản |
| `GetLatestConfig` | Lấy phiên bản mới nhất (đã xuất bản hoặc bản nháp) |
| `GetConfigByVersion` | Lấy phiên bản cụ thể |
| `GetConfigHistory` | Lấy lịch sử phiên bản |
| `UpdateConfig` | Tạo/cập nhật cấu hình (tạo phiên bản mới) |
| `PublishVersion` | Xuất bản phiên bản cụ thể |
| `DeleteConfig` | Xóa cấu hình và tất cả phiên bản |
| `ApplyConfigTemplate` | Áp dụng schema cấu hình |
| `GetConfigTemplate` | Lấy schema template |
| `ListConfigTemplates` | Liệt kê tất cả templates |

### 8.2 API HTTP REST (Truy Cập Bên Ngoài)

| Phương Thức | Endpoint | Mô Tả |
|-------------|----------|-------|
| `GET` | `/api/v1/config/templates` | Liệt kê tất cả templates |
| `GET` | `/api/v1/config/{service}/template` | Lấy template dịch vụ |
| `GET` | `/api/v1/config/{service}/scope/{scope}` | Lấy cấu hình đã xuất bản |
| `PUT` | `/api/v1/config/{service}/scope/{scope}` | Cập nhật cấu hình |
| `GET` | `/api/v1/config/{service}/scope/{scope}/latest` | Lấy cấu hình mới nhất |
| `GET` | `/api/v1/config/{service}/scope/{scope}/history` | Lấy lịch sử phiên bản |
| `POST` | `/api/v1/config/{service}/scope/{scope}/publish` | Xuất bản phiên bản |

---

## 9. Hiệu Suất & Khả Năng Mở Rộng

### 9.1 Thiết Kế Thông Lượng Cao

- **Giao Thức gRPC**: Tuần tự hóa nhị phân, ghép kênh HTTP/2, streaming hai chiều
- **Connection Pooling**: Tái sử dụng kết nối cơ sở dữ liệu
- **Kiến Trúc Stateless**: Mở rộng ngang không cần session affinity
- **Chiến Lược Cache**: Cache phía client với vô hiệu hóa dựa trên phiên bản

### 9.2 Mục Tiêu Khả Năng Mở Rộng

| Chỉ Số | Mục Tiêu |
|--------|----------|
| Yêu cầu Đọc/Giây | 10,000+ |
| Yêu cầu Ghi/Giây | 1,000+ |
| Số Lượng Cấu Hình | 100,000+ |
| Độ Trễ Phản Hồi (p99) | <50ms |
| Kết Nối Đồng Thời | 5,000+ |

---

## 10. Cân Nhắc Bảo Mật

- **Xác Thực**: Ủy quyền cho API Gateway (OAuth2/JWT)
- **Phân Quyền**: Kiểm soát truy cập dựa trên vai trò tại tầng gateway
- **Dữ Liệu Nhạy Cảm**: Kiểu trường `SECRET` cho API keys/credentials (ẩn trong UI)
- **Lịch Sử Kiểm Toán**: Lịch sử phiên bản đầy đủ với thuộc tính người dùng
- **TLS**: gRPC hỗ trợ TLS cho truyền tải mã hóa

---

## 11. Bắt Đầu

### Khởi Động Nhanh với Docker Compose

```bash
# Clone repository (thay thế bằng URL repository của bạn)
git clone <repository-url>
cd scope-config-service

# Cấu hình môi trường
cp .env.example .env

# Khởi động dịch vụ
docker compose -f compose.postgres.yml -f compose.yml up -d --build

# Các điểm truy cập:
# - gRPC: localhost:50051
# - HTTP: http://localhost:8080
# - Swagger: http://localhost:8080/swagger/index.html
# - pgAdmin: http://localhost:8888
```

### Sử Dụng CLI

```bash
# Áp dụng template
docker compose exec config-service config-cli template apply -f /app/templates/payment.yaml

# Thiết lập cấu hình
docker compose exec config-service config-cli set \
    --service-name=payment \
    --scope=PROJECT \
    --project-id=proj-123 \
    --group-id=stripe \
    stripe.enabled=true

# Lấy cấu hình
docker compose exec config-service config-cli get \
    --service-name=payment \
    --scope=PROJECT \
    --project-id=proj-123 \
    --group-id=stripe

# Xuất bản cấu hình
docker compose exec config-service config-cli publish 1 \
    --service-name=payment \
    --scope=PROJECT \
    --project-id=proj-123 \
    --group-id=stripe
```

---

## 12. Kết Luận

Scope Config Service cung cấp giải pháp mạnh mẽ, có khả năng mở rộng cho quản lý cấu hình tập trung trong hệ sinh thái microservices. Cách tiếp cận hướng schema đảm bảo tính nhất quán, trong khi kiểm soát phiên bản và hỗ trợ đa phạm vi cho phép quản lý cấu hình linh hoạt, có thể kiểm toán trên các hệ thống phân tán phức tạp.

---

## Tài Liệu Tham Khảo

- [README.md](../README.md) - Tổng quan dự án và thiết lập
- [Tài liệu HTTP Gateway](./HTTP_GATEWAY.md) - Chi tiết REST API
- [Định nghĩa Protocol Buffers](../proto/config/v1/config.proto) - Hợp đồng gRPC
- [Ví dụ Template](../templates/) - Ví dụ schema cấu hình
