package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/digitalsolutionsai/scope-config-service/pkg/httpgateway"
	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.Println("Starting HTTP Gateway for ScopeConfig Service...")

	// Get gRPC server address from environment or use default
	grpcAddr := os.Getenv("GRPC_SERVER_ADDRESS")
	if grpcAddr == "" {
		grpcAddr = "localhost:50051"
	}

	// Get HTTP server port from environment or use default
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	// Get Keycloak configuration
	keycloakPublicKey := os.Getenv("KEYCLOAK_PUBLIC_KEY")
	keycloakClient := os.Getenv("KEYCLOAK_CLIENT")
	if keycloakClient == "" {
		keycloakClient = "dsai-console"
	}
	keycloakRoles := os.Getenv("KEYCLOAK_ROLES")
	if keycloakRoles == "" {
		keycloakRoles = "ROLE_ADMIN"
	}

	// Connect to gRPC server
	log.Printf("Connecting to gRPC server at %s...", grpcAddr)
	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	// Create gRPC client
	client := configv1.NewConfigServiceClient(conn)

	// Setup authentication middleware
	var authMiddleware *httpgateway.AuthMiddleware
	if keycloakPublicKey != "" {
		log.Println("Keycloak authentication enabled")
		log.Printf("  Client: %s", keycloakClient)
		log.Printf("  Required roles: %s", keycloakRoles)
		
		roles := strings.Split(keycloakRoles, ",")
		authMiddleware, err = httpgateway.NewAuthMiddleware(keycloakPublicKey, keycloakClient, roles)
		if err != nil {
			log.Fatalf("Failed to create auth middleware: %v", err)
		}
	} else {
		log.Println("⚠️  WARNING: Running without authentication (KEYCLOAK_PUBLIC_KEY not set)")
		log.Println("⚠️  This should only be used for development/testing!")
		authMiddleware, _ = httpgateway.NewAuthMiddleware("", "", nil)
	}

	// Create HTTP router with authentication
	router := httpgateway.NewRouterWithConfig(httpgateway.RouterConfig{
		Client:         client,
		AuthMiddleware: authMiddleware,
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", httpPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server
	log.Printf("HTTP Gateway listening on port %s", httpPort)
	log.Println("Available endpoints:")
	log.Println("  GET  /api/v1/templates/{serviceName}?groupId={groupId}")
	log.Println("  GET  /api/v1/config/{serviceName}/scope/{scope}?groupId={groupId}&...")
	log.Println("  GET  /api/v1/config/{serviceName}/scope/{scope}/latest?groupId={groupId}&...")
	log.Println("  GET  /api/v1/config/{serviceName}/scope/{scope}/history?groupId={groupId}&...")
	log.Println("  POST /api/v1/config/{serviceName}/scope/{scope}/publish?groupId={groupId}")
	
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
