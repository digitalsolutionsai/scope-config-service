# Design Document: Basic Auth Middleware

## Overview

This feature adds HTTP Basic Authentication middleware to the Scope Config Service HTTP Gateway. The middleware protects the admin UI, admin API routes, and all `/api/v1/config/` endpoints by requiring valid credentials supplied via the standard `Authorization: Basic <base64>` header.

Credentials are sourced from the `AUTH_USER` and `AUTH_PASSWORD` environment variables. The design follows an "opt-in auth" principle: if either variable is missing or empty, the middleware acts as a transparent pass-through, allowing all traffic without authentication. This supports internal deployments where the service sits behind a gateway or is used on a private network. When both variables are set and non-empty, the middleware enforces Basic Auth on all protected routes. Swagger docs remain publicly accessible for discoverability regardless of configuration.

The middleware coexists with the existing Keycloak JWT `AuthMiddleware`. Basic Auth runs as an outer layer on the chi router, applied selectively to protected route groups, while the existing JWT middleware continues to operate as before when configured.

## Architecture

```mermaid
graph TD
    Client[HTTP Client] --> Router[Chi Router]
    Router --> SwaggerRoutes["/swagger/*<br/>(No Auth)"]
    Router --> ProtectedGroup["Protected Route Group<br/>(BasicAuth Middleware)"]
    ProtectedGroup --> AdminUI["/admin<br/>Admin UI"]
    ProtectedGroup --> AdminAPI["/api/v1/config/templates<br/>Admin API"]
    ProtectedGroup --> ConfigAPI["/api/v1/config/{serviceName}/**<br/>Config API"]
    
    subgraph BasicAuthMiddleware
        CheckEnv{AUTH_USER &<br/>AUTH_PASSWORD set?}
        CheckEnv -->|No| PassThrough[next.ServeHTTP<br/>Allow All Traffic]
        CheckEnv -->|Yes| ParseHeader{Parse Basic Auth<br/>Header}
        ParseHeader -->|Missing/Invalid| Deny401[401 Unauthorized<br/>+ WWW-Authenticate]
        ParseHeader -->|Valid| CompareCredentials{Constant-Time<br/>Compare}
        CompareCredentials -->|Mismatch| Deny401
        CompareCredentials -->|Match| Next[next.ServeHTTP]
    end
```

## Sequence Diagrams

### Successful Authentication Flow

```mermaid
sequenceDiagram
    participant C as HTTP Client
    participant R as Chi Router
    participant BA as BasicAuthMiddleware
    participant H as Handler (Admin/Config)

    C->>R: GET /api/v1/config/templates<br/>Authorization: Basic dXNlcjpwYXNz
    R->>BA: Middleware(next)
    BA->>BA: Check credentialsConfigured == true
    BA->>BA: r.BasicAuth() → (user, pass, ok)
    BA->>BA: subtle.ConstantTimeCompare(user, expectedUser)
    BA->>BA: subtle.ConstantTimeCompare(pass, expectedPass)
    BA-->>H: next.ServeHTTP(w, r)
    H-->>C: 200 OK + JSON body
```

### Pass-Through — Credentials Not Configured (Internal/Gateway Mode)

```mermaid
sequenceDiagram
    participant C as HTTP Client
    participant R as Chi Router
    participant BA as BasicAuthMiddleware
    participant H as Handler (Admin/Config)

    C->>R: GET /api/v1/config/templates
    R->>BA: Middleware(next)
    BA->>BA: Check credentialsConfigured == false
    BA-->>H: next.ServeHTTP(w, r) — pass-through
    H-->>C: 200 OK + JSON body
```

### Denied — Wrong Credentials

```mermaid
sequenceDiagram
    participant C as HTTP Client
    participant R as Chi Router
    participant BA as BasicAuthMiddleware

    C->>R: GET /admin<br/>Authorization: Basic d3Jvbmc6Y3JlZHM=
    R->>BA: Middleware(next)
    BA->>BA: credentialsConfigured == true
    BA->>BA: r.BasicAuth() → ("wrong", "creds", true)
    BA->>BA: ConstantTimeCompare fails
    BA-->>C: 401 Unauthorized<br/>WWW-Authenticate: Basic realm="ScopeConfig"
```

## Components and Interfaces

### Component 1: BasicAuthMiddleware

**Purpose**: Validates HTTP Basic Auth credentials against environment-configured values. Acts as a pass-through when credentials are not configured (internal/gateway deployment mode).

