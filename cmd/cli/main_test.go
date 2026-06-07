package main

import (
	"testing"

	"github.com/digitalsolutionsai/scope-config-service/pkg/version"
	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
)

func TestCliVersionMatchesPackageVersion(t *testing.T) {
	if cliVersion != version.Version {
		t.Errorf("cliVersion = %q, want %q", cliVersion, version.Version)
	}
}

func TestCreateIdentifier_Valid(t *testing.T) {
	// Set global flags
	serviceName = "my-service"
	groupID = "my-group"
	scope = "SYSTEM"
	projectID = ""
	storeID = ""
	userID = ""

	id, err := createIdentifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.ServiceName != "my-service" {
		t.Errorf("ServiceName = %q, want %q", id.ServiceName, "my-service")
	}
	if id.GroupId != "my-group" {
		t.Errorf("GroupId = %q, want %q", id.GroupId, "my-group")
	}
	if id.Scope != configv1.Scope_SYSTEM {
		t.Errorf("Scope = %v, want SYSTEM", id.Scope)
	}
}

func TestCreateIdentifier_ProjectScope(t *testing.T) {
	serviceName = "svc"
	groupID = "grp"
	scope = "PROJECT"
	projectID = "proj-123"

	id, err := createIdentifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.ProjectId != "proj-123" {
		t.Errorf("ProjectId = %q, want %q", id.ProjectId, "proj-123")
	}
}

func TestCreateIdentifier_MissingServiceName(t *testing.T) {
	serviceName = ""
	groupID = "grp"
	scope = "SYSTEM"

	_, err := createIdentifier()
	if err == nil {
		t.Error("expected error for missing service name")
	}
}

func TestCreateIdentifier_MissingGroupID(t *testing.T) {
	serviceName = "svc"
	groupID = ""
	scope = "SYSTEM"

	_, err := createIdentifier()
	if err == nil {
		t.Error("expected error for missing group ID")
	}
}

func TestCreateIdentifier_MissingScope(t *testing.T) {
	serviceName = "svc"
	groupID = "grp"
	scope = ""

	_, err := createIdentifier()
	if err == nil {
		t.Error("expected error for missing scope")
	}
}

func TestCreateIdentifier_InvalidScope(t *testing.T) {
	serviceName = "svc"
	groupID = "grp"
	scope = "INVALID"

	_, err := createIdentifier()
	if err == nil {
		t.Error("expected error for invalid scope")
	}
}

func TestCreateIdentifier_ProjectScopeMissingProjectID(t *testing.T) {
	serviceName = "svc"
	groupID = "grp"
	scope = "PROJECT"
	projectID = ""

	_, err := createIdentifier()
	if err == nil {
		t.Error("expected error for PROJECT scope without project-id")
	}
}

func TestCreateIdentifier_StoreScopeMissingStoreID(t *testing.T) {
	serviceName = "svc"
	groupID = "grp"
	scope = "STORE"
	storeID = ""

	_, err := createIdentifier()
	if err == nil {
		t.Error("expected error for STORE scope without store-id")
	}
}

func TestCreateIdentifier_UserScopeMissingUserID(t *testing.T) {
	serviceName = "svc"
	groupID = "grp"
	scope = "USER"
	userID = ""

	_, err := createIdentifier()
	if err == nil {
		t.Error("expected error for USER scope without user-id")
	}
}
