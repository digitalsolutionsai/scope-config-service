#!/usr/bin/env python3
"""
Example usage of the ScopeConfig Python SDK.

This example demonstrates:
- Creating a client using environment variables
- Building config identifiers
- Getting config values with caching
- Using inheritance and default values
- Applying configuration templates

Prerequisites:
1. Install dependencies: pip install -r requirements.txt
2. Generate proto files: buf generate (optional)
3. Set environment variables (optional):
   - GRPC_SCOPE_CONFIG_HOST (default: localhost)
   - GRPC_SCOPE_CONFIG_PORT (default: 50051)
   - GRPC_SCOPE_CONFIG_USE_TLS (default: false)

Run:
    python examples/basic_usage.py
"""

import os
import sys

# Add parent directory to path for local development
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scopeconfig import (
    ConfigClient,
    Scope,
    GetValueOptions,
    create_identifier,
    ConfigIdentifier,
)


def demonstrate_identifier_building():
    """Demonstrate how to build config identifiers for different scopes."""
    print("\n=== Building Config Identifiers ===")
    
    # SYSTEM scope (global config)
    system_id = (
        create_identifier("my-service")
        .with_scope(Scope.SYSTEM)
        .with_group_id("database")
        .build()
    )
    print(f"System identifier: service={system_id.service_name}, group={system_id.group_id}, scope={system_id.scope}")
    
    # PROJECT scope
    project_id = (
        create_identifier("my-service")
        .with_scope(Scope.PROJECT)
        .with_group_id("database")
        .with_project_id("proj-123")
        .build()
    )
    print(f"Project identifier: service={project_id.service_name}, group={project_id.group_id}, project={project_id.project_id}")
    
    # STORE scope
    store_id = (
        create_identifier("my-service")
        .with_scope(Scope.STORE)
        .with_group_id("database")
        .with_project_id("proj-123")
        .with_store_id("store-456")
        .build()
    )
    print(f"Store identifier: service={store_id.service_name}, group={store_id.group_id}, project={store_id.project_id}, store={store_id.store_id}")
    
    # USER scope
    user_id = (
        create_identifier("my-service")
        .with_scope(Scope.USER)
        .with_group_id("preferences")
        .with_user_id("user-789")
        .build()
    )
    print(f"User identifier: service={user_id.service_name}, group={user_id.group_id}, user={user_id.user_id}")


def main():
    """Main example demonstrating SDK usage."""
    print("=== ScopeConfig Python SDK Example ===")
    
    # Example 1: Show environment variable configuration
    print("\n=== Example 1: Environment Variables ===")
    print(f"GRPC_SCOPE_CONFIG_HOST: {os.environ.get('GRPC_SCOPE_CONFIG_HOST', 'localhost (default)')}")
    print(f"GRPC_SCOPE_CONFIG_PORT: {os.environ.get('GRPC_SCOPE_CONFIG_PORT', '50051 (default)')}")
    print(f"GRPC_SCOPE_CONFIG_USE_TLS: {os.environ.get('GRPC_SCOPE_CONFIG_USE_TLS', 'false (default)')}")
    
    # Example 2: Create client with context manager (recommended)
    print("\n=== Example 2: Using Context Manager ===")
    try:
        with ConfigClient(
            host="localhost",
            port=50051,
            use_tls=False,
            cache_enabled=True,
            cache_ttl_seconds=60.0,
            background_sync_enabled=True,
            background_sync_interval_seconds=30.0,
        ) as client:
            print("Client connected successfully")
            
            # Build identifier
            identifier = (
                create_identifier("payment-service")
                .with_scope(Scope.PROJECT)
                .with_group_id("database")
                .with_project_id("proj-123")
                .build()
            )
            
            # Example 3: Get configuration with caching
            print("\n=== Example 3: Get Configuration with Caching ===")
            try:
                config = client.get_config_cached(identifier)
                print(f"Configuration for {config.version_info.identifier.service_name if config.version_info else 'unknown'}:")
                for field in config.fields:
                    print(f"  {field.path} = {field.value}")
            except Exception as e:
                print(f"Failed to get config: {e}")
            
            # Example 4: Get specific value with inheritance
            print("\n=== Example 4: Get Value with Inheritance ===")
            try:
                value = client.get_value(
                    identifier,
                    "database.host",
                    GetValueOptions(use_default=True, inherit=True)
                )
                if value is not None:
                    print(f"Database host: {value}")
                else:
                    print("Database host not found")
            except Exception as e:
                print(f"Failed to get value: {e}")
            
            # Example 5: Get value as string (convenience method)
            print("\n=== Example 5: Get Value as String ===")
            try:
                host = client.get_value_string(
                    identifier,
                    "database.host",
                    GetValueOptions(use_default=True)
                )
                print(f"Database host (string): '{host}'")
            except Exception as e:
                print(f"Failed to get value string: {e}")
            
            # Example 6: Cache management
            print("\n=== Example 6: Cache Management ===")
            print(f"Cache enabled: {client.is_cache_enabled()}")
            
            # Invalidate specific config cache
            client.invalidate_cache(identifier)
            print("Cache invalidated for specific identifier")
            
            # Clear all cache
            client.clear_cache()
            print("All cache cleared")
            
    except Exception as e:
        print(f"Failed to connect: {e}")
        print("\nNote: To run this example with a live server, start the ScopeConfig service first.")
    
    # Always demonstrate identifier building
    demonstrate_identifier_building()
    
    # Example 7: Manual client management
    print("\n=== Example 7: Manual Client Management ===")
    client = ConfigClient()
    try:
        client.connect()
        print("Client connected (manual mode)")
        
        # Use the client...
        identifier = (
            create_identifier("my-service")
            .with_scope(Scope.SYSTEM)
            .with_group_id("logging")
            .build()
        )
        
        try:
            value = client.get_value(
                identifier,
                "log.level",
                GetValueOptions(use_default=True)
            )
            print(f"Log level: {value}")
        except Exception as e:
            print(f"Failed to get log level: {e}")
            
    except Exception as e:
        print(f"Failed to connect (manual mode): {e}")
    finally:
        client.close()
        print("Client closed (manual mode)")
    
    print("\n=== Example Complete ===")


if __name__ == "__main__":
    main()
