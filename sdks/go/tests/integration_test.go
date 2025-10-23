package scopeconfig_test

import (
	"context"
	"testing"

	scopeconfig "github.com/digitalsolutionsai/scope-config-service/sdks/go"
	configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
)

// TestApplyAndGetTemplate demonstrates applying a config template and retrieving it.
// This test requires the ScopeConfig service to be running on localhost:50051.
func TestApplyAndGetTemplate(t *testing.T) {

	// Create client
	client, err := scopeconfig.NewClient(
		scopeconfig.WithAddress("localhost:50051"),
		scopeconfig.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	serviceName := "test-service-go"
	groupID := "test-group-go"
	user := "go-test-runner"

	// 1. Apply a Config Template
	template := &configv1.ConfigTemplate{
		Identifier: scopeconfig.NewIdentifier(serviceName).
			WithGroupID(groupID).
			Build(),
		ServiceLabel:     "Test Service from Go",
		GroupLabel:       "Test Group from Go",
		GroupDescription: "A test group created during automated Go tests.",
		Fields: []*configv1.ConfigFieldTemplate{
			{
				Path:         "log.level",
				Label:        "Logging Level",
				Description:  "Controls the verbosity of application logging.",
				Type:         configv1.FieldType_STRING,
				DefaultValue: "INFO",
				Options: []*configv1.ValueOption{
					{Value: "DEBUG", Label: "Debug"},
					{Value: "INFO", Label: "Info"},
					{Value: "WARN", Label: "Warning"},
					{Value: "ERROR", Label: "Error"},
				},
			},
		},
	}

	appliedTemplate, err := client.ApplyConfigTemplate(ctx, template, user)
	if err != nil {
		t.Fatalf("ApplyConfigTemplate failed: %v", err)
	}
	t.Logf("Successfully applied template for %s/%s", serviceName, groupID)

	// 2. Get the Config Template to verify
	identifier := scopeconfig.NewIdentifier(serviceName).
		WithGroupID(groupID).
		Build()

	retrievedTemplate, err := client.GetConfigTemplate(ctx, identifier)
	if err != nil {
		t.Fatalf("GetConfigTemplate failed: %v", err)
	}
	t.Logf("Successfully retrieved template for %s/%s", serviceName, groupID)

	// 3. Assert that the retrieved data matches the applied data
	if retrievedTemplate.Identifier.ServiceName != serviceName {
		t.Errorf("Expected service name %s, got %s", serviceName, retrievedTemplate.Identifier.ServiceName)
	}

	if retrievedTemplate.Identifier.GroupId != groupID {
		t.Errorf("Expected group ID %s, got %s", groupID, retrievedTemplate.Identifier.GroupId)
	}

	if retrievedTemplate.GroupLabel != "Test Group from Go" {
		t.Errorf("Expected group label 'Test Group from Go', got %s", retrievedTemplate.GroupLabel)
	}

	if len(retrievedTemplate.Fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(retrievedTemplate.Fields))
	}

	if len(retrievedTemplate.Fields) > 0 {
		field := retrievedTemplate.Fields[0]
		if field.Path != "log.level" {
			t.Errorf("Expected field path 'log.level', got %s", field.Path)
		}

		if len(field.Options) != 4 {
			t.Errorf("Expected 4 options, got %d", len(field.Options))
		}

		if len(field.Options) > 0 && field.Options[0].Value != "DEBUG" {
			t.Errorf("Expected first option value 'DEBUG', got %s", field.Options[0].Value)
		}
	}

	// Verify the returned template from apply matches as well
	if appliedTemplate.GroupLabel != retrievedTemplate.GroupLabel {
		t.Errorf("Applied template doesn't match retrieved template")
	}
}

// ExampleClient demonstrates basic usage of the ScopeConfig client.
func ExampleClient() {
	// Create a client
	client, err := scopeconfig.NewClient(
		scopeconfig.WithAddress("localhost:50051"),
		scopeconfig.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Build a config identifier
	identifier := scopeconfig.NewIdentifier("my-service").
		WithScope(configv1.Scope_SYSTEM).
		WithGroupID("database").
		Build()

	// Get configuration
	config, err := client.GetConfig(ctx, identifier)
	if err != nil {
		panic(err)
	}

	// Use the configuration
	for _, field := range config.Fields {
		println(field.Path, "=", field.Value)
	}
}

// ExampleNewIdentifier demonstrates the identifier builder.
func ExampleNewIdentifier() {
	// Simple identifier
	id1 := scopeconfig.NewIdentifier("my-service").
		WithGroupID("api").
		Build()

	// Complex identifier with multiple scopes
	id2 := scopeconfig.NewIdentifier("payment-service").
		WithScope(configv1.Scope_STORE).
		WithGroupID("checkout").
		WithProjectID("proj-123").
		WithStoreID("store-456").
		Build()

	println(id1.ServiceName)
	println(id2.ServiceName, id2.ProjectId, id2.StoreId)
}
