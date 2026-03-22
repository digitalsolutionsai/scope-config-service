# HTTP Gateway API Documentation

The HTTP Gateway provides a REST API wrapper around the gRPC Scope Configuration Service. It enables frontend applications, QA teams, and external clients to interact with the configuration service using simple HTTP/JSON requests.

## Table of Contents

- [Getting Started](#getting-started)
- [Authentication](#authentication)
- [API Endpoints](#api-endpoints)
  - [List Templates](#list-templates)
  - [Import Template](#import-template)
  - [Toggle Active Template](#toggle-active-template)
  - [Get Template](#get-template)
  - [Get Configuration (Published)](#get-configuration-published)
  - [Update Configuration](#update-configuration)
  - [Get Configuration (Latest)](#get-configuration-latest)
  - [Get Version History](#get-version-history)
  - [Publish Configuration](#publish-configuration)
- [Error Handling](#error-handling)
- [Complete Workflow Examples](#complete-workflow-examples)

---

## Getting Started

### Running the Service

The config service now runs both gRPC and HTTP in a **single container**. The HTTP gateway internally connects to the local gRPC service.

#### With Docker Compose (Recommended)

```bash
# Start all services (database and unified config service with both gRPC and HTTP)
docker compose -f compose.postgres.yml -f compose.yml up -d --build
```

Both endpoints will be available from the same container:
- **gRPC**: `localhost:50051`
- **HTTP**: `http://localhost:8080`
- **Swagger UI**: `http://localhost:8080/swagger/index.html`

### Interactive API Documentation (Swagger UI)

The service provides an interactive Swagger UI for exploring and testing the API:

1. Start the service (see above)
2. Open your browser to `http://localhost:8080/swagger/index.html`
3. Browse all available endpoints with detailed documentation
4. Try out API calls directly from the Swagger interface
5. View request/response schemas and example payloads

This is the easiest way to understand the API and test endpoints during development.

#### Standalone (Local Development)

```bash
# Build and run the unified server (includes both gRPC and HTTP)
make build-server
./bin/server

# Or use make
make run-server
```

The server will start both gRPC (port 50051) and HTTP (port 8080) services.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GRPC_PORT` | `50051` | Port for the gRPC service |
| `HTTP_PORT` | `8080` | Port for the HTTP gateway |
| `KEYCLOAK_PUBLIC_KEY` | _(none)_ | RSA public key from Keycloak for JWT validation (PEM or base64 format). If not set, authentication is disabled. |
| `KEYCLOAK_CLIENT` | `dsai-console` | Keycloak client name to check for roles |
| `KEYCLOAK_ROLES` | `ROLE_ADMIN` | Comma-separated list of required roles (user needs at least one) |

**⚠️ Security Note:** Running without `KEYCLOAK_PUBLIC_KEY` disables all authentication and should only be used for development/testing.

---

## Authentication

**Authentication is handled at the API Gateway level** (e.g., Spring Cloud Gateway, Kong, Nginx).

This service exposes **public HTTP APIs** and relies on an upstream API gateway to handle:
- JWT token validation
- Role-based access control (RBAC)
- Rate limiting
- Request routing

### Architecture

```
Client → API Gateway (Auth/RBAC) → Config Service (Public APIs)
```

The API Gateway should:
1. Validate JWT tokens
2. Check user permissions/roles
3. Forward authenticated requests to this service
4. Optionally pass user information in headers (e.g., `X-User-Email`, `X-User-ID`)

### Example Request (Through API Gateway)

```bash
# Client calls API Gateway with authentication
curl -X GET "https://gateway.example.com/config/payment-service/template?groupId=stripe" \
  -H "Authorization: Bearer <JWT_TOKEN>"

# Gateway validates token and forwards to service
# Service receives request without auth headers (gateway handles it)
```

### User Information for Audit

When publishing configurations, you can provide `userName` in the request body for audit logging:

```bash
curl -X POST "http://localhost:8080/api/v1/config/payment/scope/SYSTEM/publish?groupId=payment-methods" \
  -H "Content-Type: application/json" \
  -d '{
    "version": 5,
    "userName": "user@example.com"
  }'
```

The `userName` field is **required** for audit trail purposes.

---

## API Endpoints

### List Templates

Retrieves a list of **active** configuration templates, including their group IDs, labels, and descriptions. This is useful for populating a "Select Group" dropdown in a UI. Only templates with `is_active=true` are returned.

**Endpoint:** `GET /api/v1/config/templates`

**Query Parameters:**
- `serviceName` (optional): Filter templates by service name

**Example Request:**

```bash
# Get all active templates
curl -X GET "http://localhost:8080/api/v1/config/templates"

# Get active templates for a specific service
curl -X GET "http://localhost:8080/api/v1/config/templates?serviceName=payment"

# Get ALL templates including inactive ones (Admin view)
curl -X GET "http://localhost:8080/api/v1/config/templates?includeInactive=true"
```

**Example Response:**

```json
{
  "templates": [
    {
      "serviceName": "payment",
      "serviceLabel": "Payment Service",
      "groupId": "payment-methods",
      "groupLabel": "Payment Methods",
      "description": "Configuration for available payment methods.",
      "fields": [...]
    },
    {
      "serviceName": "api-gateway",
      "serviceLabel": "API Gateway",
      "groupId": "default-rate-limit",
      "groupLabel": "Rate Limits",
      "description": "Global rate limiting settings.",
      "fields": [...]
    }
  ]
}
```

---

### Import Template

Creates or updates a template group from a JSON payload (which normally mirrors the YAML file definition).

- **Method:** `POST`
- **Path:** `/api/v1/config/templates`
- **Headers:** `X-User-Name: admin@example.com` (Required if not authenticated)

**Example Request Payload:**
```json
{
  "service": {
    "id": "chatbot",
    "label": "Chatbot Service"
  },
  "groups": [
    {
      "id": "ai-settings",
      "label": "AI Settings",
      "sortOrder": 100000,
      "fields": [
        {
          "path": "ai-provider",
          "label": "AI Provider",
          "type": "STRING",
          "defaultValue": "google",
          "displayOn": ["SYSTEM"]
        }
      ]
    }
  ],
  "userName": "admin@example.com"
}
```

**Example Response:**
```json
{
  "results": [
    {
      "serviceName": "chatbot",
      "groupId": "ai-settings",
      "status": "ok"
    }
  ]
}
```

---

### Toggle Active Template

Enables or disables a template group by toggling its `is_active` status. Inactive templates are hidden from normal config UI views.

- **Method:** `PATCH`
- **Path:** `/api/v1/config/templates/{serviceName}/{groupId}/active`

**Example Request:**
```bash
curl -X PATCH "http://localhost:8080/api/v1/config/templates/chatbot/ai-settings/active" \
  -H "Content-Type: application/json" \
  -d '{"active": false}'
```

**Example Response:**
```json
{
  "active": false,
  "groupId": "ai-settings",
  "serviceName": "chatbot"
}
```

---

### Get Template

Retrieves the template (schema) for a specific service and group. This is essential for **rendering dynamic UI forms** where users can set configuration values.

**Endpoint:** `GET /api/v1/config/{serviceName}/template`

**Query Parameters:**
- `groupId` (required): The configuration group ID

**Response:** Returns template metadata including field definitions, types, default values, and display options.

**Example Request:**

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/template?groupId=payment-methods"
```

**Example Response:**

```json
{
  "serviceName": "payment",
  "serviceLabel": "Payment Service",
  "groupId": "payment-methods",
  "groupLabel": "Payment Methods",
  "description": "Configuration for available payment methods.",
  "fields": [
    {
      "path": "methods/credit-card",
      "label": "Credit Card",
      "description": "Credit card payment method",
      "type": "BOOLEAN",
      "defaultValue": "true",
      "displayOn": [
        "SYSTEM",
        "USER"
      ]
    },
    {
      "path": "methods/paypal",
      "label": "PayPal",
      "description": "PayPal payment method",
      "type": "BOOLEAN",
      "defaultValue": "true",
      "displayOn": [
        "SYSTEM",
        "USER"
      ]
    },
    {
      "path": "methods/stripe",
      "label": "Stripe",
      "description": "Stripe payment method",
      "type": "BOOLEAN",
      "defaultValue": "false",
      "displayOn": [
        "SYSTEM",
        "USER"
      ]
    }
  ]
}
```

**Use Case for UI Rendering:**

The template response provides everything needed to dynamically render a configuration form:
- **Field labels and descriptions** for display
- **Field types** to determine input control (text box, checkbox, dropdown, etc.)
- **Default values** to pre-populate forms
- **displayOn** to show/hide fields based on scope level
- **options** to populate dropdown menus

---

### Get Configuration (Published)

Retrieves the **published** (active) configuration for a specific service, group, and scope.

**Endpoint:** `GET /api/v1/config/{serviceName}/scope/{scope}`

**Path Parameters:**
- `serviceName`: Name of the service
- `scope`: One of `SYSTEM`, `PROJECT`, `STORE`, or `USER`

**Query Parameters:**
- `groupId` (required): The configuration group ID
- `projectId` (conditional): Required if scope is `PROJECT`, `STORE`, or `USER`
- `storeId` (conditional): Required if scope is `STORE` or `USER`
- `userId` (conditional): Required if scope is `USER`

**Example Request (SYSTEM scope):**

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/scope/SYSTEM?groupId=payment-methods"
```

**Example Response:**

```json
{
  "serviceName": "payment",
  "scope": "SYSTEM",
  "groupId": "payment-methods",
  "currentVersion": 3,
  "latestVersion": 5,
  "publishedVersion": 3,
  "fields": {
    "methods/credit-card": "true",
    "methods/paypal": "true",
    "methods/stripe": "false"
  },
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-02-20T14:45:00Z"
}
```

**Example Request (USER scope):**

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/scope/USER?groupId=payment-methods&userId=user-456"
```

---

### Update Configuration

Updates or creates configuration values for a specific service, group, and scope. This endpoint is similar to the CLI `set` command and creates a new version with the updated values.

**Endpoint:** `PUT /api/v1/config/{serviceName}/scope/{scope}`

**Path Parameters:**
- `serviceName`: Name of the service
- `scope`: One of `SYSTEM`, `PROJECT`, `STORE`, or `USER`

**Query Parameters:**
- `groupId` (required): The configuration group ID

**Request Body:**

```json
{
  "fields": {
    "path.to.field1": "value1",
    "path.to.field2": "value2"
  },
  "userName": "user@example.com",
  "projectId": "project-123",  // Required for PROJECT, STORE, USER scopes
  "storeId": "store-456",      // Required for STORE, USER scopes
  "userId": "user-789"         // Required for USER scope
}
```

**Example Request (SYSTEM scope):**

```bash
curl -X PUT "http://localhost:8080/api/v1/config/billing-service/scope/SYSTEM?groupId=stripe" \
  -H "Content-Type: application/json" \
  -d '{
    "fields": {
      "stripe.apiKey": "sk_test_123456",
      "stripe.webhookSecret": "whsec_abcdef"
    },
    "userName": "admin@example.com"
  }'
```

**Example Request (PROJECT scope):**

```bash
curl -X PUT "http://localhost:8080/api/v1/config/billing-service/scope/PROJECT?groupId=stripe" \
  -H "Content-Type: application/json" \
  -d '{
    "fields": {
      "stripe.apiKey": "sk_live_project_key"
    },
    "userName": "developer@example.com",
    "projectId": "project-123"
  }'
```

**Example Response:**

```json
{
  "serviceName": "billing-service",
  "scope": "SYSTEM",
  "groupId": "stripe",
  "currentVersion": 4,
  "latestVersion": 4,
  "publishedVersion": 3,
  "fields": {
    "stripe.apiKey": "sk_test_123456",
    "stripe.webhookSecret": "whsec_abcdef"
  },
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-03-15T14:20:00Z",
  "updatedBy": "admin@example.com"
}
```

**Response Details:**
- A new version is created (version 4 in this example)
- `currentVersion` and `latestVersion` are updated to the new version
- `publishedVersion` remains unchanged until explicitly published
- The updated fields are returned along with all other fields from the template defaults

**Use Cases:**
- Setting initial configuration values
- Updating existing configuration values
- Creating configuration overrides at different scopes (PROJECT, STORE, USER)
- Similar to CLI command: `config-cli set --service-name=billing-service --scope=PROJECT --project-id=project-123 --group-id=stripe stripe.apiKey=sk_live_key`

---

### Get Configuration (Latest)

Retrieves the **latest** configuration (including unpublished changes) for a specific service, group, and scope.

**Endpoint:** `GET /api/v1/config/{serviceName}/scope/{scope}/latest`

**Parameters:** Same as [Get Configuration (Published)](#get-configuration-published)

**Example Request:**

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/scope/SYSTEM/latest?groupId=payment-methods"
```

**Example Response:**

```json
{
  "serviceName": "payment",
  "scope": "SYSTEM",
  "groupId": "payment-methods",
  "currentVersion": 5,
  "latestVersion": 5,
  "publishedVersion": 3,
  "fields": {
    "methods/credit-card": "true",
    "methods/paypal": "true",
    "methods/stripe": "true"
  },
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-03-01T09:15:00Z"
}
```

**Note:** `currentVersion` is 5 (latest) while `publishedVersion` is still 3.

---

### Get Version History

Retrieves the version history for a configuration, showing who made changes and when.

**Endpoint:** `GET /api/v1/config/{serviceName}/scope/{scope}/history`

**Query Parameters:**
- `groupId` (required): The configuration group ID
- `projectId`, `storeId`, `userId` (conditional): Based on scope, same as above
- `limit` (optional): Maximum number of history entries to return (default: 10)

**Example Request:**

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/scope/SYSTEM/history?groupId=payment-methods&limit=5"
```

**Example Response:**

```json
{
  "history": [
    {
      "version": 5,
      "createdAt": "2024-03-01T09:15:00Z",
      "createdBy": "alice@example.com"
    },
    {
      "version": 4,
      "createdAt": "2024-02-25T16:30:00Z",
      "createdBy": "bob@example.com"
    },
    {
      "version": 3,
      "createdAt": "2024-02-20T14:45:00Z",
      "createdBy": "alice@example.com"
    },
    {
      "version": 2,
      "createdAt": "2024-02-10T11:20:00Z",
      "createdBy": "charlie@example.com"
    },
    {
      "version": 1,
      "createdAt": "2024-01-15T10:30:00Z",
      "createdBy": "admin@example.com"
    }
  ]
}
```

---

### Publish Configuration

Publishes a specific version of the configuration, making it the active version for client consumption.

**Endpoint:** `POST /api/v1/config/{serviceName}/scope/{scope}/publish`

**Query Parameters:**
- `groupId` (required): The configuration group ID

**Request Body:**

```json
{
  "version": 5,
  "userName": "alice@example.com",
  "projectId": "proj-123",
  "storeId": null,
  "userId": null
}
```

**Request Body Fields:**
- `version` (required): The version number to publish
- `userName` (optional): Name or email of the user publishing. If not provided and authentication is enabled, the email from the JWT token will be used automatically.
- `projectId`, `storeId`, `userId` (conditional): Based on scope

**Example Request:**

```bash
curl -X POST "http://localhost:8080/api/v1/config/payment/scope/SYSTEM/publish?groupId=payment-methods" \
  -H "Content-Type: application/json" \
  -d '{
    "version": 5,
    "userName": "alice@example.com"
  }'
```

**Example Response:**

```json
{
  "serviceName": "payment",
  "scope": "SYSTEM",
  "groupId": "payment-methods",
  "latestVersion": 5,
  "publishedVersion": 5,
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-03-01T11:00:00Z"
}
```

---

## Error Handling

The HTTP Gateway translates gRPC errors to appropriate HTTP status codes with JSON error responses.

### Error Response Format

```json
{
  "error": "Not Found",
  "message": "template not found for service 'unknown-service' and group 'unknown'",
  "code": 404
}
```

### Error Code Mapping

| gRPC Code | HTTP Status | Description |
|-----------|-------------|-------------|
| `NotFound` | 404 | Resource not found |
| `InvalidArgument` | 400 | Invalid request parameters |
| `PermissionDenied` | 403 | Permission denied |
| `AlreadyExists` | 409 | Resource already exists |
| `Internal` | 500 | Internal server error |
| `Unavailable` | 503 | Service unavailable |
| `Unauthenticated` | 401 | Authentication required |

### Common Error Scenarios

**Missing Required Parameter:**
```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/template"
```

Response (400 Bad Request):
```json
{
  "error": "Internal Server Error",
  "message": "groupId query parameter is required",
  "code": 500
}
```

**Invalid Scope:**
```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/scope/INVALID?groupId=payment-methods"
```

Response (400 Bad Request):
```json
{
  "error": "Internal Server Error",
  "message": "invalid scope: must be one of SYSTEM, PROJECT, STORE, USER",
  "code": 500
}
```

**Template Not Found:**
```bash
curl -X GET "http://localhost:8080/api/v1/config/unknown-service/template?groupId=unknown"
```

Response (404 Not Found):
```json
{
  "error": "Not Found",
  "message": "template not found for service 'unknown-service' and group 'unknown'",
  "code": 404
}
```

---

## Complete Workflow Examples

### Workflow 1: Building a Configuration UI

This workflow demonstrates how a frontend application would use the API to render a dynamic configuration form.

**Step 1: Fetch the template to understand available fields**

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/template?groupId=payment-methods"
```

The response tells you:
- What fields are available
- What type each field is (for rendering the right input control)
- Default values
- Which scopes can configure each field
- Options for dropdown fields

**Step 2: Fetch current configuration values**

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/scope/SYSTEM?groupId=payment-methods"
```

This returns the current published values, which you use to pre-populate the form.

**Step 3: User edits values in the UI and saves**

This would typically call the gRPC `UpdateConfig` method (not exposed in HTTP gateway yet, as it modifies the service). The update creates a new version.

**Step 4: Review version history**

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/scope/SYSTEM/history?groupId=payment-methods"
```

**Step 5: Publish the new version**

```bash
curl -X POST "http://localhost:8080/api/v1/config/payment/scope/SYSTEM/publish?groupId=payment-methods" \
  -H "Content-Type: application/json" \
  -d '{
    "version": 6,
    "userName": "alice@example.com"
  }'
```

---

### Workflow 2: Multi-Scope Configuration

This example shows how configuration can be set at different scope levels.

**System-Level Configuration (Defaults for all):**

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/scope/SYSTEM?groupId=payment-methods"
```

**User-Level Configuration (User-specific settings):**

```bash
curl -X GET "http://localhost:8080/api/v1/config/payment/scope/USER?groupId=payment-methods&userId=user-789"
```

*Note: Other services (like `api-gateway`) may support additional scopes like PROJECT or STORE if defined in their template.*


---

### Workflow 3: API Gateway Rate Limiting (Project Scope)

This workflow demonstrates how **infrastructure settings** (like rate limits) often use the `PROJECT` scope to allow overrides for specific applications.

**Step 1: Check System Defaults**

```bash
curl -X GET "http://localhost:8080/api/v1/config/api-gateway/scope/SYSTEM?groupId=default-rate-limit"
```

Response (Global Defaults):
```json
{
  "serviceName": "api-gateway",
  "scope": "SYSTEM",
  "groupId": "default-rate-limit",
  "fields": {
    "capacity": "1000",
    "is-active": "true",
    "refill-duration-in-seconds": "10",
    "refill-tokens": "100",
    "url-pattern": "/**"
  }
}
```

**Step 2: Check Project-Specific Override**

High-traffic projects can have their own configuration group overrides.

```bash
curl -X GET "http://localhost:8080/api/v1/config/api-gateway/scope/PROJECT?groupId=default-rate-limit&projectId=high-traffic-app"
```

Response (Project Override):
```json
{
  "serviceName": "api-gateway",
  "scope": "PROJECT",
  "groupId": "default-rate-limit",
  "projectId": "high-traffic-app",
  "fields": {
    "capacity": "5000",
    "is-active": "true",
    "refill-duration-in-seconds": "10",
    "refill-tokens": "500",
    "url-pattern": "/**"
  }
}
```
*Note: The `capacity` (5000) and `refill-tokens` (500) are higher than the system defaults.*

---


## Testing the API

You can use the provided examples with `curl`, or use tools like:
- **Postman**: Import the examples as a collection
- **HTTPie**: `http GET localhost:8080/api/v1/config/payment/template groupId==payment-methods`
- **Browser**: For GET requests, simply paste the URL

### Health Check

The gateway doesn't currently expose a health endpoint, but you can verify it's running by making any valid request.

---

## Architecture Notes

- **Unified Service**: Both gRPC and HTTP run in the same container/process. The HTTP gateway connects to the gRPC service on localhost.
- **No Modification to gRPC Service**: The HTTP gateway is a pure wrapper. It doesn't modify the existing gRPC service implementation.
- **Stateless**: The gateway maintains no state; it simply forwards requests to the local gRPC service.
- **Error Translation**: gRPC errors are automatically translated to appropriate HTTP status codes.
- **JSON Only**: All requests and responses use JSON format.
- **Lightweight**: Built with Chi router for minimal overhead.

---

## Next Steps

For information about:
- Setting up the gRPC service: See [README.md](../README.md)
- Configuration templates: See [templates/](../templates/)
- gRPC API details: See [proto/config/v1/config.proto](../proto/config/v1/config.proto)
