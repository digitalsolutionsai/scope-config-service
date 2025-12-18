package scopeconfig

import (
	"context"
	"fmt"

	configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
)

// GetValueOptions provides options for retrieving a configuration value.
type GetValueOptions struct {
	// UseDefault returns the default value from the template if the config value is not set.
	// Note: Templates are cached to reduce gRPC calls when looking up default values.
	UseDefault bool

	// Inherit traverses parent scopes to find the value if not found in the current scope.
	// The inheritance order is: USER -> STORE -> PROJECT -> SYSTEM
	Inherit bool
}

/*
GetValue retrieves a specific configuration value by path.
Returns the value as a string, or nil if not found based on the options.

This method is optimized to reduce gRPC calls:
  - Config values are fetched by group and cached, then the specific field is extracted locally
  - Templates are cached for default value lookups

Example:

	value, err := client.GetValue(ctx, identifier, "database.host", &GetValueOptions{
	    UseDefault: true,
	    Inherit:    true,
	})
*/
func (c *Client) GetValue(ctx context.Context, identifier *configv1.ConfigIdentifier, path string, opts *GetValueOptions) (*string, error) {
	if opts == nil {
		opts = &GetValueOptions{}
	}

	// Try to get value from current scope (uses cached group config)
	value, err := c.getValueFromScope(ctx, identifier, path)
	if err == nil && value != nil {
		return value, nil
	}

	// If inherit is enabled, try parent scopes
	if opts.Inherit {
		parentIdentifiers := getParentIdentifiers(identifier)
		for _, parentID := range parentIdentifiers {
			value, err := c.getValueFromScope(ctx, parentID, path)
			if err == nil && value != nil {
				return value, nil
			}
		}
	}

	// If useDefault is enabled, try to get default from template (uses cached template)
	if opts.UseDefault {
		defaultValue, err := c.getDefaultValue(ctx, identifier, path)
		if err == nil && defaultValue != nil {
			return defaultValue, nil
		}
	}

	return nil, nil
}

// getValueFromScope retrieves a value from a specific scope's configuration.
func (c *Client) getValueFromScope(ctx context.Context, identifier *configv1.ConfigIdentifier, path string) (*string, error) {
	config, err := c.GetConfigCached(ctx, identifier)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, nil
	}

	for _, field := range config.Fields {
		if field.Path == path {
			return &field.Value, nil
		}
	}

	return nil, nil
}

// getDefaultValue retrieves the default value from the configuration template.
// Uses cached template to reduce gRPC calls.
func (c *Client) getDefaultValue(ctx context.Context, identifier *configv1.ConfigIdentifier, path string) (*string, error) {
	template, err := c.GetConfigTemplateCached(ctx, identifier)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, nil
	}

	for _, field := range template.Fields {
		if field.Path == path {
			// Return default value even if it's empty string
			// (empty string can be a valid default)
			return &field.DefaultValue, nil
		}
	}

	return nil, nil
}

// getParentIdentifiers returns the parent scope identifiers in inheritance order.
// The order is from most specific to most general: USER -> STORE -> PROJECT -> SYSTEM
func getParentIdentifiers(identifier *configv1.ConfigIdentifier) []*configv1.ConfigIdentifier {
	var parents []*configv1.ConfigIdentifier

	switch identifier.Scope {
	case configv1.Scope_USER:
		// User -> Store -> Project -> System
		if identifier.StoreId != "" {
			parents = append(parents, &configv1.ConfigIdentifier{
				ServiceName: identifier.ServiceName,
				GroupId:     identifier.GroupId,
				Scope:       configv1.Scope_STORE,
				ProjectId:   identifier.ProjectId,
				StoreId:     identifier.StoreId,
			})
		}
		if identifier.ProjectId != "" {
			parents = append(parents, &configv1.ConfigIdentifier{
				ServiceName: identifier.ServiceName,
				GroupId:     identifier.GroupId,
				Scope:       configv1.Scope_PROJECT,
				ProjectId:   identifier.ProjectId,
			})
		}
		parents = append(parents, &configv1.ConfigIdentifier{
			ServiceName: identifier.ServiceName,
			GroupId:     identifier.GroupId,
			Scope:       configv1.Scope_SYSTEM,
		})

	case configv1.Scope_STORE:
		// Store -> Project -> System
		if identifier.ProjectId != "" {
			parents = append(parents, &configv1.ConfigIdentifier{
				ServiceName: identifier.ServiceName,
				GroupId:     identifier.GroupId,
				Scope:       configv1.Scope_PROJECT,
				ProjectId:   identifier.ProjectId,
			})
		}
		parents = append(parents, &configv1.ConfigIdentifier{
			ServiceName: identifier.ServiceName,
			GroupId:     identifier.GroupId,
			Scope:       configv1.Scope_SYSTEM,
		})

	case configv1.Scope_PROJECT:
		// Project -> System
		parents = append(parents, &configv1.ConfigIdentifier{
			ServiceName: identifier.ServiceName,
			GroupId:     identifier.GroupId,
			Scope:       configv1.Scope_SYSTEM,
		})

	case configv1.Scope_SYSTEM:
		// System has no parent
	}

	return parents
}

// GetValueString is a convenience method that returns an empty string instead of nil.
func (c *Client) GetValueString(ctx context.Context, identifier *configv1.ConfigIdentifier, path string, opts *GetValueOptions) (string, error) {
	value, err := c.GetValue(ctx, identifier, path, opts)
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", nil
	}
	return *value, nil
}

// MustGetValue retrieves a value and panics if an error occurs.
// Returns an empty string if the value is not found.
func (c *Client) MustGetValue(ctx context.Context, identifier *configv1.ConfigIdentifier, path string, opts *GetValueOptions) string {
	value, err := c.GetValueString(ctx, identifier, path, opts)
	if err != nil {
		panic(fmt.Sprintf("failed to get config value %s: %v", path, err))
	}
	return value
}