```go
// BasicAuthMiddleware provides HTTP Basic Authentication.
type BasicAuthMiddleware struct {
    username              string
    password              string
    credentialsConfigured bool   // false when AUTH_USER or AUTH_PASSWORD is empty
    realm                 string // WWW-Authenticate realm, default "ScopeConfig"
}
```

**Interface**:

```go
// NewBasicAuthMiddleware creates a BasicAuthMiddleware.
// If username or password is empty, credentialsConfigured is set to false
// and the middleware will pass through all requests (internal/gateway mode).
func NewBasicAuthMiddleware(username, password string) *BasicAuthMiddleware

// Handler returns an http.Handler middleware function compatible with chi.Use().
func (b *BasicAuthMiddleware) Handler(next http.Handler) http.Handler
```

**Responsibilities**:
- Parse `Authorization: Basic <base64>` header via `r.BasicAuth()`
- Constant-time comparison of username and password using `crypto/subtle`
- Return `401 Unauthorized` with `WWW-Authenticate: Basic realm="ScopeConfig"` on failure
- Pass through all requests when credentials are not configured (internal/gateway mode)

### Component 2: RouterConfig (Extended)

**Purpose**: Extended to accept an optional `BasicAuthMiddleware` alongside the existing `AuthMiddleware`.

```go
type RouterConfig struct {
    Client              configv1.ConfigServiceClient
    AuthMiddleware      *AuthMiddleware       // existing Keycloak JWT (unchanged)
    BasicAuthMiddleware *BasicAuthMiddleware   // NEW: Basic Auth
    DB                  *sql.DB
}
```

### Component 3: NewRouterWithConfig (Modified)

**Purpose**: Applies `BasicAuthMiddleware` to a protected route group containing admin and config routes, while leaving `/swagger/*` unprotected.

**Responsibilities**:
- Mount `/swagger/*` on the root router (no auth)
- Create a sub-group for all protected routes
- Apply `BasicAuthMiddleware.Handler` to the protected group if provided
- Apply existing `AuthMiddleware.Middleware` to the protected group if provided
- Mount admin UI, admin API, and config API routes inside the protected group

## Data Models

No new persistent data models. The middleware is stateless and reads credentials from environment variables at startup.

### Configuration Model

| Environment Variable | Required | Default | Description |
|---------------------|----------|---------|-------------|
| `AUTH_USER`         | Yes (for auth) | _(empty)_ | Basic Auth username |
| `AUTH_PASSWORD`     | Yes (for auth) | _(empty)_ | Basic Auth password |

When both are set and non-empty, Basic Auth is active. When either is missing or empty, the middleware acts as a pass-through and allows all traffic (suitable for internal/gateway deployments).

## Key Functions with Formal Specifications

### Function 1: NewBasicAuthMiddleware

```go
func NewBasicAuthMiddleware(username, password string) *BasicAuthMiddleware
```

**Preconditions:**
- `username` and `password` are strings (may be empty)

**Postconditions:**
- If `username != ""` AND `password != ""`: returns middleware with `credentialsConfigured = true`
- If `username == ""` OR `password == ""`: returns middleware with `credentialsConfigured = false`
- `realm` is always set to `"ScopeConfig"`

### Function 2: BasicAuthMiddleware.Handler

```go
func (b *BasicAuthMiddleware) Handler(next http.Handler) http.Handler
```

**Preconditions:**
- `next` is a valid `http.Handler`
- `b` is a properly initialized `BasicAuthMiddleware`

**Postconditions:**
- If `b.credentialsConfigured == false`: calls `next.ServeHTTP(w, r)` immediately (pass-through, no auth required)
- If request has no `Authorization` header or invalid Basic Auth: responds with HTTP 401, sets `WWW-Authenticate: Basic realm="ScopeConfig"`, does NOT call `next`
- If credentials match (constant-time): calls `next.ServeHTTP(w, r)`
- If credentials do not match: responds with HTTP 401, sets `WWW-Authenticate: Basic realm="ScopeConfig"`, does NOT call `next`

**Loop Invariants:** N/A (no loops)

### Function 3: NewRouterWithConfig (Modified)

```go
func NewRouterWithConfig(config RouterConfig) *chi.Mux
```

**Preconditions:**
- `config.Client` is a valid gRPC client
- `config.BasicAuthMiddleware` may be nil (no basic auth applied)
- `config.AuthMiddleware` may be nil (no JWT auth applied)

