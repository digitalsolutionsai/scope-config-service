"""
In-memory cache for configuration values and templates.
"""

import logging
import threading
import time
from typing import Optional, Tuple, Dict, Callable, List
from .types import ConfigIdentifier, ScopeConfig, ConfigTemplate, Scope


class CacheEntry:
    """Represents a cached item with its expiration time."""
    
    def __init__(self, data, expires_at: float):
        self.data = data
        self.expires_at = expires_at
    
    def is_valid(self) -> bool:
        """Check if the cache entry is still valid."""
        return time.time() < self.expires_at


class ConfigCache:
    """
    Configuration cache with TTL support and stale fallback.
    
    - Config values are cached by group to reduce gRPC calls
    - Templates are cached for default value lookups
    """
    
    def __init__(self, ttl_seconds: float = 60.0):
        """
        Initialize the cache.
        
        Args:
            ttl_seconds: Time-to-live for cache entries in seconds (default: 60)
        """
        self.ttl_seconds = ttl_seconds
        self._configs: Dict[str, CacheEntry] = {}
        self._templates: Dict[str, CacheEntry] = {}
        self._lock = threading.RLock()
        self._sync_thread: Optional[threading.Thread] = None
        self._sync_stop_event = threading.Event()
    
    def _config_key(self, identifier: ConfigIdentifier) -> str:
        """Generate a unique cache key for a config identifier."""
        return (
            f"{identifier.service_name}:{identifier.group_id}:{identifier.scope.value}:"
            f"{identifier.project_id or ''}:{identifier.store_id or ''}:{identifier.user_id or ''}"
        )
    
    def _template_key(self, identifier: ConfigIdentifier) -> str:
        """Generate a unique cache key for a template identifier."""
        return f"template:{identifier.service_name}:{identifier.group_id}"
    
    def get(self, identifier: ConfigIdentifier) -> Tuple[Optional[ScopeConfig], bool]:
        """
        Get a config from cache.
        
        Returns:
            Tuple of (config, is_valid) - config may be stale if is_valid is False
        """
        with self._lock:
            key = self._config_key(identifier)
            entry = self._configs.get(key)
            
            if entry is None:
                return None, False
            
            return entry.data, entry.is_valid()
    
    def get_stale(self, identifier: ConfigIdentifier) -> Optional[ScopeConfig]:
        """Get a config from cache even if expired."""
        with self._lock:
            key = self._config_key(identifier)
            entry = self._configs.get(key)
            return entry.data if entry else None
    
    def set(self, identifier: ConfigIdentifier, config: ScopeConfig) -> None:
        """Store a config in the cache."""
        with self._lock:
            key = self._config_key(identifier)
            self._configs[key] = CacheEntry(config, time.time() + self.ttl_seconds)
    
    def get_template(self, identifier: ConfigIdentifier) -> Tuple[Optional[ConfigTemplate], bool]:
        """
        Get a template from cache.
        
        Returns:
            Tuple of (template, is_valid) - template may be stale if is_valid is False
        """
        with self._lock:
            key = self._template_key(identifier)
            entry = self._templates.get(key)
            
            if entry is None:
                return None, False
            
            return entry.data, entry.is_valid()
    
    def get_template_stale(self, identifier: ConfigIdentifier) -> Optional[ConfigTemplate]:
        """Get a template from cache even if expired."""
        with self._lock:
            key = self._template_key(identifier)
            entry = self._templates.get(key)
            return entry.data if entry else None
    
    def set_template(self, identifier: ConfigIdentifier, template: ConfigTemplate) -> None:
        """Store a template in the cache."""
        with self._lock:
            key = self._template_key(identifier)
            self._templates[key] = CacheEntry(template, time.time() + self.ttl_seconds)
    
    def invalidate(self, identifier: ConfigIdentifier) -> None:
        """Remove a specific config from the cache."""
        with self._lock:
            key = self._config_key(identifier)
            self._configs.pop(key, None)
    
    def invalidate_template(self, identifier: ConfigIdentifier) -> None:
        """Remove a specific template from the cache."""
        with self._lock:
            key = self._template_key(identifier)
            self._templates.pop(key, None)
    
    def clear(self) -> None:
        """Remove all entries from the cache."""
        with self._lock:
            self._configs.clear()
            self._templates.clear()
    
    def get_cached_identifiers(self) -> List[ConfigIdentifier]:
        """Get all cached config identifiers (for background sync)."""
        identifiers = []
        with self._lock:
            for key in self._configs.keys():
                parts = key.split(":")
                if len(parts) >= 3:
                    try:
                        scope_value = int(parts[2])
                        identifiers.append(ConfigIdentifier(
                            service_name=parts[0],
                            group_id=parts[1],
                            scope=Scope(scope_value) if scope_value in [e.value for e in Scope] else Scope.SCOPE_UNSPECIFIED,
                            project_id=parts[3] if len(parts) > 3 and parts[3] else None,
                            store_id=parts[4] if len(parts) > 4 and parts[4] else None,
                            user_id=parts[5] if len(parts) > 5 and parts[5] else None,
                        ))
                    except (ValueError, IndexError):
                        continue
        return identifiers
    
    def start_background_sync(
        self,
        interval_seconds: float,
        sync_fn: Callable[[ConfigIdentifier], None]
    ) -> None:
        """Start background sync at the specified interval."""
        self.stop_background_sync()
        self._sync_stop_event.clear()
        
        def sync_loop():
            while not self._sync_stop_event.wait(interval_seconds):
                identifiers = self.get_cached_identifiers()
                for identifier in identifiers:
                    try:
                        sync_fn(identifier)
                    except Exception as e:
                        logging.warning(
                            f"Background sync failed for {identifier.service_name}/{identifier.group_id}: {e}"
                        )
        
        self._sync_thread = threading.Thread(target=sync_loop, daemon=True)
        self._sync_thread.start()
    
    def stop_background_sync(self) -> None:
        """Stop background sync."""
        self._sync_stop_event.set()
        if self._sync_thread and self._sync_thread.is_alive():
            self._sync_thread.join(timeout=5.0)
        self._sync_thread = None
