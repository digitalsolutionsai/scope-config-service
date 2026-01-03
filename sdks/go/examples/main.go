/*
Example usage of the ScopeConfig Go SDK.

This example demonstrates:
- Creating a client using environment variables
- Building config identifiers
- Getting config values with caching
- Using inheritance and default values
- Applying configuration templates

Prerequisites:
1. Generate proto files: buf generate
2. Set environment variables (optional):
  - GRPC_SCOPE_CONFIG_HOST (default: localhost)
  - GRPC_SCOPE_CONFIG_PORT (default: 50051)
  - GRPC_SCOPE_CONFIG_USE_TLS (default: false)

Run:

	go run main.go
*/
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	scopeconfig "github.com/digitalsolutionsai/scope-config-service/sdks/go"
	configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
)

func main() {
	// Example 1: Create client using environment variables
	fmt.Println("=== Example 1: Using Environment Variables ===")
	clientEnv, err := scopeconfig.NewClient(scopeconfig.FromEnvironment()...)
	if err != nil {
		log.Printf("Failed to create client from env: %v (this is expected if server is not running)", err)
	} else {
		defer clientEnv.Close()
		fmt.Println("Client created successfully using environment variables")
	}

	// Example 2: Create client with explicit configuration
	fmt.Println("\n=== Example 2: Explicit Configuration ===")
	client, err := scopeconfig.NewClient(
		scopeconfig.WithAddress("localhost:50051"),
		scopeconfig.WithInsecure(),
		scopeconfig.WithCache(time.Minute),
		scopeconfig.WithBackgroundSync(30*time.Second),
	)
	if err != nil {
		log.Printf("Failed to create client: %v (this is expected if server is not running)", err)
		fmt.Println("\nNote: To run this example with a live server, start the ScopeConfig service first.")
		demonstrateIdentifierBuilding()
		return
	}
	defer client.Close()
	fmt.Println("Client created successfully with explicit configuration")

	ctx := context.Background()

	// Example 3: Build config identifier
	fmt.Println("\n=== Example 3: Building Config Identifiers ===")
	demonstrateIdentifierBuilding()

	// Example 4: Get configuration with caching
	fmt.Println("\n=== Example 4: Get Configuration with Caching ===")
	identifier := scopeconfig.NewIdentifier("payment-service").
		WithScope(configv1.Scope_PROJECT).
		WithGroupID("database").
		WithProjectID("proj-123").
		Build()

	config, err := client.GetConfigCached(ctx, identifier)
	if err != nil {
		log.Printf("Failed to get config: %v", err)
	} else {
		fmt.Printf("Configuration for %s:\n", config.VersionInfo.Identifier.ServiceName)
		for _, field := range config.Fields {
			fmt.Printf("  %s = %s\n", field.Path, field.Value)
		}
	}

	// Example 5: Get specific value with inheritance
	fmt.Println("\n=== Example 5: Get Value with Inheritance ===")
	value, err := client.GetValue(ctx, identifier, "database.host", &scopeconfig.GetValueOptions{
		UseDefault: true, // Use template default if not set
		Inherit:    true, // Traverse parent scopes (STORE → PROJECT → SYSTEM)
	})
	if err != nil {
		log.Printf("Failed to get value: %v", err)
	} else if value != nil {
		fmt.Printf("Database host: %s\n", *value)
	} else {
		fmt.Println("Database host not found")
	}

	// Example 6: Get value as string (convenience method)
	fmt.Println("\n=== Example 6: Get Value as String ===")
	host, err := client.GetValueString(ctx, identifier, "database.host", &scopeconfig.GetValueOptions{
		UseDefault: true,
	})
	if err != nil {
		log.Printf("Failed to get value string: %v", err)
	} else {
		fmt.Printf("Database host (string): '%s'\n", host)
	}

	// Example 7: Apply configuration template
	fmt.Println("\n=== Example 7: Apply Configuration Template ===")
	template := &configv1.ConfigTemplate{
		Identifier: scopeconfig.NewIdentifier("payment-service").
			WithGroupID("logging").
			Build(),
		ServiceLabel:     "Payment Service",
		GroupLabel:       "Logging Configuration",
		GroupDescription: "Controls logging behavior for the payment service",
		Fields: []*configv1.ConfigFieldTemplate{
			{
				Path:         "log.level",
				Label:        "Log Level",
				Description:  "Application logging level",
				Type:         configv1.FieldType_STRING,
				DefaultValue: "INFO",
				DisplayOn:    []configv1.Scope{configv1.Scope_SYSTEM, configv1.Scope_PROJECT},
				Options: []*configv1.ValueOption{
					{Value: "DEBUG", Label: "Debug"},
					{Value: "INFO", Label: "Info"},
					{Value: "WARN", Label: "Warning"},
					{Value: "ERROR", Label: "Error"},
				},
				SortOrder: 100000,
			},
			{
				Path:         "log.format",
				Label:        "Log Format",
				Description:  "Output format for log messages",
				Type:         configv1.FieldType_STRING,
				DefaultValue: "json",
				DisplayOn:    []configv1.Scope{configv1.Scope_SYSTEM},
				Options: []*configv1.ValueOption{
					{Value: "json", Label: "JSON"},
					{Value: "text", Label: "Plain Text"},
				},
				SortOrder: 100001,
			},
		},
		SortOrder: 100000,
	}

	result, err := client.ApplyConfigTemplate(ctx, template, "admin@example.com")
	if err != nil {
		log.Printf("Failed to apply template: %v", err)
	} else {
		fmt.Printf("Applied template: %s - %s\n", result.ServiceLabel, result.GroupLabel)
	}

	// Example 8: Load templates from directory
	fmt.Println("\n=== Example 8: Load Templates from Directory ===")
	err = client.LoadTemplatesFromDir(ctx, "./templates", "system")
	if err != nil {
		log.Printf("Failed to load templates: %v (templates directory may not exist)", err)
	} else {
		fmt.Println("Templates loaded successfully")
	}

	// Example 9: Cache management
	fmt.Println("\n=== Example 9: Cache Management ===")
	fmt.Printf("Cache enabled: %v\n", client.IsCacheEnabled())

	// Invalidate specific config cache
	client.InvalidateCache(identifier)
	fmt.Println("Cache invalidated for specific identifier")

	// Clear all cache
	client.ClearCache()
	fmt.Println("All cache cleared")

	fmt.Println("\n=== Example Complete ===")
}