**Postconditions:**
- `/swagger/*` is always accessible without any authentication
- If `config.BasicAuthMiddleware != nil`: all admin and config routes require Basic Auth
- If `config.AuthMiddleware != nil`: JWT auth is applied inside the protected group (after Basic Auth)
- All `/admin`, `/api/v1/config/templates`, and `/api/v1/config/{serviceName}/**` routes are inside the protected group

## Algorithmic Pseudocode

### Basic Auth Middleware Handler

```go
func (b *BasicAuthMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // STEP 1: Pass-through if credentials not configured (internal/gateway mode)
        if !b.credentialsConfigured {
            next.ServeHTTP(w, r)
            return
        }

        // STEP 2: Extract Basic Auth from request
        username, password, ok := r.BasicAuth()
        if !ok {
            w.Header().Set("WWW-Authenticate", `Basic realm="`+b.realm+`"`)
            writeBasicAuthError(w, "missing or invalid Authorization header")
            return
        }

        // STEP 3: Constant-time comparison to prevent timing attacks
        usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(b.username)) == 1
        passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(b.password)) == 1

        if !usernameMatch || !passwordMatch {
            w.Header().Set("WWW-Authenticate", `Basic realm="`+b.realm+`"`)
            writeBasicAuthError(w, "invalid credentials")
            return
        }

        // STEP 4: Credentials valid — proceed
        next.ServeHTTP(w, r)
    })
}
```

### Router Construction with Protected Group

```go
func NewRouterWithConfig(config RouterConfig) *chi.Mux {
    gateway := NewGateway(config.Client)
    adminGW := NewAdminGateway(config.Client, config.DB)
    r := chi.NewRouter()

    // Global middleware (logging, recovery, etc.)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)

    // Public routes — no auth
    r.Get("/swagger/*", httpSwagger.Handler(
        httpSwagger.URL("/swagger/doc.json"),
    ))

    // Protected route group
    r.Group(func(r chi.Router) {
        // Apply Basic Auth if configured
        if config.BasicAuthMiddleware != nil {
            r.Use(config.BasicAuthMiddleware.Handler)
        }

        // Apply existing JWT auth if configured
        if config.AuthMiddleware != nil {
            r.Use(config.AuthMiddleware.Middleware)
        }

        // Admin UI
        r.Get("/admin", ServeAdminUI)
        r.Get("/admin/", ServeAdminUI)

        // Admin API routes
        r.Get("/api/v1/config/templates", adminGW.ListAllTemplates)
        r.Post("/api/v1/config/templates", adminGW.ImportTemplate)
        r.Patch("/api/v1/config/templates/{serviceName}/{groupId}/active", adminGW.SetTemplateActive)

        // Config API routes
        r.Route("/api/v1/config/{serviceName}", func(r chi.Router) {
            r.Get("/template", gateway.GetTemplate)
            r.Get("/scope/{scope}", gateway.GetConfig)
            r.Put("/scope/{scope}", gateway.UpdateConfig)
            r.Get("/scope/{scope}/latest", gateway.GetLatestConfig)
            r.Get("/scope/{scope}/version/{version}", gateway.GetConfigByVersion)
            r.Get("/scope/{scope}/history", gateway.GetConfigHistory)
            r.Post("/scope/{scope}/publish", gateway.PublishConfig)
        })
    })

    return r
}
```

## Example Usage

### Creating the middleware in cmd/server/main.go

```go
// Read Basic Auth credentials from environment
authUser := os.Getenv("AUTH_USER")
authPassword := os.Getenv("AUTH_PASSWORD")
basicAuth := httpgateway.NewBasicAuthMiddleware(authUser, authPassword)

if authUser != "" && authPassword != "" {
    log.Println("Basic Auth enabled for protected routes")
} else {
    log.Println("INFO: AUTH_USER/AUTH_PASSWORD not set — running in open mode (internal/gateway deployment)")
}

router := httpgateway.NewRouterWithConfig(httpgateway.RouterConfig{
    Client:              client,
    BasicAuthMiddleware: basicAuth,
    DB:                  db,
})
```

### Client request with Basic Auth

