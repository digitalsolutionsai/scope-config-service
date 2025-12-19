#!/usr/bin/env python3
"""
FastAPI Integration Example for ScopeConfig Python SDK.

This example demonstrates how to integrate the ScopeConfig SDK
with a FastAPI application using dependency injection.

Integration steps:
1. Copy the SDK to your project
2. Install dependencies
3. Create the ScopeConfig service
4. Use dependency injection in routes

Prerequisites:
- FastAPI application
- Environment variables (optional):
  - GRPC_SCOPE_CONFIG_HOST (default: localhost)
  - GRPC_SCOPE_CONFIG_PORT (default: 50051)
  - GRPC_SCOPE_CONFIG_USE_TLS (default: false)
"""

# =============================================================================
# Step 1: Copy SDK to your project
# =============================================================================
# Copy the sdks/python folder to your project:
#   cp -r sdks/python/scopeconfig your-fastapi-app/scopeconfig

# =============================================================================
# Step 2: Install dependencies
# =============================================================================
# pip install fastapi uvicorn grpcio grpcio-tools

# =============================================================================
# Step 3: Create ScopeConfig Service (scopeconfig_service.py)
# =============================================================================

import os
import sys
from typing import Optional
from contextlib import asynccontextmanager

# Add parent directory to path for local development
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scopeconfig import (
    ConfigClient,
    ConfigIdentifier,
    ScopeConfig,
    Scope,
    GetValueOptions,
    create_identifier,
)


class ScopeConfigService:
    """Service wrapper for ScopeConfig SDK with FastAPI integration."""
    
    def __init__(self):
        self._client: Optional[ConfigClient] = None
    
    def connect(self) -> None:
        """Connect to the ScopeConfig service."""
        self._client = ConfigClient(
            cache_enabled=True,
            cache_ttl_seconds=60.0,
            background_sync_enabled=True,
            background_sync_interval_seconds=30.0,
        )
        self._client.connect()
    
    def close(self) -> None:
        """Close the connection to the ScopeConfig service."""
        if self._client:
            self._client.close()
            self._client = None
    
    @property
    def client(self) -> ConfigClient:
        """Get the underlying client."""
        if self._client is None:
            raise RuntimeError("ScopeConfig service not connected")
        return self._client
    
    def get_config(self, identifier: ConfigIdentifier) -> ScopeConfig:
        """Get configuration with caching."""
        return self.client.get_config_cached(identifier)
    
    def get_value(
        self,
        identifier: ConfigIdentifier,
        path: str,
        use_default: bool = True,
        inherit: bool = True,
    ) -> Optional[str]:
        """Get specific value with options."""
        return self.client.get_value(
            identifier,
            path,
            GetValueOptions(use_default=use_default, inherit=inherit)
        )
    
    def get_project_value(
        self,
        service_name: str,
        group_id: str,
        project_id: str,
        path: str,
        use_default: bool = True,
        inherit: bool = True,
    ) -> Optional[str]:
        """Get project-level config value."""
        identifier = (
            create_identifier(service_name)
            .with_scope(Scope.PROJECT)
            .with_group_id(group_id)
            .with_project_id(project_id)
            .build()
        )
        return self.get_value(identifier, path, use_default, inherit)
    
    def get_store_value(
        self,
        service_name: str,
        group_id: str,
        project_id: str,
        store_id: str,
        path: str,
        use_default: bool = True,
        inherit: bool = True,
    ) -> Optional[str]:
        """Get store-level config value with inheritance."""
        identifier = (
            create_identifier(service_name)
            .with_scope(Scope.STORE)
            .with_group_id(group_id)
            .with_project_id(project_id)
            .with_store_id(store_id)
            .build()
        )
        return self.get_value(identifier, path, use_default, inherit)
    
    def get_user_value(
        self,
        service_name: str,
        group_id: str,
        user_id: str,
        path: str,
        use_default: bool = True,
        inherit: bool = True,
    ) -> Optional[str]:
        """Get user-level config value with inheritance."""
        identifier = (
            create_identifier(service_name)
            .with_scope(Scope.USER)
            .with_group_id(group_id)
            .with_user_id(user_id)
            .build()
        )
        return self.get_value(identifier, path, use_default, inherit)
    
    def invalidate_cache(self, identifier: ConfigIdentifier) -> None:
        """Invalidate cache for specific config."""
        self.client.invalidate_cache(identifier)
    
    def clear_cache(self) -> None:
        """Clear all cached configs."""
        self.client.clear_cache()


# Global service instance
scope_config_service = ScopeConfigService()


# =============================================================================
# Step 4: FastAPI Application with Lifespan
# =============================================================================

# Note: This is a complete FastAPI application example
# In a real project, split this into separate files

