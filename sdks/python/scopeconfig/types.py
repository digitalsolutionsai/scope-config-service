"""
Type definitions for the ScopeConfig SDK.
"""

from enum import IntEnum
from dataclasses import dataclass, field
from typing import Optional, List
from datetime import datetime


class Scope(IntEnum):
    """Scope levels for configuration."""
    SCOPE_UNSPECIFIED = 0
    SYSTEM = 1
    PROJECT = 2
    STORE = 3
    USER = 4


class FieldType(IntEnum):
    """Configuration field types."""
    FIELD_TYPE_UNSPECIFIED = 0
    STRING = 1
    INT = 2
    FLOAT = 3
    BOOLEAN = 4
    JSON = 5
    ARRAY_STRING = 6
    SECRET = 7


@dataclass
class ConfigIdentifier:
    """A unique identifier for a configuration set."""
    service_name: str
    scope: Scope
    group_id: str
    project_id: Optional[str] = None
    store_id: Optional[str] = None
    user_id: Optional[str] = None


@dataclass
class ConfigField:
    """A specific configuration field with its value."""
    path: str
    value: str
    type: FieldType = FieldType.STRING


@dataclass
class ConfigVersion:
    """Represents a configuration version."""
    id: int
    identifier: ConfigIdentifier
    latest_version: int
    published_version: int
    created_at: Optional[datetime] = None
    created_by: str = ""
    updated_at: Optional[datetime] = None
    updated_by: str = ""


@dataclass
class ScopeConfig:
    """Represents a complete configuration set at a specific version."""
    version_info: ConfigVersion
    current_version: int
    fields: List[ConfigField] = field(default_factory=list)


@dataclass
class ValueOption:
    """A predefined choice for a configuration field."""
    value: str
    label: str


@dataclass
class ConfigFieldTemplate:
    """Defines the schema for a single configuration field."""
    path: str
    label: str
    description: str
    type: FieldType
    default_value: str = ""
    display_on: List[Scope] = field(default_factory=list)
    options: List[ValueOption] = field(default_factory=list)
    sort_order: int = 0


@dataclass
class ConfigTemplate:
    """A container that groups all field templates for a specific service and group."""
    identifier: ConfigIdentifier
    service_label: str
    group_label: str
    group_description: str
    fields: List[ConfigFieldTemplate] = field(default_factory=list)
    sort_order: int = 0


@dataclass
class GetValueOptions:
    """Options for retrieving a configuration value."""
    # Use default value from template if config value is not set
    use_default: bool = False
    # Traverse parent scopes to find the value
    # Hierarchy: STORE → PROJECT → SYSTEM, USER → SYSTEM
    inherit: bool = False


class ConfigServiceError(Exception):
    """Error wrapper for gRPC errors."""
    
    def __init__(self, message: str, code: int = 0, details: Optional[str] = None):
        super().__init__(message)
        self.code = code
        self.details = details
