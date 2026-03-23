package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/digitalsolutionsai/scope-config-service/pkg/database"
	"github.com/digitalsolutionsai/scope-config-service/pkg/httpgateway"
	"github.com/digitalsolutionsai/scope-config-service/pkg/seedloader"
	"github.com/digitalsolutionsai/scope-config-service/pkg/service"
	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.Println("Starting ScopeConfig Service...")

	// Detect database dialect and DSN from environment.
	// If DATABASE_URL is set, PostgreSQL is used. Otherwise, SQLite is the default.
	dialect, dsn := database.DetectDialect()
	log.Printf("Using database dialect: %s", dialect)

	// For PostgreSQL, run migrations before opening the connection.
	if dialect == database.DialectPostgres {
		if err := database.RunMigrations(dialect, dsn, nil); err != nil {
			log.Fatalf("Failed to run database migrations: %v", err)
		}
	}

	// Open the database connection.
	db, err := database.Open(dialect, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// For SQLite, initialize the schema after opening the connection.
	if dialect == database.DialectSQLite {
		if err := database.RunMigrations(dialect, dsn, db); err != nil {
			log.Fatalf("Failed to initialize SQLite schema: %v", err)
		}
	}

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
	go startGRPCServer(db, dialect, grpcPort)

	// Give gRPC server a moment to start
	time.Sleep(500 * time.Millisecond)

	// Start the HTTP gateway server
	startHTTPGateway(db, grpcPort, httpPort)
}

func startGRPCServer(db *sql.DB, dialect database.Dialect, port string) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	s := grpc.NewServer()
	// Pass the database connection to the service.
	configService := service.NewConfigService(db, dialect)
	configv1.RegisterConfigServiceServer(s, configService)

	// Load and apply seed templates after server is initialized
	seedDir := os.Getenv("SEED_TEMPLATES_DIR")
	if seedDir == "" {
		seedDir = "templates"
	}
	loader := seedloader.NewLoader(seedDir, configService)
	if err := loader.LoadAndApplyAll(context.Background()); err != nil {
		log.Printf("Warning: Failed to load seed templates: %v", err)
	}

	log.Printf("gRPC server listening on port %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func startHTTPGateway(db *sql.DB, grpcPort, httpPort string) {
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

	// Create HTTP router without authentication
	// Authentication is handled at the API Gateway level (e.g., Spring Gateway)
	log.Println("HTTP service is public - authentication handled at gateway level")
	router := httpgateway.NewRouterWithConfig(httpgateway.RouterConfig{
		Client: client,
		DB:     db,
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
	log.Println("  GET  /api/v1/config/{serviceName}/template?groupId={groupId}")
	log.Println("  GET  /api/v1/config/{serviceName}/scope/{scope}?groupId={groupId}&...")
	log.Println("  GET  /api/v1/config/{serviceName}/scope/{scope}/latest?groupId={groupId}&...")
	log.Println("  GET  /api/v1/config/{serviceName}/scope/{scope}/history?groupId={groupId}&...")
	log.Println("  POST /api/v1/config/{serviceName}/scope/{scope}/publish?groupId={groupId}")
	
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
