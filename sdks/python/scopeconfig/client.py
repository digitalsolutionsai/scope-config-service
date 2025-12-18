"""
gRPC client for the ScopeConfig service with caching support.

Features:
- In-memory caching for config values by group (reduces gRPC calls)
- In-memory caching for templates (for default value lookups)
- Background sync to refresh cached configs periodically
- Stale cache fallback when server is unavailable
- GetValue extracts specific field from cached group config
- GetValue with inheritance and default value support
- Environment variable support for configuration
"""

import os
import logging
from typing import Optional, List
from datetime import datetime

import grpc

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
from .cache import ConfigCache

# Try to import generated proto files
try:
    from . import config_pb2
    from . import config_pb2_grpc
except ImportError:
    config_pb2 = None
    config_pb2_grpc = None

# Environment variable names
ENV_HOST = "GRPC_SCOPE_CONFIG_HOST"
ENV_PORT = "GRPC_SCOPE_CONFIG_PORT"
ENV_USE_TLS = "GRPC_SCOPE_CONFIG_USE_TLS"

# Default values
DEFAULT_HOST = "localhost"
DEFAULT_PORT = 50051
DEFAULT_CACHE_TTL_SECONDS = 60.0
DEFAULT_SYNC_INTERVAL_SECONDS = 30.0

logger = logging.getLogger(__name__)


