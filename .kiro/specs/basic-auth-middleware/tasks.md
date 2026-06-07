# Tasks

## Task 1: Implement BasicAuthMiddleware struct and constructor

- [x] 1.1 Create `pkg/httpgateway/basic_auth.go` with `BasicAuthMiddleware` struct containing `username`, `password`, `credentialsConfigured`, and `realm` fields
- [x] 1.2 Implement `NewBasicAuthMiddleware(username, password string) *BasicAuthMiddleware` that sets `credentialsConfigured = true` only when both username and password are non-empty, and sets realm to `"ScopeConfig"`
- [x] 1.3 Implement `writeBasicAuthError(w http.ResponseWriter, message string)` helper that writes a JSON error response with HTTP 401 status using the existing `ErrorResponse` format

## Task 2: Implement BasicAuthMiddleware.Handler method

- [x] 2.1 Implement `Handler(next http.Handler) http.Handler` method that returns immediately via `next.ServeHTTP` when `credentialsConfigured` is false (pass-through mode)
- [x] 2.2 In enforced mode, parse the `Authorization` header using `r.BasicAuth()` and return 401 with `WWW-Authenticate` header when missing or invalid
- [x] 2.3 Compare credentials using `crypto/subtle.ConstantTimeCompare` for both username and password, returning 401 on mismatch
- [x] 2.4 Call `next.ServeHTTP(w, r)` when credentials match

## Task 3: Modify RouterConfig and NewRouterWithConfig for protected route groups

- [x] 3.1 Add `BasicAuthMiddleware *BasicAuthMiddleware` field to `RouterConfig` struct in `pkg/httpgateway/router.go`
- [x] 3.2 Refactor `NewRouterWithConfig` to mount `/swagger/*` on the root router (outside any auth group)
- [x] 3.3 Create a protected `r.Group()` that applies `BasicAuthMiddleware.Handler` (if non-nil) then `AuthMiddleware.Middleware` (if non-nil)
- [x] 3.4 Move `/admin`, `/api/v1/config/templates`, and `/api/v1/config/{serviceName}/**` routes into the protected group

## Task 4: Wire BasicAuthMiddleware into server startup

- [x] 4.1 In `cmd/httpgateway/main.go` (or `cmd/server/main.go`), read `AUTH_USER` and `AUTH_PASSWORD` from environment variables
- [x] 4.2 Create `BasicAuthMiddleware` instance and pass it to `RouterConfig`
- [x] 4.3 Log whether Basic Auth is enabled or running in open mode

## Task 5: Unit tests for BasicAuthMiddleware

- [x] 5.1 Create `pkg/httpgateway/basic_auth_test.go` with `TestNewBasicAuthMiddleware_CredentialsConfigured` — verify `credentialsConfigured` is true when both args are non-empty, false otherwise
- [x] 5.2 Add `TestBasicAuth_PassThroughMode` — verify requests with and without Authorization headers pass through when credentials are not configured
- [x] 5.3 Add `TestBasicAuth_ValidCredentials` — verify 200 when correct Basic Auth header is sent in enforced mode
- [x] 5.4 Add `TestBasicAuth_InvalidCredentials` — verify 401 + `WWW-Authenticate` header when wrong credentials are sent
- [x] 5.5 Add `TestBasicAuth_MissingHeader` — verify 401 + `WWW-Authenticate` header when no Authorization header is present in enforced mode
- [x] 5.6 Add `TestBasicAuth_MalformedHeader` — verify 401 when Authorization header is not valid Basic Auth format
- [x] 5.7 Add `TestBasicAuth_ResponseIsJSON` — verify rejected responses have valid JSON body with error message

## Task 6: Integration tests for route protection

- [x] 6.1 Add `TestRouter_SwaggerPublicWithBasicAuth` — verify `/swagger/*` returns 200 without credentials when BasicAuthMiddleware is configured
- [x] 6.2 Add `TestRouter_ProtectedRoutesRequireBasicAuth` — verify `/admin`, `/api/v1/config/templates`, and `/api/v1/config/{serviceName}/template` return 401 without credentials
- [x] 6.3 Add `TestRouter_ProtectedRoutesAccessibleWithBasicAuth` — verify protected routes return 200 with valid Basic Auth credentials
- [x] 6.4 Add `TestRouter_NoAuthConfigured` — verify all routes are accessible when neither BasicAuthMiddleware nor AuthMiddleware is configured

## Task 7: Property-based tests

- [x] 7.1 [PBT] Property 1: Initialization mode — *For any* pair of strings (username, password), `credentialsConfigured` is true iff both are non-empty (Validates: Requirements 1.1, 1.2, 1.3)
- [x] 7.2 [PBT] Property 2: Pass-through forwards all — *For any* request with any Authorization header value, pass-through mode always invokes the next handler and never returns 401 (Validates: Requirements 2.1, 2.2)
- [x] 7.3 [PBT] Property 3: Valid credentials grant access — *For any* non-empty credential pair, a request carrying matching Basic Auth always reaches the next handler (Validates: Requirement 3.1)
- [x] 7.4 [PBT] Property 4: Invalid credentials are rejected — *For any* two distinct credential pairs where at least one field differs, the middleware returns 401 with WWW-Authenticate header (Validates: Requirements 3.2, 3.3, 3.4, 7.1, 7.2)

## Task 8: Update documentation and environment variable references

- [x] 8.1 Add `AUTH_USER` and `AUTH_PASSWORD` to `.env.example`
- [x] 8.2 Update `docs/HTTP_GATEWAY.md` environment variables table to include `AUTH_USER` and `AUTH_PASSWORD`
- [x] 8.3 Add Basic Auth section to `docs/HTTP_GATEWAY.md` Authentication documentation
