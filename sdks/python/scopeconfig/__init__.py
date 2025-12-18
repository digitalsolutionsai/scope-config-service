"""
ScopeConfig Python SDK

A Python client for the ScopeConfig gRPC service with caching support.

Features:
- In-memory caching for config values by group (reduces gRPC calls)
- In-memory caching for templates (for default value lookups)
- Background sync to refresh cached configs periodically
- Stale cache fallback when server is unavailable
- GetValue extracts specific field from cached group config
- GetValue with inheritance and default value support
- Environment variable support for configuration
"""

from .client import ConfigClient
from .cache import ConfigCache
from .types import (
    Scope,
    FieldType,
    ConfigIdentifier,
    ConfigField,
    ConfigVersion,
    ScopeConfig,
    ConfigTemplate,
    ConfigFieldTemplate,
    ValueOption,
    GetValueOptions,
    ConfigServiceError,
)
from .identifier import IdentifierBuilder, create_identifier

__all__ = [
    "ConfigClient",
    "ConfigCache",
    "Scope",
    "FieldType",
    "ConfigIdentifier",
    "ConfigField",
    "ConfigVersion",
    "ScopeConfig",
    "ConfigTemplate",
    "ConfigFieldTemplate",
    "ValueOption",
    "GetValueOptions",
    "ConfigServiceError",
    "IdentifierBuilder",
    "create_identifier",
]

__version__ = "1.0.0"