class ConfigClient:
    """
    ScopeConfig gRPC Client with caching support.
    
    Example:
        # Create client using environment variables
        client = ConfigClient()
        
        # Or with explicit configuration
        client = ConfigClient(
            host="localhost",
            port=50051,
            use_tls=False,
            cache_enabled=True,
            cache_ttl_seconds=60.0,
        )
        
        # Connect to the server
        client.connect()
        
        # Get a specific config value
        value = client.get_value(
            identifier,
            "database.host",
            GetValueOptions(use_default=True, inherit=True)
        )
        
        # Close connection
        client.close()
    """
    
    def __init__(
        self,
        host: Optional[str] = None,
        port: Optional[int] = None,
        use_tls: Optional[bool] = None,
        cache_enabled: bool = True,
        cache_ttl_seconds: float = DEFAULT_CACHE_TTL_SECONDS,
        background_sync_enabled: bool = False,
        background_sync_interval_seconds: float = DEFAULT_SYNC_INTERVAL_SECONDS,
    ):
        """
        Initialize the client.
        
        Args:
            host: Server host (default: from GRPC_SCOPE_CONFIG_HOST or "localhost")
            port: Server port (default: from GRPC_SCOPE_CONFIG_PORT or 50051)
            use_tls: Whether to use TLS (default: from GRPC_SCOPE_CONFIG_USE_TLS or False)
            cache_enabled: Enable in-memory caching (default: True)
            cache_ttl_seconds: Cache TTL in seconds (default: 60)
            background_sync_enabled: Enable background sync (default: False)
            background_sync_interval_seconds: Background sync interval (default: 30)
        """
        # Load configuration from environment variables with fallbacks
        self.host = host or os.getenv(ENV_HOST, DEFAULT_HOST)
        self.port = port or int(os.getenv(ENV_PORT, str(DEFAULT_PORT)))
        
        if use_tls is not None:
            self.use_tls = use_tls
        else:
            env_tls = os.getenv(ENV_USE_TLS, "false").lower()
            self.use_tls = env_tls in ("true", "1", "yes")
        
        self.cache_enabled = cache_enabled
        self.cache_ttl_seconds = cache_ttl_seconds
        self.background_sync_enabled = background_sync_enabled
        self.background_sync_interval_seconds = background_sync_interval_seconds
        
        self._channel: Optional[grpc.Channel] = None
        self._stub = None
        self._cache: Optional[ConfigCache] = None
        
        if cache_enabled:
            self._cache = ConfigCache(cache_ttl_seconds)
    
    @property
    def address(self) -> str:
        """Get the server address."""
        return f"{self.host}:{self.port}"
    
    def connect(self) -> None:
        """Connect to the gRPC server."""
        if config_pb2 is None or config_pb2_grpc is None:
            raise ConfigServiceError(
                "Generated proto files not found. Please run buf generate first.",
                code=grpc.StatusCode.INTERNAL.value[0] if isinstance(grpc.StatusCode.INTERNAL.value, tuple) else grpc.StatusCode.INTERNAL.value,
            )
        
        if self.use_tls:
            self._channel = grpc.secure_channel(self.address, grpc.ssl_channel_credentials())
        else:
            self._channel = grpc.insecure_channel(self.address)
        
        self._stub = config_pb2_grpc.ConfigServiceStub(self._channel)
        
        # Start background sync if enabled
        if self.background_sync_enabled and self._cache:
            self._cache.start_background_sync(
                self.background_sync_interval_seconds,
                self._sync_config
            )
    
    def close(self) -> None:
        """Close the client connection."""
        if self._cache:
            self._cache.stop_background_sync()
        
        if self._channel:
            self._channel.close()
            self._channel = None
            self._stub = None
    
    def __enter__(self):
        """Context manager entry."""
        self.connect()
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()
    
    def _to_proto_identifier(self, identifier: ConfigIdentifier):
        """Convert ConfigIdentifier to proto message."""
        return config_pb2.ConfigIdentifier(
            service_name=identifier.service_name,
            scope=identifier.scope.value,
            group_id=identifier.group_id,
            project_id=identifier.project_id or "",
            store_id=identifier.store_id or "",
            user_id=identifier.user_id or "",
        )
    
    def _from_proto_config(self, proto_config) -> ScopeConfig:
        """Convert proto ScopeConfig to dataclass."""
        version_info = ConfigVersion(
            id=proto_config.version_info.id,
            identifier=self._from_proto_identifier(proto_config.version_info.identifier),
            latest_version=proto_config.version_info.latest_version,
            published_version=proto_config.version_info.published_version,
            created_by=proto_config.version_info.created_by,
            updated_by=proto_config.version_info.updated_by,
        )
        
        fields = [
            ConfigField(
                path=f.path,
                value=f.value,
                type=FieldType(f.type) if f.type in [e.value for e in FieldType] else FieldType.STRING,
            )
            for f in proto_config.fields
        ]
        
        return ScopeConfig(
            version_info=version_info,
            current_version=proto_config.current_version,
            fields=fields,
        )
    
    def _from_proto_identifier(self, proto_id) -> ConfigIdentifier:
        """Convert proto ConfigIdentifier to dataclass."""
        return ConfigIdentifier(
            service_name=proto_id.service_name,
            scope=Scope(proto_id.scope) if proto_id.scope in [e.value for e in Scope] else Scope.SCOPE_UNSPECIFIED,
            group_id=proto_id.group_id,
            project_id=proto_id.project_id or None,
            store_id=proto_id.store_id or None,
            user_id=proto_id.user_id or None,
        )
    
    def _from_proto_template(self, proto_template) -> ConfigTemplate:
        """Convert proto ConfigTemplate to dataclass."""
        fields = [
            ConfigFieldTemplate(
                path=f.path,
                label=f.label,
                description=f.description,
                type=FieldType(f.type) if f.type in [e.value for e in FieldType] else FieldType.STRING,
                default_value=f.default_value,
                display_on=[Scope(s) for s in f.display_on if s in [e.value for e in Scope]],
                options=[ValueOption(value=o.value, label=o.label) for o in f.options],
                sort_order=f.sort_order,
            )
            for f in proto_template.fields
        ]
        
        return ConfigTemplate(
            identifier=self._from_proto_identifier(proto_template.identifier),
            service_label=proto_template.service_label,
            group_label=proto_template.group_label,
            group_description=proto_template.group_description,
            fields=fields,
            sort_order=proto_template.sort_order,
        )
    
    def _wrap_error(self, method: str, error: grpc.RpcError) -> ConfigServiceError:
        """Wrap gRPC errors with additional context."""
        code = error.code()
        details = error.details()
        code_value = code.value[0] if isinstance(code.value, tuple) else code.value
        
        if code == grpc.StatusCode.NOT_FOUND:
            return ConfigServiceError(f"{method}: resource not found: {details}", code_value, details)
        elif code == grpc.StatusCode.INVALID_ARGUMENT:
            return ConfigServiceError(f"{method}: invalid argument: {details}", code_value, details)
        elif code == grpc.StatusCode.UNAVAILABLE:
            return ConfigServiceError(f"{method}: service unavailable: {details}", code_value, details)
        else:
            return ConfigServiceError(f"{method} failed: {details}", code_value, details)
    
    def _sync_config(self, identifier: ConfigIdentifier) -> None:
        """Sync a config from the server (for background sync)."""
        try:
            config = self.get_config(identifier)
            if self._cache:
                self._cache.set(identifier, config)
        except Exception as e:
            logger.warning(f"Background sync failed for {identifier.service_name}/{identifier.group_id}: {e}")
    
    # === Public API ===
    
    def get_config(self, identifier: ConfigIdentifier) -> ScopeConfig:
        """
        Get the published configuration for the given identifier.
        Always fetches from the server.
        """
        if not self._stub:
            raise ConfigServiceError("Client not connected. Call connect() first.")
        
        try:
            request = config_pb2.GetConfigRequest(
                identifier=self._to_proto_identifier(identifier)
            )
            response = self._stub.GetConfig(request)
            config = self._from_proto_config(response)
            
            # Update cache if enabled
            if self._cache:
                self._cache.set(identifier, config)
            
            return config
        except grpc.RpcError as e:
            raise self._wrap_error("GetConfig", e)
    
    def get_config_cached(self, identifier: ConfigIdentifier) -> ScopeConfig:
        """
        Get configuration with caching support.
        Returns cached value if valid, falls back to stale cache on error.
        """
        # Try cache first
        if self._cache:
            cached, is_valid = self._cache.get(identifier)
            if cached and is_valid:
                return cached
        
        # Fetch from server
        try:
            return self.get_config(identifier)
        except ConfigServiceError as e:
            # On error, try stale cache
            if self._cache:
                stale = self._cache.get_stale(identifier)
                if stale:
                    logger.warning(f"Using stale cache for {identifier.service_name}/{identifier.group_id}")
                    return stale
            raise
    
    def get_latest_config(self, identifier: ConfigIdentifier) -> ScopeConfig:
        """Get the latest configuration (published or not)."""
        if not self._stub:
            raise ConfigServiceError("Client not connected. Call connect() first.")
        
        try:
            request = config_pb2.GetConfigRequest(
                identifier=self._to_proto_identifier(identifier)
            )
            response = self._stub.GetLatestConfig(request)
            return self._from_proto_config(response)
        except grpc.RpcError as e:
            raise self._wrap_error("GetLatestConfig", e)
    
    def get_config_template(self, identifier: ConfigIdentifier) -> ConfigTemplate:
        """
        Get the configuration template for the given identifier.
        Always fetches from the server.
        """
        if not self._stub:
            raise ConfigServiceError("Client not connected. Call connect() first.")
        
        try:
            request = config_pb2.GetConfigTemplateRequest(
                identifier=self._to_proto_identifier(identifier)
            )
            response = self._stub.GetConfigTemplate(request)
            template = self._from_proto_template(response)
            
            # Update cache if enabled
            if self._cache:
                self._cache.set_template(identifier, template)
            
            return template
        except grpc.RpcError as e:
            raise self._wrap_error("GetConfigTemplate", e)
    
    def get_config_template_cached(self, identifier: ConfigIdentifier) -> ConfigTemplate:
        """
        Get configuration template with caching support.
        Templates are cached for default value lookups.
        """
        # Try cache first
        if self._cache:
            cached, is_valid = self._cache.get_template(identifier)
            if cached and is_valid:
                return cached
        
        # Fetch from server
        try:
            return self.get_config_template(identifier)
        except ConfigServiceError as e:
            # On error, try stale cache
            if self._cache:
                stale = self._cache.get_template_stale(identifier)
                if stale:
                    return stale
            raise
    
    def get_value(
        self,
        identifier: ConfigIdentifier,
        path: str,
        options: Optional[GetValueOptions] = None
    ) -> Optional[str]:
        """
        Get a specific configuration value by path.
        
        This method is optimized to reduce gRPC calls:
        - Config values are fetched by group and cached
        - Templates are cached for default value lookups
        
        Args:
            identifier: Config identifier
            path: Field path (e.g., "database.host")
            options: GetValue options (use_default, inherit)
            
        Returns:
            The value as a string, or None if not found
        """
        opts = options or GetValueOptions()
        
        # Try to get value from current scope
        value = self._get_value_from_scope(identifier, path)
        if value is not None:
            return value
        
        # If inherit is enabled, try parent scopes
        if opts.inherit:
            parent_identifiers = self._get_parent_identifiers(identifier)
            for parent_id in parent_identifiers:
                value = self._get_value_from_scope(parent_id, path)
                if value is not None:
                    return value
        
        # If use_default is enabled, try to get default from template
        if opts.use_default:
            default_value = self._get_default_value(identifier, path)
            if default_value is not None:
                return default_value
        
        return None
    
    def get_value_string(
        self,
        identifier: ConfigIdentifier,
        path: str,
        options: Optional[GetValueOptions] = None
    ) -> str:
        """
        Convenience method that returns an empty string instead of None.
        """
        value = self.get_value(identifier, path, options)
        return value if value is not None else ""
    
    def _get_value_from_scope(self, identifier: ConfigIdentifier, path: str) -> Optional[str]:
        """Get a value from a specific scope's configuration."""
        try:
            config = self.get_config_cached(identifier)
            for field in config.fields:
                if field.path == path:
                    return field.value
            return None
        except Exception:
            return None
    
    def _get_default_value(self, identifier: ConfigIdentifier, path: str) -> Optional[str]:
        """Get the default value from the configuration template."""
        try:
            template = self.get_config_template_cached(identifier)
            for field in template.fields:
                if field.path == path:
                    return field.default_value
            return None
        except Exception:
            return None
    
    def _get_parent_identifiers(self, identifier: ConfigIdentifier) -> List[ConfigIdentifier]:
        """
        Get parent scope identifiers for inheritance.
        
        The inheritance hierarchy is:
            SYSTEM
            ├── PROJECT → STORE
            └── USER
            
        So: STORE → PROJECT → SYSTEM, USER → SYSTEM, PROJECT → SYSTEM
        """
        parents = []
        
        if identifier.scope == Scope.USER:
            # User -> System (USER is at same level as PROJECT, not under STORE)
            parents.append(ConfigIdentifier(
                service_name=identifier.service_name,
                group_id=identifier.group_id,
                scope=Scope.SYSTEM,
            ))
        
        elif identifier.scope == Scope.STORE:
            # Store -> Project -> System
            if identifier.project_id:
                parents.append(ConfigIdentifier(
                    service_name=identifier.service_name,
                    group_id=identifier.group_id,
                    scope=Scope.PROJECT,
                    project_id=identifier.project_id,
                ))
            parents.append(ConfigIdentifier(
                service_name=identifier.service_name,
                group_id=identifier.group_id,
                scope=Scope.SYSTEM,
            ))
        
        elif identifier.scope == Scope.PROJECT:
            # Project -> System
            parents.append(ConfigIdentifier(
                service_name=identifier.service_name,
                group_id=identifier.group_id,
                scope=Scope.SYSTEM,
            ))
        
        # SYSTEM has no parent
        
        return parents
    
    def invalidate_cache(self, identifier: ConfigIdentifier) -> None:
        """Invalidate the cache for a specific identifier."""
        if self._cache:
            self._cache.invalidate(identifier)
    
    def clear_cache(self) -> None:
        """Clear all cached configurations."""
        if self._cache:
            self._cache.clear()
    
    def is_cache_enabled(self) -> bool:
        """Check if caching is enabled."""
        return self._cache is not None
    
    def apply_config_template(self, template: ConfigTemplate, user: str) -> ConfigTemplate:
        """
        Apply a configuration template.
        
        Args:
            template: The template to apply
            user: The user performing the action
            
        Returns:
            The applied template
        """
        if not self._stub:
            raise ConfigServiceError("Client not connected. Call connect() first.")
        
        try:
            # Convert to proto
            proto_fields = []
            for f in template.fields:
                proto_options = [
                    config_pb2.ValueOption(value=o.value, label=o.label)
                    for o in f.options
                ]
                proto_fields.append(config_pb2.ConfigFieldTemplate(
                    path=f.path,
                    label=f.label,
                    description=f.description,
                    type=f.type.value,
                    default_value=f.default_value,
                    display_on=[s.value for s in f.display_on],
                    options=proto_options,
                    sort_order=f.sort_order,
                ))
            
            proto_template = config_pb2.ConfigTemplate(
                identifier=self._to_proto_identifier(template.identifier),
                service_label=template.service_label,
                group_label=template.group_label,
                group_description=template.group_description,
                fields=proto_fields,
                sort_order=template.sort_order,
            )
            
            request = config_pb2.ApplyConfigTemplateRequest(
                template=proto_template,
                user=user,
            )
            response = self._stub.ApplyConfigTemplate(request)
            return self._from_proto_template(response)
        except grpc.RpcError as e:
            raise self._wrap_error("ApplyConfigTemplate", e)