func demonstrateIdentifierBuilding() {
	// SYSTEM scope (global config)
	systemID := scopeconfig.NewIdentifier("my-service").
		WithScope(configv1.Scope_SYSTEM).
		WithGroupID("database").
		Build()
	fmt.Printf("System identifier: service=%s, group=%s, scope=%v\n",
		systemID.ServiceName, systemID.GroupId, systemID.Scope)

	// PROJECT scope
	projectID := scopeconfig.NewIdentifier("my-service").
		WithScope(configv1.Scope_PROJECT).
		WithGroupID("database").
		WithProjectID("proj-123").
		Build()
	fmt.Printf("Project identifier: service=%s, group=%s, project=%s\n",
		projectID.ServiceName, projectID.GroupId, projectID.ProjectId)

	// STORE scope
	storeID := scopeconfig.NewIdentifier("my-service").
		WithScope(configv1.Scope_STORE).
		WithGroupID("database").
		WithProjectID("proj-123").
		WithStoreID("store-456").
		Build()
	fmt.Printf("Store identifier: service=%s, group=%s, project=%s, store=%s\n",
		storeID.ServiceName, storeID.GroupId, storeID.ProjectId, storeID.StoreId)

	// USER scope
	userID := scopeconfig.NewIdentifier("my-service").
		WithScope(configv1.Scope_USER).
		WithGroupID("preferences").
		WithUserID("user-789").
		Build()
	fmt.Printf("User identifier: service=%s, group=%s, user=%s\n",
		userID.ServiceName, userID.GroupId, userID.UserId)
}
