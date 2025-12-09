package httpgateway

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
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

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Template routes
		r.Get("/templates/{serviceName}", gateway.GetTemplate)

		// Config routes
		r.Get("/config/{serviceName}/scope/{scope}", gateway.GetConfig)
		r.Get("/config/{serviceName}/scope/{scope}/latest", gateway.GetLatestConfig)
		r.Get("/config/{serviceName}/scope/{scope}/history", gateway.GetConfigHistory)
		r.Post("/config/{serviceName}/scope/{scope}/publish", gateway.PublishConfig)
	})

	return r
}
