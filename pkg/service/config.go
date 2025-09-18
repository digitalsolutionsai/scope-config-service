package service

import (
	"database/sql"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
)

// server is used to implement configv1.ConfigServiceServer.
type server struct {
	configv1.UnimplementedConfigServiceServer
	db *sql.DB
}

// NewConfigService creates a new gRPC server for the Config service.
func NewConfigService(db *sql.DB) configv1.ConfigServiceServer {
	return &server{db: db}
}
