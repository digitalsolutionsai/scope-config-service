"""
Identifier builder for creating ConfigIdentifier objects.
"""

from .types import ConfigIdentifier, Scope


class IdentifierBuilder:
    """
    Builder for creating ConfigIdentifier objects.
    
    Example:
        identifier = (
            IdentifierBuilder("my-service")
            .with_scope(Scope.PROJECT)
            .with_group_id("database")
            .with_project_id("proj-123")
            .build()
        )
    """
    
    def __init__(self, service_name: str):
        self._service_name = service_name
        self._scope = Scope.SCOPE_UNSPECIFIED
        self._group_id = ""
        self._project_id = None
        self._store_id = None
        self._user_id = None
    
    def with_scope(self, scope: Scope) -> "IdentifierBuilder":
        """Set the scope."""
        self._scope = scope
        return self
    
    def with_group_id(self, group_id: str) -> "IdentifierBuilder":
        """Set the group ID."""
        self._group_id = group_id
        return self
    
    def with_project_id(self, project_id: str) -> "IdentifierBuilder":
        """Set the project ID."""
        self._project_id = project_id
        return self
    
    def with_store_id(self, store_id: str) -> "IdentifierBuilder":
        """Set the store ID."""
        self._store_id = store_id
        return self
    
    def with_user_id(self, user_id: str) -> "IdentifierBuilder":
        """Set the user ID."""
        self._user_id = user_id
        return self
    
    def build(self) -> ConfigIdentifier:
        """Build the ConfigIdentifier."""
        return ConfigIdentifier(
            service_name=self._service_name,
            scope=self._scope,
            group_id=self._group_id,
            project_id=self._project_id,
            store_id=self._store_id,
            user_id=self._user_id,
        )


def create_identifier(service_name: str) -> IdentifierBuilder:
    """Create a new IdentifierBuilder."""
    return IdentifierBuilder(service_name)
