package scopeconfig

import (
	configv1 "github.com/digitalsolutionsai/scope-config-service/sdks/go/gen/config/v1"
)

// IdentifierBuilder provides a fluent API for building ConfigIdentifier objects.
type IdentifierBuilder struct {
	identifier *configv1.ConfigIdentifier
}

// NewIdentifier creates a new IdentifierBuilder with the required service name.
func NewIdentifier(serviceName string) *IdentifierBuilder {
	return &IdentifierBuilder{
		identifier: &configv1.ConfigIdentifier{
			ServiceName: serviceName,
			Scope:       configv1.Scope_SCOPE_UNSPECIFIED,
		},
	}
}

// WithScope sets the scope for the configuration.
func (b *IdentifierBuilder) WithScope(scope configv1.Scope) *IdentifierBuilder {
	b.identifier.Scope = scope
	return b
}

// WithGroupID sets the group ID.
func (b *IdentifierBuilder) WithGroupID(groupID string) *IdentifierBuilder {
	b.identifier.GroupId = groupID
	return b
}

// WithProjectID sets the project ID.
func (b *IdentifierBuilder) WithProjectID(projectID string) *IdentifierBuilder {
	b.identifier.ProjectId = projectID
	return b
}

// WithStoreID sets the store ID.
func (b *IdentifierBuilder) WithStoreID(storeID string) *IdentifierBuilder {
	b.identifier.StoreId = storeID
	return b
}

// WithUserID sets the user ID.
func (b *IdentifierBuilder) WithUserID(userID string) *IdentifierBuilder {
	b.identifier.UserId = userID
	return b
}

// Build returns the constructed ConfigIdentifier.
func (b *IdentifierBuilder) Build() *configv1.ConfigIdentifier {
	return b.identifier
}
