package httpgateway

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	httpSwagger "github.com/swaggo/http-swagger"
	
	_ "github.com/digitalsolutionsai/scope-config-service/docs" // Import generated swagger docs
)

// RouterConfig holds configuration for the HTTP router.
type RouterConfig struct {
	Client         configv1.ConfigServiceClient
	AuthMiddleware *AuthMiddleware
}

// NewRouter creates a new HTTP router with all the gateway endpoints.
func NewRouter(client configv1.ConfigServiceClient) *chi.Mux {
	return NewRouterWithConfig(RouterConfig{
		Client:         client,
		AuthMiddleware: nil, // No auth by default
	})
}

// NewRouterWithConfig creates a new HTTP router with custom configuration.
func NewRouterWithConfig(config RouterConfig) *chi.Mux {
	gateway := NewGateway(config.Client)
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// Add authentication middleware if provided
	if config.AuthMiddleware != nil {
		r.Use(config.AuthMiddleware.Middleware)
	}

	// Swagger UI endpoint - accessible without authentication
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Global templates list endpoint
	r.Get("/api/v1/config/templates", gateway.ListTemplates)

	// API routes - all under /api/v1/config/{serviceName} for consistent path-based routing
	// Each service has one template, so template is nested under service path
	r.Route("/api/v1/config/{serviceName}", func(r chi.Router) {
		// Template route - get template for this service
		r.Get("/template", gateway.GetTemplate)

		// Config routes - manage configurations for this service
		r.Get("/scope/{scope}", gateway.GetConfig)
		r.Put("/scope/{scope}", gateway.UpdateConfig) // Update config values
		r.Get("/scope/{scope}/latest", gateway.GetLatestConfig)
		r.Get("/scope/{scope}/version/{version}", gateway.GetConfigByVersion) // Get config by specific version
		r.Get("/scope/{scope}/history", gateway.GetConfigHistory)
		r.Post("/scope/{scope}/publish", gateway.PublishConfig)
	})

	return r
}
