/*
Example of using Go client for the ScopeConfig service using gRPC.
*/

package scopeconfig

import (
    "context"
    "fmt"
    configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// Client is a gRPC client for the ScopeConfig service.
type Client struct {
    conn   *grpc.ClientConn
    client configv1.ConfigServiceClient
}

/*
NewClient creates a new ScopeConfig client with the provided options.

Example:

    client, err := NewClient(
        WithAddress("localhost:50051"),
        WithInsecure(),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
*/
func NewClient(opts ...ClientOption) (*Client, error) {
    cfg := &clientConfig{}

    // Apply all options
    for _, opt := range opts {
        opt(cfg)
    }

    // Validate configuration
    if cfg.address == "" {
        return nil, fmt.Errorf("address is required")
    }

    if len(cfg.dialOptions) == 0 {
        return nil, fmt.Errorf("transport credentials required (use WithInsecure() or WithTLS())")
    }

    // Establish connection
    conn, err := grpc.NewClient(cfg.address, cfg.dialOptions...)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to %s: %w", cfg.address, err)
    }

    return &Client{
        conn:   conn,
        client: configv1.NewConfigServiceClient(conn),
    }, nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
    if c.conn != nil {
        return c.conn.Close()
    }
    return nil
}

/*
GetConfig retrieves the published configuration for the given identifier.

Example:

    identifier := NewIdentifier("my-service").
        WithScope(configv1.Scope_SYSTEM).
        WithGroupID("my-group").
        Build()
    config, err := client.GetConfig(ctx, identifier)
*/
func (c *Client) GetConfig(ctx context.Context, identifier *configv1.ConfigIdentifier) (*configv1.ScopeConfig, error) {
    req := &configv1.GetConfigRequest{
        Identifier: identifier,
    }

    resp, err := c.client.GetConfig(ctx, req)
    if err != nil {
        return nil, wrapError("GetConfig", err)
    }

    return resp, nil
}

// GetLatestConfig retrieves the latest configuration (published or not) for the given identifier.
func (c *Client) GetLatestConfig(ctx context.Context, identifier *configv1.ConfigIdentifier) (*configv1.ScopeConfig, error) {
    req := &configv1.GetConfigRequest{
        Identifier: identifier,
    }

    resp, err := c.client.GetLatestConfig(ctx, req)
    if err != nil {
        return nil, wrapError("GetLatestConfig", err)
    }

    return resp, nil
}

/*
UpdateConfig creates or updates a configuration with the provided fields.

Example:

    identifier := NewIdentifier("my-service").
        WithScope(configv1.Scope_SYSTEM).
        WithGroupID("my-group").
        Build()
    fields := []*configv1.ConfigField{
        {Path: "log.level", Value: "INFO", Type: configv1.FieldType_STRING},
    }
    config, err := client.UpdateConfig(ctx, identifier, fields, "user@example.com")
*/
func (c *Client) UpdateConfig(
    ctx context.Context,
    identifier *configv1.ConfigIdentifier,
    fields []*configv1.ConfigField,
    user string,
) (*configv1.ScopeConfig, error) {
    req := &configv1.UpdateConfigRequest{
        Identifier: identifier,
        Fields:     fields,
        User:       user,
    }

    resp, err := c.client.UpdateConfig(ctx, req)
    if err != nil {
        return nil, wrapError("UpdateConfig", err)
    }

    return resp, nil
}

/*
GetConfigTemplate retrieves the configuration template for the given identifier.

Example:

    identifier := NewIdentifier("my-service").
        WithGroupID("my-group").
        Build()
    template, err := client.GetConfigTemplate(ctx, identifier)
*/
func (c *Client) GetConfigTemplate(ctx context.Context, identifier *configv1.ConfigIdentifier) (*configv1.ConfigTemplate, error) {
    req := &configv1.GetConfigTemplateRequest{
        Identifier: identifier,
    }

    resp, err := c.client.GetConfigTemplate(ctx, req)
    if err != nil {
        return nil, wrapError("GetConfigTemplate", err)
    }

    return resp, nil
}

/*
ApplyConfigTemplate applies a configuration template to the service.

Example:

    template := &configv1.ConfigTemplate{
        Identifier: NewIdentifier("my-service").WithGroupID("my-group").Build(),
        ServiceLabel: "My Service",
        GroupLabel: "My Group",
        GroupDescription: "Configuration for my service",
        Fields: []*configv1.ConfigFieldTemplate{
            {
                Path: "log.level",
                Label: "Log Level",
                Description: "Application logging level",
                Type: configv1.FieldType_STRING,
                DefaultValue: "INFO",
            },
        },
    }
    result, err := client.ApplyConfigTemplate(ctx, template, "user@example.com")
*/
func (c *Client) ApplyConfigTemplate(
    ctx context.Context,
    template *configv1.ConfigTemplate,
    user string,
) (*configv1.ConfigTemplate, error) {
    req := &configv1.ApplyConfigTemplateRequest{
        Template: template,
        User:     user,
    }

    resp, err := c.client.ApplyConfigTemplate(ctx, req)
    if err != nil {
        return nil, wrapError("ApplyConfigTemplate", err)
    }

    return resp, nil
}

/*
Additional methods that can be implemented when needed:

- GetConfigByVersion: Retrieve a specific version of a configuration
- GetConfigHistory: Get version history for a configuration
- PublishVersion: Mark a version as published
- DeleteConfig: Delete a configuration
*/

// wrapError wraps gRPC errors with additional context.
func wrapError(method string, err error) error {
    if err == nil {
        return nil
    }

    st, ok := status.FromError(err)
    if !ok {
        return fmt.Errorf("%s failed: %w", method, err)
    }

    switch st.Code() {
    case codes.NotFound:
        return fmt.Errorf("%s: resource not found: %s", method, st.Message())
    case codes.InvalidArgument:
        return fmt.Errorf("%s: invalid argument: %s", method, st.Message())
    case codes.AlreadyExists:
        return fmt.Errorf("%s: resource already exists: %s", method, st.Message())
    case codes.PermissionDenied:
        return fmt.Errorf("%s: permission denied: %s", method, st.Message())
    case codes.Unavailable:
        return fmt.Errorf("%s: service unavailable: %s", method, st.Message())
    default:
        return fmt.Errorf("%s failed with %s: %s", method, st.Code(), st.Message())
    }
}