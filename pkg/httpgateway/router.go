package httpgateway

import (
	"database/sql"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/digitalsolutionsai/scope-config-service/docs" // Import generated swagger docs
)

// RouterConfig holds configuration for the HTTP router.
type RouterConfig struct {
	Client              configv1.ConfigServiceClient
	AuthMiddleware      *AuthMiddleware      // existing Keycloak JWT (unchanged)
	BasicAuthMiddleware *BasicAuthMiddleware // Basic Auth
	DB                  *sql.DB              // for admin operations that need direct DB access
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
	adminGW := NewAdminGateway(config.Client, config.DB)
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// Swagger UI endpoint - accessible without authentication
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Protected route group
	r.Group(func(r chi.Router) {
		// Apply Basic Auth if configured
		if config.BasicAuthMiddleware != nil {
			r.Use(config.BasicAuthMiddleware.Handler)
		}

		// Apply existing JWT auth if configured
		if config.AuthMiddleware != nil {
			r.Use(config.AuthMiddleware.Middleware)
		}

		// Built-in Admin UI
		r.Get("/admin", ServeAdminUI)
		r.Get("/admin/", ServeAdminUI)

		// Global templates list — supports ?includeInactive=true for admin UI
		r.Get("/api/v1/config/templates", adminGW.ListAllTemplates)
		// Import / upsert one or more template groups
		r.Post("/api/v1/config/templates", adminGW.ImportTemplate)
		// Toggle is_active on a template
		r.Patch("/api/v1/config/templates/{serviceName}/{groupId}/active", adminGW.SetTemplateActive)

		// API routes - all under /api/v1/config/{serviceName} for consistent path-based routing
		r.Route("/api/v1/config/{serviceName}", func(r chi.Router) {
			r.Get("/template", gateway.GetTemplate)
			r.Get("/scope/{scope}", gateway.GetConfig)
			r.Put("/scope/{scope}", gateway.UpdateConfig)
			r.Get("/scope/{scope}/latest", gateway.GetLatestConfig)
			r.Get("/scope/{scope}/version/{version}", gateway.GetConfigByVersion)
			r.Get("/scope/{scope}/history", gateway.GetConfigHistory)
			r.Post("/scope/{scope}/publish", gateway.PublishConfig)
		})
	})

	return r
}
