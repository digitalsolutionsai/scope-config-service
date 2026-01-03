# Go SDK TODO & Enhancement Plan

## 📋 Overview

This document outlines the tasks needed to enhance the Go SDK for the ScopeConfig service, including testing improvements, integration testing, packaging for remote installation, and bug fixes.

---

## 🎯 High Priority Tasks

### 1. Package Distribution & Installation

#### 1.1 Make SDK Installable from GitHub
**Status**: ❌ Not Done  
**Priority**: HIGH  
**Effort**: Medium

**Current State**:
- SDK uses hardcoded import paths: `github.com/digitalsolutionsai/scope-config-service/sdks/go`
- Users must copy the entire SDK directory and manually update import paths
- No versioning or release management

**Tasks**:
- [ ] Create a standalone Go module structure
- [ ] Set up proper `go.mod` with correct module path
- [ ] Add version tags (e.g., `v1.0.0`, `v1.0.1`)
- [ ] Update README with installation instructions:
  ```go
  go get github.com/digitalsolutionsai/scope-config-service/sdks/go@v1.0.0
  ```
- [ ] Test installation from GitHub in a separate project
- [ ] Document private repository access (if needed):
  ```bash
  git config --global url."git@github.com:".insteadOf "https://github.com/"
  # Or use GOPRIVATE
  export GOPRIVATE=github.com/digitalsolutionsai/*
  ```

**Files to Update**:
- `go.mod` - Ensure correct module path
- `README.md` - Add installation section
- All `.go` files - Verify import paths are correct

---

### 1.2 Generated Proto Files Management
**Status**: ⚠️ Needs Decision  
**Priority**: HIGH  
**Effort**: Low

**Current State**:
- `gen/` directory is NOT in `.gitignore` (proto files ARE committed)
- `proto/` directory IS in `.gitignore` (source proto files are NOT committed)
- Users must copy proto files and run `buf generate`

**Decision Needed**:
**Option A**: Commit generated files (like TypeScript SDK)
- ✅ Pros: Users can `go get` and use immediately
- ❌ Cons: Larger repo, merge conflicts on proto changes

**Option B**: Don't commit generated files (current approach)
- ✅ Pros: Cleaner repo
- ❌ Cons: Users need buf toolchain

**Recommended**: **Option A** - Commit `gen/` directory

**Tasks**:
- [ ] Decide on approach
- [ ] If Option A: Remove `gen/` from `.gitignore`, commit generated files
- [ ] If Option B: Add `gen/` to `.gitignore`, document buf requirement
- [ ] Update README with clear instructions

---

## 🧪 Testing & Quality

### 2. Integration Testing

#### 2.1 Comprehensive Integration Test Suite
**Status**: ⚠️ Partial (only 1 test exists)  
**Priority**: HIGH  
**Effort**: High

**Current State**:
- Only one integration test: `TestApplyAndGetTemplate`
- No tests for caching, inheritance, or error scenarios
- Tests require manual server setup

**Tasks**:
- [ ] **Test Coverage**:
  - [ ] GetConfig / GetConfigCached
  - [ ] GetLatestConfig
  - [ ] UpdateConfig
  - [ ] GetValue with inheritance
  - [ ] GetValue with default values
  - [ ] Template loading from YAML files
  - [ ] Cache invalidation
  - [ ] Background sync
  - [ ] Error handling (server unavailable, invalid requests)
  - [ ] Concurrent access scenarios

- [ ] **Test Infrastructure**:
  - [ ] Add testcontainers-go for automated server setup
  - [ ] Create test fixtures (sample templates, configs)
  - [ ] Add table-driven tests for different scopes
  - [ ] Mock gRPC server for unit tests

- [ ] **Test Organization**:
  - [ ] Separate unit tests from integration tests
  - [ ] Add build tags: `//go:build integration`
  - [ ] Create `Makefile` targets:
    ```makefile
    test-unit:
        go test -v -short ./...
    
    test-integration:
        go test -v -tags=integration ./tests/...
    ```

**Files to Create/Update**:
- `tests/integration_test.go` - Expand existing tests
- `tests/cache_test.go` - Cache-specific integration tests
- `tests/value_test.go` - GetValue integration tests
- `tests/fixtures/` - Test data directory
- `Makefile` - Test automation

---

#### 2.2 Unit Testing
**Status**: ⚠️ Partial (3 unit tests exist)  
**Priority**: MEDIUM  
**Effort**: Medium

**Current Tests**:
- `cache_test.go` - Basic cache tests
- `options_test.go` - Options parsing tests
- `value_test.go` - Value helper tests

**Tasks**:
- [ ] Add unit tests for:
  - [ ] Identifier builder edge cases
  - [ ] Error wrapping logic
  - [ ] Template loader YAML parsing
  - [ ] Cache eviction policies
  - [ ] Background sync logic (with mocked time)

- [ ] Increase test coverage to >80%
- [ ] Add benchmarks for critical paths:
  ```go
  func BenchmarkGetValueCached(b *testing.B) { ... }
  func BenchmarkIdentifierBuild(b *testing.B) { ... }
  ```

