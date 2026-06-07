# Requirements Document

## Introduction

This document defines the requirements for adding HTTP Basic Authentication middleware to the Scope Config Service HTTP Gateway. The middleware provides an opt-in authentication layer that protects admin UI, admin API, and config API routes using credentials sourced from environment variables. When credentials are not configured, the middleware acts as a transparent pass-through, supporting internal deployments where the service sits behind an upstream gateway or runs on a private network.

## Glossary

- **BasicAuthMiddleware**: The HTTP middleware component that validates Basic Authentication credentials against environment-configured values
- **Protected_Routes**: The set of HTTP routes requiring authentication: `/admin`, `/api/v1/config/templates`, and `/api/v1/config/{serviceName}/**`
- **Pass_Through_Mode**: The middleware operating mode when `AUTH_USER` or `AUTH_PASSWORD` is empty or unset, allowing all traffic without authentication
- **Enforced_Mode**: The middleware operating mode when both `AUTH_USER` and `AUTH_PASSWORD` are set and non-empty, requiring valid Basic Auth credentials
- **Router**: The Chi HTTP router that dispatches requests to handlers
- **AuthMiddleware**: The existing Keycloak JWT authentication middleware
- **Swagger_Routes**: The `/swagger/*` endpoints serving API documentation

## Requirements

### Requirement 1: Middleware Initialization

**User Story:** As a system operator, I want the Basic Auth middleware to configure itself from environment variables at startup, so that I can control authentication behavior through deployment configuration.

#### Acceptance Criteria

1. WHEN `AUTH_USER` is non-empty AND `AUTH_PASSWORD` is non-empty, THE BasicAuthMiddleware SHALL operate in Enforced_Mode
2. WHEN `AUTH_USER` is empty, THE BasicAuthMiddleware SHALL operate in Pass_Through_Mode regardless of `AUTH_PASSWORD` value
3. WHEN `AUTH_PASSWORD` is empty, THE BasicAuthMiddleware SHALL operate in Pass_Through_Mode regardless of `AUTH_USER` value
4. THE BasicAuthMiddleware SHALL set the WWW-Authenticate realm to "ScopeConfig"

### Requirement 2: Pass-Through Mode (Internal/Gateway Deployment)

**User Story:** As a system operator deploying the service behind an API gateway or on a private network, I want the middleware to allow all traffic when credentials are not configured, so that I do not need to set up Basic Auth for internal deployments.

#### Acceptance Criteria

1. WHILE the BasicAuthMiddleware is in Pass_Through_Mode, THE BasicAuthMiddleware SHALL forward all requests to the next handler without inspecting the Authorization header
2. WHILE the BasicAuthMiddleware is in Pass_Through_Mode, THE BasicAuthMiddleware SHALL return no authentication-related error responses

### Requirement 3: Credential Enforcement

**User Story:** As a system operator, I want the middleware to enforce Basic Auth on protected routes when credentials are configured, so that unauthorized users cannot access admin and config endpoints.

#### Acceptance Criteria

1. WHILE the BasicAuthMiddleware is in Enforced_Mode, WHEN a request includes a valid `Authorization: Basic` header with matching credentials, THE BasicAuthMiddleware SHALL forward the request to the next handler
2. WHILE the BasicAuthMiddleware is in Enforced_Mode, WHEN a request includes an `Authorization: Basic` header with non-matching credentials, THE BasicAuthMiddleware SHALL respond with HTTP 401 and set the `WWW-Authenticate: Basic realm="ScopeConfig"` header
3. WHILE the BasicAuthMiddleware is in Enforced_Mode, WHEN a request lacks an `Authorization` header, THE BasicAuthMiddleware SHALL respond with HTTP 401 and set the `WWW-Authenticate: Basic realm="ScopeConfig"` header
4. WHILE the BasicAuthMiddleware is in Enforced_Mode, WHEN a request includes a malformed `Authorization` header that is not valid Basic Auth, THE BasicAuthMiddleware SHALL respond with HTTP 401 and set the `WWW-Authenticate: Basic realm="ScopeConfig"` header

### Requirement 4: Timing-Safe Credential Comparison

**User Story:** As a security engineer, I want credential comparison to use constant-time algorithms, so that the system is resistant to timing side-channel attacks.

#### Acceptance Criteria

1. THE BasicAuthMiddleware SHALL compare the username using `crypto/subtle.ConstantTimeCompare`
2. THE BasicAuthMiddleware SHALL compare the password using `crypto/subtle.ConstantTimeCompare`

### Requirement 5: Route Protection Configuration

**User Story:** As a system operator, I want Swagger documentation to remain publicly accessible while admin and config routes are protected, so that API discoverability is not hindered by authentication.

#### Acceptance Criteria

1. THE Router SHALL serve Swagger_Routes without applying BasicAuthMiddleware
2. WHEN BasicAuthMiddleware is provided, THE Router SHALL apply BasicAuthMiddleware to the `/admin` route
3. WHEN BasicAuthMiddleware is provided, THE Router SHALL apply BasicAuthMiddleware to the `/api/v1/config/templates` routes
4. WHEN BasicAuthMiddleware is provided, THE Router SHALL apply BasicAuthMiddleware to the `/api/v1/config/{serviceName}/**` routes

### Requirement 6: Middleware Composability

**User Story:** As a system operator, I want Basic Auth and Keycloak JWT authentication to coexist, so that I can layer multiple authentication mechanisms as needed.

#### Acceptance Criteria

1. WHEN both BasicAuthMiddleware and AuthMiddleware are configured, THE Router SHALL apply BasicAuthMiddleware before AuthMiddleware on Protected_Routes
2. WHEN only BasicAuthMiddleware is configured, THE Router SHALL apply BasicAuthMiddleware to Protected_Routes without requiring AuthMiddleware
3. WHEN only AuthMiddleware is configured, THE Router SHALL apply AuthMiddleware to Protected_Routes without requiring BasicAuthMiddleware
4. WHEN neither BasicAuthMiddleware nor AuthMiddleware is configured, THE Router SHALL serve Protected_Routes without any authentication

### Requirement 7: Error Response Format

**User Story:** As an API consumer, I want authentication error responses to follow a consistent JSON format, so that I can programmatically handle authentication failures.

#### Acceptance Criteria

1. WHEN the BasicAuthMiddleware rejects a request, THE BasicAuthMiddleware SHALL return a JSON response body containing an error message
2. WHEN the BasicAuthMiddleware rejects a request, THE BasicAuthMiddleware SHALL include the `WWW-Authenticate: Basic realm="ScopeConfig"` response header

### Requirement 8: Server Startup Integration

**User Story:** As a system operator, I want the server to read `AUTH_USER` and `AUTH_PASSWORD` from environment variables at startup and wire the middleware into the router, so that the feature works without code changes at deployment time.

#### Acceptance Criteria

1. WHEN the HTTP gateway starts, THE Server SHALL read `AUTH_USER` and `AUTH_PASSWORD` from environment variables
2. WHEN the HTTP gateway starts, THE Server SHALL create a BasicAuthMiddleware instance and pass it to the RouterConfig
3. WHEN Basic Auth is active, THE Server SHALL log a message indicating Basic Auth is enabled
4. WHEN Basic Auth is inactive, THE Server SHALL log a message indicating the service is running in open mode