def load_templates_from_dir(client: ConfigClient, dir_path: str, user: str) -> None:
    """
    Load and apply all YAML templates from a directory.
    
    Simply place your template files in the specified directory and this function
    will automatically load and apply them to the config service.
    
    Args:
        client: The connected ConfigClient instance
        dir_path: Path to the templates directory
        user: The user performing the action
        
    Example:
        # Initialize client and auto-load templates
        with ConfigClient() as client:
            load_templates_from_dir(client, "./templates", "system")
    """
    import os
    import glob
    
    try:
        import yaml
    except ImportError:
        raise ConfigServiceError("PyYAML is required for template loading. Install with: pip install pyyaml")
    
    if not os.path.isdir(dir_path):
        logger.info(f"Templates directory {dir_path} does not exist, skipping template import")
        return
    
    # Find all YAML files
    yaml_files = glob.glob(os.path.join(dir_path, "*.yaml")) + glob.glob(os.path.join(dir_path, "*.yml"))
    
    if not yaml_files:
        logger.info(f"No template files found in {dir_path}")
        return
    
    logger.info(f"Found {len(yaml_files)} template file(s) to import")
    
    for file_path in yaml_files:
        _load_and_apply_template_file(client, file_path, user)


def _load_and_apply_template_file(client: ConfigClient, file_path: str, user: str) -> None:
    """Load and apply a single template file."""
    import yaml
    
    try:
        with open(file_path, 'r') as f:
            data = yaml.safe_load(f)
    except Exception as e:
        raise ConfigServiceError(f"Failed to read template file {file_path}: {e}")
    
    if not data:
        logger.warning(f"Empty template file: {file_path}")
        return
    
    # Validate required fields
    if 'service' not in data or 'id' not in data['service']:
        raise ConfigServiceError(f"Template file {file_path} missing 'service.id'")
    
    service_name = data['service']['id']
    service_label = data['service'].get('label', service_name)
    
    groups = data.get('groups', [])
    if not groups:
        logger.warning(f"No groups defined in template: {file_path}")
        return
    
    for group in groups:
        _apply_group_template(client, service_name, service_label, group, user)
        logger.info(f"Successfully imported template: service={service_name}, group={group.get('id')} from {os.path.basename(file_path)}")


