//go:build integration

/*
Integration tests for the ScopeConfig Go SDK.

These tests use testcontainers-go to spin up PostgreSQL and the ScopeConfig service.

Prerequisites:
1. Docker must be running
2. The scope-config-service Docker image must be available. You can:
   a. Pre-build it: docker build -t scope-config-service:test .
   b. Let testcontainers build it (requires network access)

Run tests:

	go test -v -tags=integration -timeout=10m ./tests/...

Or use the Makefile:

	make test-integration

Environment variables:
- SCOPE_CONFIG_IMAGE: Docker image to use (default: builds from Dockerfile)
- SKIP_INTEGRATION_TESTS: Set to "true" to skip integration tests
*/
package scopeconfig_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	scopeconfig "github.com/digitalsolutionsai/scope-config-service/sdks/go"
	configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
)

var (
	testClient  *scopeconfig.Client
	testAddress string
	skipTests   bool
)

// TestMain sets up the test containers and runs all tests.
func TestMain(m *testing.M) {
	// Check if integration tests should be skipped
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		log.Println("Skipping integration tests (SKIP_INTEGRATION_TESTS=true)")
		os.Exit(0)
	}

	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("config_db"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		log.Printf("Failed to start postgres container: %v", err)
		log.Println("Skipping integration tests - Docker may not be available")
		os.Exit(0)
	}
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			log.Printf("Failed to terminate postgres container: %v", err)
		}
	}()

	// Get PostgreSQL connection details
	pgHost, err := postgresContainer.Host(ctx)
	if err != nil {
		log.Fatalf("Failed to get postgres host: %v", err)
	}
	pgPort, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		log.Fatalf("Failed to get postgres port: %v", err)
	}

	databaseURL := fmt.Sprintf("postgresql://testuser:testpass@%s:%s/config_db?sslmode=disable", pgHost, pgPort.Port())

	// Determine how to create the config service container
	var configServiceContainer testcontainers.Container

	// Check if a pre-built image should be used
	customImage := os.Getenv("SCOPE_CONFIG_IMAGE")
	if customImage != "" {
		log.Printf("Using pre-built image: %s", customImage)
		configServiceContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        customImage,
				ExposedPorts: []string{"50051/tcp", "8080/tcp"},
				Env: map[string]string{
					"DATABASE_URL": databaseURL,
					"GRPC_PORT":    "50051",
					"HTTP_PORT":    "8080",
				},
				WaitingFor: wait.ForAll(
					wait.ForListeningPort("50051/tcp"),
					wait.ForLog("Starting gRPC server").WithStartupTimeout(120*time.Second),
				),
			},
			Started: true,
		})
	} else {
		// Build from Dockerfile
		projectRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
		if err != nil {
			log.Fatalf("Failed to get project root: %v", err)
		}

		log.Printf("Building image from Dockerfile at %s", projectRoot)
		configServiceContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				FromDockerfile: testcontainers.FromDockerfile{
					Context:    projectRoot,
					Dockerfile: "Dockerfile",
				},
				ExposedPorts: []string{"50051/tcp", "8080/tcp"},
				Env: map[string]string{
					"DATABASE_URL": databaseURL,
					"GRPC_PORT":    "50051",
					"HTTP_PORT":    "8080",
				},
				WaitingFor: wait.ForAll(
					wait.ForListeningPort("50051/tcp"),
					wait.ForLog("Starting gRPC server").WithStartupTimeout(180*time.Second),
				),
			},
			Started: true,
		})
	}

	if err != nil {
		log.Printf("Failed to start config service container: %v", err)
		log.Println("Skipping integration tests - Could not build/start config service")
		log.Println("Tip: Pre-build the image with: docker build -t scope-config-service:test .")
		log.Println("     Then run with: SCOPE_CONFIG_IMAGE=scope-config-service:test go test -tags=integration ./tests/...")
		skipTests = true
		os.Exit(0)
	}
	defer func() {
		if err := configServiceContainer.Terminate(ctx); err != nil {
			log.Printf("Failed to terminate config service container: %v", err)
		}
	}()

	// Get the config service host and port
	host, err := configServiceContainer.Host(ctx)
	if err != nil {
		log.Fatalf("Failed to get config service host: %v", err)
	}
	port, err := configServiceContainer.MappedPort(ctx, "50051")
	if err != nil {
		log.Fatalf("Failed to get config service port: %v", err)
	}

	testAddress = fmt.Sprintf("%s:%s", host, port.Port())
	log.Printf("Config service running at %s", testAddress)

	// Create a test client
	testClient, err = scopeconfig.NewClient(
		scopeconfig.WithAddress(testAddress),
		scopeconfig.WithInsecure(),
		scopeconfig.WithCache(time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to create test client: %v", err)
	}
	defer testClient.Close()

	// Wait a bit for the service to be fully ready
	time.Sleep(2 * time.Second)

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// TestApplyAndGetTemplate tests applying a config template and retrieving it.
func TestApplyAndGetTemplate(t *testing.T) {
	ctx := context.Background()
	serviceName := "integration-test-service"
	groupID := "test-group"
	user := "integration-test"

	// Apply a Config Template
	template := &configv1.ConfigTemplate{
		Identifier: scopeconfig.NewIdentifier(serviceName).
			WithGroupID(groupID).
			Build(),
		ServiceLabel:     "Integration Test Service",
		GroupLabel:       "Test Group",
		GroupDescription: "A test group for integration tests",
		Fields: []*configv1.ConfigFieldTemplate{
			{
				Path:         "log.level",
				Label:        "Log Level",
				Description:  "Application logging level",
				Type:         configv1.FieldType_STRING,
				DefaultValue: "INFO",
				Options: []*configv1.ValueOption{
					{Value: "DEBUG", Label: "Debug"},
					{Value: "INFO", Label: "Info"},
					{Value: "WARN", Label: "Warning"},
					{Value: "ERROR", Label: "Error"},
				},
			},
			{
				Path:         "log.format",
				Label:        "Log Format",
				Description:  "Log output format",
				Type:         configv1.FieldType_STRING,
				DefaultValue: "json",
			},
		},
	}

	appliedTemplate, err := testClient.ApplyConfigTemplate(ctx, template, user)
	require.NoError(t, err, "ApplyConfigTemplate should succeed")
	assert.Equal(t, "Test Group", appliedTemplate.GroupLabel)

	// Retrieve the template
	identifier := scopeconfig.NewIdentifier(serviceName).
		WithGroupID(groupID).
		Build()

	retrievedTemplate, err := testClient.GetConfigTemplate(ctx, identifier)
	require.NoError(t, err, "GetConfigTemplate should succeed")

	assert.Equal(t, serviceName, retrievedTemplate.Identifier.ServiceName)
	assert.Equal(t, groupID, retrievedTemplate.Identifier.GroupId)
	assert.Equal(t, "Test Group", retrievedTemplate.GroupLabel)
	assert.Len(t, retrievedTemplate.Fields, 2)
}

// TestUpdateAndGetConfig tests updating and retrieving configuration values.
func TestUpdateAndGetConfig(t *testing.T) {
	ctx := context.Background()
	serviceName := "config-test-service"
	groupID := "settings"
	user := "config-test"

	// First, apply a template
	template := &configv1.ConfigTemplate{
		Identifier: scopeconfig.NewIdentifier(serviceName).
			WithGroupID(groupID).
			Build(),
		ServiceLabel:     "Config Test Service",
		GroupLabel:       "Settings",
		GroupDescription: "Test settings group",
		Fields: []*configv1.ConfigFieldTemplate{
			{
				Path:         "database.host",
				Label:        "Database Host",
				Type:         configv1.FieldType_STRING,
				DefaultValue: "localhost",
			},
			{
				Path:         "database.port",
				Label:        "Database Port",
				Type:         configv1.FieldType_INT,
				DefaultValue: "5432",
			},
		},
	}

	_, err := testClient.ApplyConfigTemplate(ctx, template, user)
	require.NoError(t, err)

	// Update config at SYSTEM scope
	identifier := scopeconfig.NewIdentifier(serviceName).
		WithScope(configv1.Scope_SYSTEM).
		WithGroupID(groupID).
		Build()

	fields := []*configv1.ConfigField{
		{Path: "database.host", Value: "db.example.com", Type: configv1.FieldType_STRING},
		{Path: "database.port", Value: "5433", Type: configv1.FieldType_INT},
	}

	updatedConfig, err := testClient.UpdateConfig(ctx, identifier, fields, user)
	require.NoError(t, err, "UpdateConfig should succeed")
	assert.NotNil(t, updatedConfig)

	// Get the config
	config, err := testClient.GetConfig(ctx, identifier)
	require.NoError(t, err, "GetConfig should succeed")
	assert.NotNil(t, config)

	// Verify the values
	var hostValue, portValue string
	for _, field := range config.Fields {
		if field.Path == "database.host" {
			hostValue = field.Value
		}
		if field.Path == "database.port" {
			portValue = field.Value
		}
	}
	assert.Equal(t, "db.example.com", hostValue)
	assert.Equal(t, "5433", portValue)
}

// TestGetConfigCached tests the caching functionality.
func TestGetConfigCached(t *testing.T) {
	ctx := context.Background()
	serviceName := "cache-test-service"
	groupID := "cache-group"
	user := "cache-test"

	// Apply a template
	template := &configv1.ConfigTemplate{
		Identifier: scopeconfig.NewIdentifier(serviceName).
			WithGroupID(groupID).
			Build(),
		ServiceLabel: "Cache Test Service",
		GroupLabel:   "Cache Group",
		Fields: []*configv1.ConfigFieldTemplate{
			{
				Path:         "cache.enabled",
				Label:        "Cache Enabled",
				Type:         configv1.FieldType_BOOLEAN,
				DefaultValue: "true",
			},
		},
	}

	_, err := testClient.ApplyConfigTemplate(ctx, template, user)
	require.NoError(t, err)

	// Update config
	identifier := scopeconfig.NewIdentifier(serviceName).
		WithScope(configv1.Scope_SYSTEM).
		WithGroupID(groupID).
		Build()

	fields := []*configv1.ConfigField{
		{Path: "cache.enabled", Value: "true", Type: configv1.FieldType_BOOLEAN},
	}

	_, err = testClient.UpdateConfig(ctx, identifier, fields, user)
	require.NoError(t, err)

	// First call should hit the server
	config1, err := testClient.GetConfigCached(ctx, identifier)
	require.NoError(t, err)
	assert.NotNil(t, config1)

	// Second call should return cached value
	config2, err := testClient.GetConfigCached(ctx, identifier)
	require.NoError(t, err)
	assert.NotNil(t, config2)

	// Verify cache is enabled
	assert.True(t, testClient.IsCacheEnabled())
}

// TestGetValue tests retrieving specific configuration values.
func TestGetValue(t *testing.T) {
	ctx := context.Background()
	serviceName := "value-test-service"
	groupID := "api"
	user := "value-test"

	// Apply a template with default values
	template := &configv1.ConfigTemplate{
		Identifier: scopeconfig.NewIdentifier(serviceName).
			WithGroupID(groupID).
			Build(),
		ServiceLabel: "Value Test Service",
		GroupLabel:   "API Config",
		Fields: []*configv1.ConfigFieldTemplate{
			{
				Path:         "api.timeout",
				Label:        "API Timeout",
				Type:         configv1.FieldType_INT,
				DefaultValue: "30",
			},
			{
				Path:         "api.retries",
				Label:        "API Retries",
				Type:         configv1.FieldType_INT,
				DefaultValue: "3",
			},
		},
	}

	_, err := testClient.ApplyConfigTemplate(ctx, template, user)
	require.NoError(t, err)

	// Set a value at SYSTEM scope
	identifier := scopeconfig.NewIdentifier(serviceName).
		WithScope(configv1.Scope_SYSTEM).
		WithGroupID(groupID).
		Build()

	fields := []*configv1.ConfigField{
		{Path: "api.timeout", Value: "60", Type: configv1.FieldType_INT},
	}

	_, err = testClient.UpdateConfig(ctx, identifier, fields, user)
	require.NoError(t, err)

	// Get the value
	value, err := testClient.GetValue(ctx, identifier, "api.timeout", &scopeconfig.GetValueOptions{
		UseDefault: true,
	})
	require.NoError(t, err)
	require.NotNil(t, value)
	assert.Equal(t, "60", *value)

	// Get a value that uses default
	retriesValue, err := testClient.GetValue(ctx, identifier, "api.retries", &scopeconfig.GetValueOptions{
		UseDefault: true,
	})
	require.NoError(t, err)
	require.NotNil(t, retriesValue)
	assert.Equal(t, "3", *retriesValue)
}

// TestGetValueWithInheritance tests value inheritance across scopes.
func TestGetValueWithInheritance(t *testing.T) {
	ctx := context.Background()
	serviceName := "inheritance-test-service"
	groupID := "inherit-group"
	user := "inherit-test"

	// Apply a template
	template := &configv1.ConfigTemplate{
		Identifier: scopeconfig.NewIdentifier(serviceName).
			WithGroupID(groupID).
			Build(),
		ServiceLabel: "Inheritance Test Service",
		GroupLabel:   "Inheritance Group",
		Fields: []*configv1.ConfigFieldTemplate{
			{
				Path:         "feature.enabled",
				Label:        "Feature Enabled",
				Type:         configv1.FieldType_BOOLEAN,
				DefaultValue: "false",
				DisplayOn:    []configv1.Scope{configv1.Scope_SYSTEM, configv1.Scope_PROJECT},
			},
		},
	}

	_, err := testClient.ApplyConfigTemplate(ctx, template, user)
	require.NoError(t, err)

	// Set value at SYSTEM scope
	systemIdentifier := scopeconfig.NewIdentifier(serviceName).
		WithScope(configv1.Scope_SYSTEM).
		WithGroupID(groupID).
		Build()

	_, err = testClient.UpdateConfig(ctx, systemIdentifier, []*configv1.ConfigField{
		{Path: "feature.enabled", Value: "true", Type: configv1.FieldType_BOOLEAN},
	}, user)
	require.NoError(t, err)

	// Get value at PROJECT scope with inheritance
	projectIdentifier := scopeconfig.NewIdentifier(serviceName).
		WithScope(configv1.Scope_PROJECT).
		WithGroupID(groupID).
		WithProjectID("proj-123").
		Build()

	value, err := testClient.GetValue(ctx, projectIdentifier, "feature.enabled", &scopeconfig.GetValueOptions{
		Inherit:    true,
		UseDefault: true,
	})
	require.NoError(t, err)
	require.NotNil(t, value)
	assert.Equal(t, "true", *value, "Should inherit value from SYSTEM scope")
}

// TestGetValueString tests the convenience method for getting string values.
func TestGetValueString(t *testing.T) {
	ctx := context.Background()
	serviceName := "string-test-service"
	groupID := "string-group"
	user := "string-test"

	// Apply a template
	template := &configv1.ConfigTemplate{
		Identifier: scopeconfig.NewIdentifier(serviceName).
			WithGroupID(groupID).
			Build(),
		ServiceLabel: "String Test Service",
		GroupLabel:   "String Group",
		Fields: []*configv1.ConfigFieldTemplate{
			{
				Path:         "app.name",
				Label:        "Application Name",
				Type:         configv1.FieldType_STRING,
				DefaultValue: "MyApp",
			},
		},
	}

	_, err := testClient.ApplyConfigTemplate(ctx, template, user)
	require.NoError(t, err)

	identifier := scopeconfig.NewIdentifier(serviceName).
		WithScope(configv1.Scope_SYSTEM).
		WithGroupID(groupID).
		Build()

	// Get default value as string
	value, err := testClient.GetValueString(ctx, identifier, "app.name", &scopeconfig.GetValueOptions{
		UseDefault: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "MyApp", value)

	// Get non-existent value (should return empty string)
	nonExistent, err := testClient.GetValueString(ctx, identifier, "non.existent", nil)
	require.NoError(t, err)
	assert.Equal(t, "", nonExistent)
}

// TestCacheOperations tests cache invalidation and clearing.
func TestCacheOperations(t *testing.T) {
	ctx := context.Background()
	serviceName := "cache-ops-service"
	groupID := "cache-ops"
	user := "cache-ops-test"

	// Apply a template
	template := &configv1.ConfigTemplate{
		Identifier: scopeconfig.NewIdentifier(serviceName).
			WithGroupID(groupID).
			Build(),
		ServiceLabel: "Cache Ops Service",
		GroupLabel:   "Cache Ops Group",
		Fields: []*configv1.ConfigFieldTemplate{
			{
				Path:         "setting",
				Label:        "Setting",
				Type:         configv1.FieldType_STRING,
				DefaultValue: "default",
			},
		},
	}

	_, err := testClient.ApplyConfigTemplate(ctx, template, user)
	require.NoError(t, err)

	identifier := scopeconfig.NewIdentifier(serviceName).
		WithScope(configv1.Scope_SYSTEM).
		WithGroupID(groupID).
		Build()

	// Set initial value
	_, err = testClient.UpdateConfig(ctx, identifier, []*configv1.ConfigField{
		{Path: "setting", Value: "value1", Type: configv1.FieldType_STRING},
	}, user)
	require.NoError(t, err)

	// Get cached
	_, err = testClient.GetConfigCached(ctx, identifier)
	require.NoError(t, err)

	// Invalidate cache
	testClient.InvalidateCache(identifier)

	// Clear all cache
	testClient.ClearCache()

	// Should still work after cache clear
	config, err := testClient.GetConfigCached(ctx, identifier)
	require.NoError(t, err)
	assert.NotNil(t, config)
}

// TestGetLatestConfig tests retrieving the latest configuration version.
func TestGetLatestConfig(t *testing.T) {
	ctx := context.Background()
	serviceName := "latest-config-service"
	groupID := "latest-group"
	user := "latest-test"

	// Apply a template
	template := &configv1.ConfigTemplate{
		Identifier: scopeconfig.NewIdentifier(serviceName).
			WithGroupID(groupID).
			Build(),
		ServiceLabel: "Latest Config Service",
		GroupLabel:   "Latest Group",
		Fields: []*configv1.ConfigFieldTemplate{
			{
				Path:         "version.info",
				Label:        "Version Info",
				Type:         configv1.FieldType_STRING,
				DefaultValue: "1.0.0",
			},
		},
	}

	_, err := testClient.ApplyConfigTemplate(ctx, template, user)
	require.NoError(t, err)

	identifier := scopeconfig.NewIdentifier(serviceName).
		WithScope(configv1.Scope_SYSTEM).
		WithGroupID(groupID).
		Build()

	// Set a value
	_, err = testClient.UpdateConfig(ctx, identifier, []*configv1.ConfigField{
		{Path: "version.info", Value: "1.0.1", Type: configv1.FieldType_STRING},
	}, user)
	require.NoError(t, err)

	// Get latest config
	latestConfig, err := testClient.GetLatestConfig(ctx, identifier)
	require.NoError(t, err)
	assert.NotNil(t, latestConfig)
}

// TestErrorHandling tests error handling for various scenarios.
func TestErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Test getting a non-existent config
	identifier := scopeconfig.NewIdentifier("non-existent-service").
		WithScope(configv1.Scope_SYSTEM).
		WithGroupID("non-existent-group").
		Build()

	_, err := testClient.GetConfig(ctx, identifier)
	assert.Error(t, err, "Should error when getting non-existent config")

	// Test getting a non-existent template
	_, err = testClient.GetConfigTemplate(ctx, identifier)
	assert.Error(t, err, "Should error when getting non-existent template")
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
