package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/digitalsolutionsai/scope-config-service/pkg/service"
	"github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func main() {
	log.Println("Starting ScopeConfig Service...")

	// Use the DATABASE_URL from the environment, with a fallback for local development.
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5432/config_db?sslmode=disable"
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

	// Start the gRPC server.
	startGRPCServer(db, 50051)
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

func startGRPCServer(db *sql.DB, port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}

	s := grpc.NewServer()
	// Pass the database connection to the service.
	configService := service.NewConfigService(db)
	configv1.RegisterConfigServiceServer(s, configService)

	log.Printf("gRPC server listening on port %d", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