**Files to Create**:
- `identifier_test.go`
- `template_loader_test.go`
- `client_test.go` (with mocked gRPC)

---

### 2.3 Example Programs
**Status**: ✅ Good (1 comprehensive example exists)  
**Priority**: LOW  
**Effort**: Low

**Current State**:
- `examples/main.go` - Comprehensive example covering most features

**Enhancement Tasks**:
- [ ] Add more focused examples:
  - [ ] `examples/basic/` - Minimal working example
  - [ ] `examples/caching/` - Caching demonstration
  - [ ] `examples/templates/` - Template loading example
  - [ ] `examples/production/` - Production-ready setup with TLS

- [ ] Add example templates in `examples/templates/`
- [ ] Create `examples/README.md` with running instructions

---

## 🐛 Bug Fixes & Improvements

### 3. Known Issues

#### 3.1 Import Path Consistency
**Status**: ❌ Bug  
**Priority**: HIGH  
**Effort**: Low

**Issue**:
All files use hardcoded import path that may not match actual module path when installed.

**Tasks**:
- [ ] Audit all import statements in:
  - `client.go`
  - `identifier.go`
  - `template_loader.go`
  - `value.go`
  - `examples/main.go`
  - `tests/integration_test.go`

- [ ] Ensure consistency with `go.mod` module path
- [ ] Test imports work correctly when installed via `go get`

---

#### 3.2 Error Handling Improvements
**Status**: ⚠️ Needs Enhancement  
**Priority**: MEDIUM  
**Effort**: Medium

**Current State**:
- Basic error wrapping exists
- No custom error types for different failure modes

**Tasks**:
- [ ] Create custom error types:
  ```go
  type ConfigNotFoundError struct { ... }
  type TemplateNotFoundError struct { ... }
  type ServerUnavailableError struct { ... }
  ```

- [ ] Add error inspection helpers:
  ```go
  func IsNotFound(err error) bool { ... }
  func IsServerError(err error) bool { ... }
  ```

- [ ] Improve error messages with context
- [ ] Add error examples to README

**Files to Create/Update**:
- `errors.go` - Custom error types
- `client.go` - Use custom errors

---

#### 3.3 Context Cancellation Handling
**Status**: ⚠️ Needs Review  
**Priority**: MEDIUM  
**Effort**: Low

**Tasks**:
- [ ] Audit all gRPC calls for proper context handling
- [ ] Ensure background sync respects context cancellation
- [ ] Add timeout examples to README
- [ ] Test behavior with cancelled contexts

---

### 4. Missing Features

#### 4.1 Additional gRPC Methods
**Status**: ❌ Not Implemented  
**Priority**: MEDIUM  
**Effort**: Medium

**Methods to Implement** (currently commented in `client.go`):
- [ ] `GetConfigByVersion(ctx, identifier, version)` - Get specific version
- [ ] `GetConfigHistory(ctx, identifier, limit)` - Get version history
- [ ] `PublishVersion(ctx, identifier, version, user)` - Publish a version
- [ ] `DeleteConfig(ctx, identifier)` - Delete configuration
- [ ] `ListConfigTemplates(ctx, serviceName, isActive)` - List all templates

**Files to Update**:
- `client.go` - Add new methods
- `tests/integration_test.go` - Add tests for new methods
- `README.md` - Document new methods

---

#### 4.2 Retry Logic
**Status**: ❌ Not Implemented  
**Priority**: LOW  
**Effort**: Medium

**Tasks**:
- [ ] Add configurable retry logic for transient failures
- [ ] Use exponential backoff
- [ ] Make retry policy configurable:
  ```go
  WithRetryPolicy(maxRetries int, backoff time.Duration)
  ```

**Files to Create/Update**:
- `retry.go` - Retry logic
- `options.go` - Add retry options
- `client.go` - Integrate retry logic

---

#### 4.3 Metrics & Observability
**Status**: ❌ Not Implemented  
**Priority**: LOW  
**Effort**: High

**Tasks**:
- [ ] Add Prometheus metrics:
  - Cache hit/miss rate
  - gRPC call latency
  - Error rates
  - Background sync status

- [ ] Add structured logging with levels
- [ ] Add OpenTelemetry tracing support (optional)

**Files to Create**:
- `metrics.go` - Prometheus metrics
- `logging.go` - Structured logging

---

## 📦 Distribution & Documentation

### 5. Documentation

#### 5.1 README Improvements
**Status**: ✅ Good, needs minor updates  
**Priority**: MEDIUM  
**Effort**: Low

**Tasks**:
- [ ] Update installation section for `go get`
- [ ] Add troubleshooting section
- [ ] Add FAQ section
- [ ] Add migration guide from manual copy to `go get`
- [ ] Add examples of common patterns
- [ ] Document private repo access

---

#### 5.2 GoDoc Comments
**Status**: ✅ Good  
**Priority**: LOW  
**Effort**: Low

**Tasks**:
- [ ] Review all exported functions for godoc comments
- [ ] Add package-level documentation
- [ ] Add examples in godoc format:
  ```go
  // Example_getValue demonstrates getting a config value.
  func Example_getValue() { ... }
  ```

