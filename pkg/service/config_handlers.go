package service

import (
	"context"
	"database/sql"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetConfig retrieves the published configuration for a given identifier.
func (s *server) GetConfig(ctx context.Context, req *configv1.GetConfigRequest) (*configv1.ScopeConfig, error) {
	identifier := req.Identifier
	if identifier == nil {
		return nil, status.Error(codes.InvalidArgument, "identifier cannot be nil")
	}

	cv := &configv1.ConfigVersion{Identifier: identifier}
	var publishedVersion sql.NullInt32
	var createdAt, updatedAt sql.NullTime

	query := `SELECT id, latest_version, published_version, created_at, updated_at FROM config_version
              WHERE service_name = $1 AND COALESCE(project_id, '') = $2 AND COALESCE(store_id, '') = $3 AND COALESCE(group_id, '') = $4 AND scope = $5`
	err := s.db.QueryRowContext(ctx, query, identifier.ServiceName, identifier.ProjectId, identifier.StoreId, identifier.GroupId, identifier.Scope.String()).Scan(
		&cv.Id, &cv.LatestVersion, &publishedVersion, &createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return &configv1.ScopeConfig{VersionInfo: cv, Fields: []*configv1.ConfigField{}}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query config version: %v", err)
	}

	if createdAt.Valid {
		cv.CreatedAt = timestamppb.New(createdAt.Time)
	}
	if updatedAt.Valid {
		cv.UpdatedAt = timestamppb.New(updatedAt.Time)
	}

	if publishedVersion.Valid {
		cv.PublishedVersion = publishedVersion.Int32
	} else {
		// No version is published, return the version info but no fields.
		return &configv1.ScopeConfig{VersionInfo: cv, CurrentVersion: 0, Fields: []*configv1.ConfigField{}}, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT path, value FROM config_field WHERE config_version_id = $1 AND version = $2`,
		cv.Id, cv.PublishedVersion)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query config fields: %v", err)
	}
	defer rows.Close()

	var fields []*configv1.ConfigField
	for rows.Next() {
		var path, value string
		if err := rows.Scan(&path, &value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan config field: %v", err)
		}
		fields = append(fields, &configv1.ConfigField{Path: path, Value: value})
	}

	return &configv1.ScopeConfig{
		VersionInfo:    cv,
		CurrentVersion: cv.PublishedVersion,
		Fields:         fields,
	}, nil
}

// GetLatestConfig retrieves the latest configuration for a given identifier.
func (s *server) GetLatestConfig(ctx context.Context, req *configv1.GetConfigRequest) (*configv1.ScopeConfig, error) {
	identifier := req.Identifier
	if identifier == nil {
		return nil, status.Error(codes.InvalidArgument, "identifier cannot be nil")
	}

	cv := &configv1.ConfigVersion{Identifier: identifier}
	var publishedVersion sql.NullInt32
	var createdAt, updatedAt sql.NullTime

	query := `SELECT id, latest_version, published_version, created_at, updated_at FROM config_version
              WHERE service_name = $1 AND COALESCE(project_id, '') = $2 AND COALESCE(store_id, '') = $3 AND COALESCE(group_id, '') = $4 AND scope = $5`
	err := s.db.QueryRowContext(ctx, query, identifier.ServiceName, identifier.ProjectId, identifier.StoreId, identifier.GroupId, identifier.Scope.String()).Scan(
		&cv.Id, &cv.LatestVersion, &publishedVersion, &createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return &configv1.ScopeConfig{VersionInfo: cv, Fields: []*configv1.ConfigField{}}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query config version: %v", err)
	}

	if createdAt.Valid {
		cv.CreatedAt = timestamppb.New(createdAt.Time)
	}
	if updatedAt.Valid {
		cv.UpdatedAt = timestamppb.New(updatedAt.Time)
	}

	if publishedVersion.Valid {
		cv.PublishedVersion = publishedVersion.Int32
	}

	rows, err := s.db.QueryContext(ctx, `SELECT path, value FROM config_field WHERE config_version_id = $1 AND version = $2`,
		cv.Id, cv.LatestVersion)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query config fields: %v", err)
	}
	defer rows.Close()

	var fields []*configv1.ConfigField
	for rows.Next() {
		var path, value string
		if err := rows.Scan(&path, &value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan config field: %v", err)
		}
		fields = append(fields, &configv1.ConfigField{Path: path, Value: value})
	}

	return &configv1.ScopeConfig{
		VersionInfo:    cv,
		CurrentVersion: cv.LatestVersion,
		Fields:         fields,
	}, nil
}

// UpdateConfig creates a new version of a configuration.
func (s *server) UpdateConfig(ctx context.Context, req *configv1.UpdateConfigRequest) (*configv1.ScopeConfig, error) {
	identifier := req.Identifier
	if identifier == nil {
		return nil, status.Error(codes.InvalidArgument, "identifier cannot be nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	var configVersionID int32
	var latestVersion int32
	newVersion := int32(1)

	row := tx.QueryRowContext(ctx, `SELECT id, latest_version FROM config_version
        WHERE service_name = $1 AND COALESCE(project_id, '') = $2 AND COALESCE(store_id, '') = $3 AND COALESCE(group_id, '') = $4 AND scope = $5 FOR UPDATE`,
		identifier.ServiceName, identifier.ProjectId, identifier.StoreId, identifier.GroupId, identifier.Scope.String())

	if err = row.Scan(&configVersionID, &latestVersion); err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `INSERT INTO config_version (service_name, project_id, store_id, group_id, scope, latest_version, created_by, updated_by)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
			identifier.ServiceName, identifier.ProjectId, identifier.StoreId, identifier.GroupId, identifier.Scope.String(), newVersion, req.User, req.User,
		).Scan(&configVersionID)

	} else if err == nil {
		newVersion = latestVersion + 1
		_, err = tx.ExecContext(ctx, `UPDATE config_version SET latest_version = $1, updated_at = NOW(), updated_by = $2 WHERE id = $3`,
			newVersion, req.User, configVersionID)
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upsert config version: %v", err)
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO config_field (config_version_id, version, path, value, type) VALUES ($1, $2, $3, $4, 'STRING')`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to prepare field insert statement: %v", err)
	}
	defer stmt.Close()

	for _, field := range req.Fields {
		if _, err := stmt.ExecContext(ctx, configVersionID, newVersion, field.Path, field.Value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to insert config field: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return &configv1.ScopeConfig{
		VersionInfo: &configv1.ConfigVersion{
			Id:            configVersionID,
			Identifier:    identifier,
			LatestVersion: newVersion,
			CreatedAt:     timestamppb.Now(),
			UpdatedAt:     timestamppb.Now(),
		},
		CurrentVersion: newVersion,
		Fields:         req.Fields,
	}, nil
}

// PublishVersion marks a specific version as "published".
func (s *server) PublishVersion(ctx context.Context, req *configv1.PublishVersionRequest) (*configv1.ConfigVersion, error) {
	identifier := req.Identifier
	if identifier == nil {
		return nil, status.Error(codes.InvalidArgument, "identifier cannot be nil")
	}

	var cv configv1.ConfigVersion
	cv.Identifier = identifier
	var createdAt, updatedAt sql.NullTime

	query := `UPDATE config_version SET published_version = $1, updated_at = NOW(), updated_by = $2
			  WHERE service_name = $3 AND COALESCE(project_id, '') = $4 AND COALESCE(store_id, '') = $5 AND COALESCE(group_id, '') = $6 AND scope = $7
			  RETURNING id, latest_version, published_version, created_at, updated_at`

	err := s.db.QueryRowContext(ctx, query, req.VersionToPublish, req.User, identifier.ServiceName, identifier.ProjectId, identifier.StoreId, identifier.GroupId, identifier.Scope.String()).Scan(
		&cv.Id, &cv.LatestVersion, &cv.PublishedVersion, &createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "config identifier not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish version: %v", err)
	}

	if createdAt.Valid {
		cv.CreatedAt = timestamppb.New(createdAt.Time)
	}
	if updatedAt.Valid {
		cv.UpdatedAt = timestamppb.New(updatedAt.Time)
	}

	return &cv, nil
}

func (s *server) GetConfigByVersion(ctx context.Context, req *configv1.GetConfigByVersionRequest) (*configv1.ScopeConfig, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConfigByVersion not implemented")
}

func (s *server) GetConfigHistory(ctx context.Context, req *configv1.GetConfigHistoryRequest) (*configv1.GetConfigHistoryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConfigHistory not implemented")
}

func (s *server) DeleteConfig(ctx context.Context, req *configv1.DeleteConfigRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteConfig not implemented")
}
