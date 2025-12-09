package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/digitalsolutionsai/scope-config-service/pkg/httpgateway"
	"github.com/digitalsolutionsai/scope-config-service/pkg/service"
	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.Println("Starting ScopeConfig Service...")

	// Use the DATABASE_URL from the environment, with a fallback for local development.
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5555/config_db?sslmode=disable"
	}

	// Run database migrations before connecting.
	migrationsPath := "file://db/migrations"
	runMigrations(dbURL, migrationsPath)

	// Connect to the database.
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to the database.")

	// Get gRPC port from environment or use default
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	// Get HTTP port from environment or use default
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	// Start the gRPC server in a goroutine
	go startGRPCServer(db, grpcPort)

	// Give gRPC server a moment to start
	time.Sleep(500 * time.Millisecond)

	// Start the HTTP gateway server
	startHTTPGateway(grpcPort, httpPort)
}

func runMigrations(databaseURL string, migrationsPath string) {
	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	log.Println("Database migrations applied successfully.")
}

func startGRPCServer(db *sql.DB, port string) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	s := grpc.NewServer()
	// Pass the database connection to the service.
	configService := service.NewConfigService(db)
	configv1.RegisterConfigServiceServer(s, configService)

	log.Printf("gRPC server listening on port %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func startHTTPGateway(grpcPort, httpPort string) {
	log.Println("Starting HTTP Gateway...")

	// Connect to local gRPC server
	grpcAddr := fmt.Sprintf("localhost:%s", grpcPort)
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
