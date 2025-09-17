package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/digitalsolutionsai/scope-config-service/pkg/service"
	"github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"google.golang.org/grpc"
)

func main() {
	// This is a placeholder for the full implementation.
	fmt.Println("Starting ScopeConfig Service...")

	// Use the DATABASE_URL from the environment, with a fallback for local development.
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5432/config_db?sslmode=disable"
	}

	migrationsPath := "file://db/migrations"

	// Run database migrations.
	runMigrations(dbURL, migrationsPath)

	// Start the gRPC server.
	startGRPCServer(50051)
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

func startGRPCServer(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}

	s := grpc.NewServer()
	configService := service.NewConfigService()
	configv1.RegisterConfigServiceServer(s, configService)

	log.Printf("gRPC server listening on port %d", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