def _apply_group_template(client: ConfigClient, service_name: str, service_label: str, group: dict, user: str) -> None:
    """Apply a single group template."""
    group_id = group.get('id', '')
    group_label = group.get('label', group_id)
    group_description = group.get('description', '')
    sort_order = group.get('sortOrder', 0)
    
    fields = []
    for f in group.get('fields', []):
        display_on = [_to_scope(s) for s in f.get('displayOn', [])]
        options = [ValueOption(value=o['value'], label=o.get('label', o['value'])) for o in f.get('options', [])]
        
        fields.append(ConfigFieldTemplate(
            path=f.get('path', ''),
            label=f.get('label', ''),
            description=f.get('description', ''),
            type=_to_field_type(f.get('type', 'STRING')),
            default_value=f.get('defaultValue', ''),
            display_on=display_on,
            options=options,
            sort_order=f.get('sortOrder', 0),
        ))
    
    template = ConfigTemplate(
        identifier=ConfigIdentifier(
            service_name=service_name,
            group_id=group_id,
            scope=Scope.SCOPE_UNSPECIFIED,
        ),
        service_label=service_label,
        group_label=group_label,
        group_description=group_description,
        fields=fields,
        sort_order=sort_order,
    )
    
    client.apply_config_template(template, user)


def _to_scope(s: str) -> Scope:
    """Convert string to Scope enum."""
    scope_map = {
        'SYSTEM': Scope.SYSTEM,
        'PROJECT': Scope.PROJECT,
        'STORE': Scope.STORE,
        'USER': Scope.USER,
    }
    return scope_map.get(s.upper(), Scope.SCOPE_UNSPECIFIED)


def _to_field_type(t: str) -> FieldType:
    """Convert string to FieldType enum."""
    type_map = {
        'STRING': FieldType.STRING,
        'INT': FieldType.INT,
        'FLOAT': FieldType.FLOAT,
        'BOOLEAN': FieldType.BOOLEAN,
        'JSON': FieldType.JSON,
        'ARRAY_STRING': FieldType.ARRAY_STRING,
        'SECRET': FieldType.SECRET,
    }
    return type_map.get(t.upper(), FieldType.STRING)