---

### 6. CI/CD & Automation

#### 6.1 GitHub Actions
**Status**: ❌ Not Implemented  
**Priority**: MEDIUM  
**Effort**: Medium

**Tasks**:
- [ ] Create `.github/workflows/go-sdk.yml`:
  - [ ] Run tests on PR
  - [ ] Run linters (golangci-lint)
  - [ ] Check code coverage
  - [ ] Verify buf generate is up-to-date
  - [ ] Test installation via `go get`

- [ ] Add status badges to README

**Files to Create**:
- `.github/workflows/go-sdk.yml`
- `.golangci.yml` - Linter configuration

---

#### 6.2 Versioning & Releases
**Status**: ❌ Not Implemented  
**Priority**: HIGH  
**Effort**: Low

**Tasks**:
- [ ] Define versioning strategy (semantic versioning)
- [ ] Create release process:
  1. Update CHANGELOG.md
  2. Tag release: `git tag sdks/go/v1.0.0`
  3. Push tag: `git push origin sdks/go/v1.0.0`

- [ ] Document release process in CONTRIBUTING.md
- [ ] Automate release notes generation

**Files to Create**:
- `CHANGELOG.md`
- `CONTRIBUTING.md`

---

## 🔧 Technical Debt

### 7. Code Quality

#### 7.1 Linting
**Status**: ⚠️ Unknown  
**Priority**: MEDIUM  
**Effort**: Low

**Tasks**:
- [ ] Run `golangci-lint run`
- [ ] Fix all linter warnings
- [ ] Add linter to CI/CD
- [ ] Configure linter rules in `.golangci.yml`

---

#### 7.2 Code Organization
**Status**: ✅ Good  
**Priority**: LOW  
**Effort**: N/A

**Current Structure**:
```
sdks/go/
├── client.go          # Main client
├── cache.go           # Cache implementation
├── identifier.go      # Identifier builder
├── options.go         # Client options
├── value.go           # Value helpers
├── template_loader.go # Template loading
├── gen/               # Generated proto code
├── examples/          # Example programs
└── tests/             # Integration tests
```

**Notes**:
- Structure is clean and logical
- No major refactoring needed

---

## 📊 Priority Matrix

| Task | Priority | Effort | Impact | Status |
|------|----------|--------|--------|--------|
| Make SDK installable via `go get` | HIGH | Medium | HIGH | ❌ |
| Decide on generated files strategy | HIGH | Low | HIGH | ⚠️ |
| Comprehensive integration tests | HIGH | High | HIGH | ⚠️ |
| Fix import path consistency | HIGH | Low | MEDIUM | ❌ |
| Implement missing gRPC methods | MEDIUM | Medium | MEDIUM | ❌ |
| Add GitHub Actions CI/CD | MEDIUM | Medium | MEDIUM | ❌ |
| Error handling improvements | MEDIUM | Medium | MEDIUM | ⚠️ |
| Expand unit test coverage | MEDIUM | Medium | MEDIUM | ⚠️ |
| README improvements | MEDIUM | Low | MEDIUM | ✅ |
| Add retry logic | LOW | Medium | LOW | ❌ |
| Add metrics & observability | LOW | High | LOW | ❌ |

---

## 🚀 Recommended Implementation Order

### Phase 1: Foundation (Week 1)
1. ✅ Decide on generated files strategy
2. ✅ Make SDK installable via `go get`
3. ✅ Fix import path consistency
4. ✅ Update README with installation instructions

### Phase 2: Testing (Week 2-3)
5. ✅ Expand integration test suite
6. ✅ Add unit tests for uncovered code
7. ✅ Add GitHub Actions CI/CD
8. ✅ Set up test fixtures and helpers

### Phase 3: Features (Week 4)
9. ✅ Implement missing gRPC methods
10. ✅ Improve error handling
11. ✅ Add more examples
12. ✅ Document new features

### Phase 4: Polish (Week 5)
13. ✅ Add retry logic
14. ✅ Run linters and fix issues
15. ✅ Add metrics (optional)
16. ✅ Create v1.0.0 release

---

## 📝 Notes

### Comparison with TypeScript SDK
The TypeScript SDK has:
- ✅ Published to GitHub Packages
- ✅ Comprehensive README
- ✅ Generated files committed
- ✅ Clear installation instructions

The Go SDK should match this level of polish.

### Testing Strategy
- **Unit tests**: Fast, no external dependencies, use mocks
- **Integration tests**: Require running server, use testcontainers
- **Example programs**: Manual verification, documentation

### Module Path Considerations
Current path: `github.com/digitalsolutionsai/scope-config-service/sdks/go`

This is fine for a monorepo. Alternative: Create separate repo for SDK.

---

## ✅ Definition of Done

A task is considered complete when:
- [ ] Code is written and tested
- [ ] Tests pass in CI/CD
- [ ] Documentation is updated
- [ ] Code review is completed
- [ ] Changes are merged to main branch
- [ ] Release notes are updated (if applicable)

---

**Last Updated**: 2026-01-04  
**Maintainer**: @digitalsolutionsai  
**Version**: 1.0.0
