package scopeconfig

import (
	"testing"

	configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
)

func TestNewIdentifier(t *testing.T) {
	builder := NewIdentifier("test-service")
	if builder == nil {
		t.Fatal("Expected builder to be created")
	}

	identifier := builder.Build()
	if identifier.ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", identifier.ServiceName)
	}
	if identifier.Scope != configv1.Scope_SCOPE_UNSPECIFIED {
		t.Errorf("Expected default scope to be SCOPE_UNSPECIFIED, got %v", identifier.Scope)
	}
}

func TestIdentifierBuilderWithScope(t *testing.T) {
	tests := []struct {
		name     string
		scope    configv1.Scope
		expected configv1.Scope
	}{
		{"SYSTEM scope", configv1.Scope_SYSTEM, configv1.Scope_SYSTEM},
		{"PROJECT scope", configv1.Scope_PROJECT, configv1.Scope_PROJECT},
		{"STORE scope", configv1.Scope_STORE, configv1.Scope_STORE},
		{"USER scope", configv1.Scope_USER, configv1.Scope_USER},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identifier := NewIdentifier("test-service").
				WithScope(tt.scope).
				Build()

			if identifier.Scope != tt.expected {
				t.Errorf("Expected scope %v, got %v", tt.expected, identifier.Scope)
			}
		})
	}
}

func TestIdentifierBuilderWithGroupID(t *testing.T) {
	identifier := NewIdentifier("test-service").
		WithGroupID("test-group").
		Build()

	if identifier.GroupId != "test-group" {
		t.Errorf("Expected group ID 'test-group', got '%s'", identifier.GroupId)
	}
}

func TestIdentifierBuilderWithProjectID(t *testing.T) {
	identifier := NewIdentifier("test-service").
		WithProjectID("proj-123").
		Build()

	if identifier.ProjectId != "proj-123" {
		t.Errorf("Expected project ID 'proj-123', got '%s'", identifier.ProjectId)
	}
}

func TestIdentifierBuilderWithStoreID(t *testing.T) {
	identifier := NewIdentifier("test-service").
		WithStoreID("store-456").
		Build()

	if identifier.StoreId != "store-456" {
		t.Errorf("Expected store ID 'store-456', got '%s'", identifier.StoreId)
	}
}

func TestIdentifierBuilderWithUserID(t *testing.T) {
	identifier := NewIdentifier("test-service").
		WithUserID("user-789").
		Build()

	if identifier.UserId != "user-789" {
		t.Errorf("Expected user ID 'user-789', got '%s'", identifier.UserId)
	}
}

func TestIdentifierBuilderChaining(t *testing.T) {
	identifier := NewIdentifier("payment-service").
		WithScope(configv1.Scope_STORE).
		WithGroupID("checkout").
		WithProjectID("proj-123").
		WithStoreID("store-456").
		Build()

	if identifier.ServiceName != "payment-service" {
		t.Errorf("Expected service name 'payment-service', got '%s'", identifier.ServiceName)
	}
	if identifier.Scope != configv1.Scope_STORE {
		t.Errorf("Expected scope STORE, got %v", identifier.Scope)
	}
	if identifier.GroupId != "checkout" {
		t.Errorf("Expected group ID 'checkout', got '%s'", identifier.GroupId)
	}
	if identifier.ProjectId != "proj-123" {
		t.Errorf("Expected project ID 'proj-123', got '%s'", identifier.ProjectId)
	}
	if identifier.StoreId != "store-456" {
		t.Errorf("Expected store ID 'store-456', got '%s'", identifier.StoreId)
	}
}

func TestIdentifierBuilderEmptyServiceName(t *testing.T) {
	identifier := NewIdentifier("").Build()
	if identifier.ServiceName != "" {
		t.Errorf("Expected empty service name, got '%s'", identifier.ServiceName)
	}
}

func TestIdentifierBuilderMultipleBuilds(t *testing.T) {
	builder := NewIdentifier("test-service").
		WithScope(configv1.Scope_PROJECT).
		WithGroupID("api")

	// Build multiple times should return the same identifier
	id1 := builder.Build()
	id2 := builder.Build()

	if id1.ServiceName != id2.ServiceName {
		t.Error("Multiple builds should return consistent identifiers")
	}
	if id1.Scope != id2.Scope {
		t.Error("Multiple builds should return consistent scopes")
	}
}