try:
    from fastapi import FastAPI, Depends, HTTPException
    from pydantic import BaseModel
    
    # Lifespan context manager for startup/shutdown
    @asynccontextmanager
    async def lifespan(app: FastAPI):
        """Manage ScopeConfig service lifecycle."""
        # Startup
        try:
            scope_config_service.connect()
            print("ScopeConfig service connected")
        except Exception as e:
            print(f"Warning: Could not connect to ScopeConfig service: {e}")
        
        yield
        
        # Shutdown
        scope_config_service.close()
        print("ScopeConfig service disconnected")
    
    # Create FastAPI app
    app = FastAPI(
        title="ScopeConfig FastAPI Example",
        description="Example FastAPI application using ScopeConfig SDK",
        lifespan=lifespan,
    )
    
    # Dependency injection
    def get_scope_config() -> ScopeConfigService:
        """Dependency to get ScopeConfig service."""
        return scope_config_service
    
    # Pydantic models
    class ConfigValue(BaseModel):
        path: str
        value: Optional[str]
    
    class ConfigRequest(BaseModel):
        service_name: str
        group_id: str
        project_id: Optional[str] = None
        store_id: Optional[str] = None
        user_id: Optional[str] = None
        path: str
        use_default: bool = True
        inherit: bool = True
    
    # Routes
    @app.get("/")
    async def root():
        """Root endpoint."""
        return {"message": "ScopeConfig FastAPI Example", "status": "running"}
    
    @app.post("/config/value", response_model=ConfigValue)
    async def get_config_value(
        request: ConfigRequest,
        config: ScopeConfigService = Depends(get_scope_config),
    ):
        """Get a configuration value."""
        try:
            # Determine scope and build identifier
            if request.store_id:
                value = config.get_store_value(
                    request.service_name,
                    request.group_id,
                    request.project_id or "",
                    request.store_id,
                    request.path,
                    request.use_default,
                    request.inherit,
                )
            elif request.user_id:
                value = config.get_user_value(
                    request.service_name,
                    request.group_id,
                    request.user_id,
                    request.path,
                    request.use_default,
                    request.inherit,
                )
            elif request.project_id:
                value = config.get_project_value(
                    request.service_name,
                    request.group_id,
                    request.project_id,
                    request.path,
                    request.use_default,
                    request.inherit,
                )
            else:
                # SYSTEM scope
                identifier = (
                    create_identifier(request.service_name)
                    .with_scope(Scope.SYSTEM)
                    .with_group_id(request.group_id)
                    .build()
                )
                value = config.get_value(
                    identifier,
                    request.path,
                    request.use_default,
                    request.inherit,
                )
            
            return ConfigValue(path=request.path, value=value)
        
        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))
    
    @app.post("/config/cache/invalidate")
    async def invalidate_cache(
        service_name: str,
        group_id: str,
        project_id: Optional[str] = None,
        config: ScopeConfigService = Depends(get_scope_config),
    ):
        """Invalidate cache for a specific configuration."""
        try:
            builder = create_identifier(service_name).with_group_id(group_id)
            if project_id:
                builder = builder.with_scope(Scope.PROJECT).with_project_id(project_id)
            else:
                builder = builder.with_scope(Scope.SYSTEM)
            
            identifier = builder.build()
            config.invalidate_cache(identifier)
            return {"message": "Cache invalidated"}
        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))
    
    @app.post("/config/cache/clear")
    async def clear_cache(
        config: ScopeConfigService = Depends(get_scope_config),
    ):
        """Clear all cached configurations."""
        try:
            config.clear_cache()
            return {"message": "Cache cleared"}
        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))
    
    # Example usage in a business service
    @app.get("/payment/gateway-url/{project_id}/{store_id}")
    async def get_payment_gateway_url(
        project_id: str,
        store_id: str,
        config: ScopeConfigService = Depends(get_scope_config),
    ):
        """Example: Get payment gateway URL for a store."""
        try:
            url = config.get_store_value(
                "payment-service",
                "gateway",
                project_id,
                store_id,
                "gateway.url",
                use_default=True,
                inherit=True,
            )
            return {
                "project_id": project_id,
                "store_id": store_id,
                "gateway_url": url or "https://default-gateway.example.com",
            }
        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))
    
    @app.get("/features/{project_id}/{feature_name}")
    async def is_feature_enabled(
        project_id: str,
        feature_name: str,
        config: ScopeConfigService = Depends(get_scope_config),
    ):
        """Example: Check if a feature is enabled for a project."""
        try:
            value = config.get_project_value(
                "payment-service",
                "features",
                project_id,
                f"feature.{feature_name}.enabled",
                use_default=True,
                inherit=True,
            )
            return {
                "project_id": project_id,
                "feature": feature_name,
                "enabled": value == "true" if value else False,
            }
        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))

except ImportError:
    print("FastAPI not installed. Install with: pip install fastapi uvicorn")
    print("This example demonstrates FastAPI integration patterns.")
    app = None


def main():
    """Run the example FastAPI application."""
    if app is None:
        print("\nTo run this example:")
        print("1. Install FastAPI: pip install fastapi uvicorn")
        print("2. Run: uvicorn examples.fastapi_integration:app --reload")
        return
    
    try:
        import uvicorn
        print("\nStarting FastAPI server...")
        print("API docs available at: http://localhost:8000/docs")
        uvicorn.run(app, host="0.0.0.0", port=8000)
    except ImportError:
        print("\nTo run this example:")
        print("1. Install uvicorn: pip install uvicorn")
        print("2. Run: uvicorn examples.fastapi_integration:app --reload")


if __name__ == "__main__":
    main()