```bash
# Successful request
curl -u admin:secret123 http://localhost:8080/api/v1/config/templates

# Or with explicit header
curl -H "Authorization: Basic YWRtaW46c2VjcmV0MTIz" http://localhost:8080/api/v1/config/templates

# Swagger remains accessible without auth
curl http://localhost:8080/swagger/index.html
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system — essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Initialization mode is determined by credential presence

*For any* pair of strings (username, password), `NewBasicAuthMiddleware(username, password).credentialsConfigured` is `true` if and only if both username and password are non-empty.

**Validates: Requirements 1.1, 1.2, 1.3**

### Property 2: Pass-through mode forwards all requests

*For any* request (with or without an Authorization header), when the BasicAuthMiddleware is in Pass_Through_Mode, the next handler is always invoked and the response status is never 401.

**Validates: Requirements 2.1, 2.2**

### Property 3: Valid credentials grant access

*For any* non-empty credential pair (user, pass), when the BasicAuthMiddleware is initialized with that pair and a request carries `Authorization: Basic base64(user + ":" + pass)`, the next handler is invoked.

**Validates: Requirement 3.1**

### Property 4: Invalid credentials are rejected

*For any* two distinct credential pairs (configuredUser, configuredPass) and (requestUser, requestPass) where at least one field differs, the BasicAuthMiddleware in Enforced_Mode responds with HTTP 401 and sets the `WWW-Authenticate` header.

**Validates: Requirements 3.2, 3.3, 3.4, 7.1, 7.2**

### Property 5: Rejection responses contain well-formed JSON

*For any* rejected request (missing header, malformed header, or wrong credentials), the response body is valid JSON containing an error message field.

**Validates: Requirement 7.1**

## Error Handling

### Scenario 1: Credentials Not Configured (Internal/Gateway Mode)

**Condition**: `AUTH_USER` or `AUTH_PASSWORD` environment variable is empty or unset at startup
**Behavior**: Middleware acts as a pass-through — all requests to protected routes are forwarded to handlers without authentication
**Use case**: Service is deployed behind a gateway or on a private network where external auth is handled upstream

### Error Scenario 2: Missing Authorization Header

**Condition**: Client sends request to protected route without `Authorization` header
**Response**: HTTP 401 with `WWW-Authenticate: Basic realm="ScopeConfig"` header and JSON body
**Recovery**: Client retries with `Authorization: Basic <base64(user:pass)>` header

### Error Scenario 3: Invalid Credentials

**Condition**: Client provides wrong username or password
**Response**: HTTP 401 with `WWW-Authenticate: Basic realm="ScopeConfig"` header and JSON body
**Recovery**: Client retries with correct credentials

## Testing Strategy

### Unit Testing Approach

Test the `BasicAuthMiddleware` in isolation using `httptest`:

1. **TestBasicAuth_CredentialsNotConfigured**: Create middleware with empty username → verify all requests pass through (200)
2. **TestBasicAuth_ValidCredentials**: Create middleware with known creds → send matching Basic Auth → verify 200
3. **TestBasicAuth_InvalidCredentials**: Send wrong creds → verify 401 + WWW-Authenticate header
4. **TestBasicAuth_MissingHeader**: Send no Authorization header → verify 401 + WWW-Authenticate header
5. **TestBasicAuth_EmptyPassword**: Create middleware with empty password → verify all requests pass through (200)
6. **TestBasicAuth_EmptyUsername**: Create middleware with empty username → verify all requests pass through (200)

### Integration Testing Approach

Test the router with BasicAuthMiddleware wired in:

1. Verify `/swagger/index.html` is accessible without auth
2. Verify `/api/v1/config/templates` requires Basic Auth
3. Verify `/admin` requires Basic Auth
4. Verify `/api/v1/config/{serviceName}/template` requires Basic Auth

## Security Considerations

- **Constant-time comparison**: Uses `crypto/subtle.ConstantTimeCompare` to prevent timing attacks on credential validation
- **Open by default for internal use**: Missing credentials = pass-through, suitable for services behind a gateway or on private networks
- **No credential logging**: Middleware must never log the password or the full Authorization header
- **HTTPS recommended**: Basic Auth transmits credentials in base64 (not encrypted). Production deployments should use TLS termination at the load balancer or reverse proxy
- **Realm header**: `WWW-Authenticate: Basic realm="ScopeConfig"` prompts browser-based clients for credentials

## Dependencies

- `crypto/subtle` (Go stdlib) — constant-time byte comparison
- `net/http` (Go stdlib) — `r.BasicAuth()` for parsing the Authorization header
- `github.com/go-chi/chi/v5` — route grouping with `r.Group()`
- No new external dependencies required
