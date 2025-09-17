package service

import (
	"context"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// configs is a simple in-memory store for demonstration purposes.
var configs = make(map[string]*configv1.ScopeConfig)

// server is used to implement configv1.ConfigServiceServer.
type server struct {
	configv1.UnimplementedConfigServiceServer
}

// NewConfigService creates a new gRPC server for the Config service.
func NewConfigService() configv1.ConfigServiceServer {
	return &server{}
}

// GetConfig retrieves a configuration.
func (s *server) GetConfig(ctx context.Context, req *configv1.GetConfigRequest) (*configv1.ScopeConfig, error) {
	config, ok := configs[req.Identifier.String()]
	if !ok {
		// Return an empty config if not found. This allows clients to handle
		// non-existent configs gracefully without needing to check for a NotFound error.
		return &configv1.ScopeConfig{}, nil
	}
	return config, nil
}

func (s *server) GetConfigByVersion(ctx context.Context, req *configv1.GetConfigByVersionRequest) (*configv1.ScopeConfig, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConfigByVersion not implemented")
}

func (s *server) GetConfigHistory(ctx context.Context, req *configv1.GetConfigHistoryRequest) (*configv1.GetConfigHistoryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConfigHistory not implemented")
}

// UpdateConfig updates or creates a new configuration set.
func (s *server) UpdateConfig(ctx context.Context, req *configv1.UpdateConfigRequest) (*configv1.ScopeConfig, error) {
	key := req.Identifier.String()
	config, ok := configs[key]
	if !ok {
		config = &configv1.ScopeConfig{
			Fields: []*configv1.ConfigField{},
		}
	}

	// This is a simple in-memory implementation. A real implementation would handle updates more robustly.
	config.Fields = req.Fields
	configs[key] = config

	return config, nil
}

func (s *server) PublishVersion(ctx context.Context, req *configv1.PublishVersionRequest) (*configv1.ConfigVersion, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PublishVersion not implemented")
}

// DeleteConfig deletes a configuration set.
func (s *server) DeleteConfig(ctx context.Context, req *configv1.DeleteConfigRequest) (*emptypb.Empty, error) {
	key := req.Identifier.String()
	delete(configs, key)
	return &emptypb.Empty{}, nil
}
