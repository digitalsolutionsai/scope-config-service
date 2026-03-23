package service

import (
	"testing"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
)

func TestGetIdentifier_System(t *testing.T) {
	id := &configv1.ConfigIdentifier{
		ServiceName: "svc",
		Scope:       configv1.Scope_SYSTEM,
		GroupId:     "grp",
	}
	scope, scopeID, err := getIdentifier(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope != configv1.Scope_SYSTEM {
		t.Errorf("scope = %v, want SYSTEM", scope)
	}
	if scopeID != "system" {
		t.Errorf("scopeID = %q, want %q", scopeID, "system")
	}
}

func TestGetIdentifier_Project(t *testing.T) {
	id := &configv1.ConfigIdentifier{
		ServiceName: "svc",
		Scope:       configv1.Scope_PROJECT,
		GroupId:     "grp",
		ProjectId:   "proj-123",
	}
	scope, scopeID, err := getIdentifier(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope != configv1.Scope_PROJECT {
		t.Errorf("scope = %v, want PROJECT", scope)
	}
	if scopeID != "proj-123" {
		t.Errorf("scopeID = %q, want %q", scopeID, "proj-123")
	}
}

func TestGetIdentifier_Store(t *testing.T) {
	id := &configv1.ConfigIdentifier{
		ServiceName: "svc",
		Scope:       configv1.Scope_STORE,
		GroupId:     "grp",
		StoreId:     "store-456",
	}
	_, scopeID, err := getIdentifier(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scopeID != "store-456" {
		t.Errorf("scopeID = %q, want %q", scopeID, "store-456")
	}
}

func TestGetIdentifier_User(t *testing.T) {
	id := &configv1.ConfigIdentifier{
		ServiceName: "svc",
		Scope:       configv1.Scope_USER,
		GroupId:     "grp",
		UserId:      "user-789",
	}
	_, scopeID, err := getIdentifier(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scopeID != "user-789" {
		t.Errorf("scopeID = %q, want %q", scopeID, "user-789")
	}
}

func TestGetIdentifier_NilIdentifier(t *testing.T) {
	_, _, err := getIdentifier(nil)
	if err == nil {
		t.Error("expected error for nil identifier")
	}
}

func TestGetIdentifier_EmptyScopeID(t *testing.T) {
	id := &configv1.ConfigIdentifier{
		ServiceName: "svc",
		Scope:       configv1.Scope_PROJECT,
		GroupId:     "grp",
		ProjectId:   "", // empty
	}
	_, _, err := getIdentifier(id)
	if err == nil {
		t.Error("expected error for empty project ID")
	}
}

func TestGetIdentifier_UnsupportedScope(t *testing.T) {
	id := &configv1.ConfigIdentifier{
		ServiceName: "svc",
		Scope:       configv1.Scope_SCOPE_UNSPECIFIED,
		GroupId:     "grp",
	}
	_, _, err := getIdentifier(id)
	if err == nil {
		t.Error("expected error for unspecified scope")
	}
}
