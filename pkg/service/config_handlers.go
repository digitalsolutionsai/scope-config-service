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

// getIdentifier extracts the scope and scope_id from the request identifier.
func getIdentifier(identifier *configv1.ConfigIdentifier) (scope configv1.Scope, scopeID string, err error) {
	if identifier == nil {
		return configv1.Scope_SCOPE_UNSPECIFIED, "", status.Error(codes.InvalidArgument, "identifier cannot be nil")
	}
	scope = identifier.Scope
	switch scope {
	case configv1.Scope_SYSTEM:
		scopeID = "system" // Or any other singleton value
	case configv1.Scope_PROJECT:
		scopeID = identifier.ProjectId
	case configv1.Scope_STORE:
		scopeID = identifier.StoreId
	case configv1.Scope_USER:
		scopeID = identifier.UserId
	default:
		return configv1.Scope_SCOPE_UNSPECIFIED, "", status.Errorf(codes.InvalidArgument, "unsupported scope: %s", scope)
	}
	if scopeID == "" {
		return configv1.Scope_SCOPE_UNSPECIFIED, "", status.Errorf(codes.InvalidArgument, "scope_id for %s cannot be empty", scope)
	}
	return scope, scopeID, nil
}

// setIdentifier populates the correct ID field in the identifier based on the scope.
func setIdentifier(baseIdentifier *configv1.ConfigIdentifier, scope configv1.Scope, scopeID string) {
	switch scope {
	case configv1.Scope_PROJECT:
		baseIdentifier.ProjectId = scopeID
	case configv1.Scope_STORE:
		baseIdentifier.StoreId = scopeID
	case configv1.Scope_USER:
		baseIdentifier.UserId = scopeID
	}
}

// getConfig retrieves a configuration from the database by a specific version number.
func (s *server) getConfig(ctx context.Context, req *configv1.GetConfigRequest, version int32) (*configv1.ScopeConfig, error) {
	scope, scopeID, err := getIdentifier(req.Identifier)
	if err != nil {
		return nil, err
	}

	cv := &configv1.ConfigVersion{Identifier: req.Identifier}
	var publishedVersion sql.NullInt32
	var createdAt, updatedAt sql.NullTime

	query := `SELECT id, latest_version, published_version, created_at, updated_at FROM config_version
              WHERE service_name = $1 AND scope = $2 AND scope_id = $3 AND group_id = $4`
	err = s.db.QueryRowContext(ctx, query, req.Identifier.ServiceName, scope.String(), scopeID, req.Identifier.GroupId).Scan(
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

	var versionToFetch int32
	if version > 0 {
		versionToFetch = version
	} else if version == -1 { // Fetch latest version
		versionToFetch = cv.LatestVersion
	} else { // Fetch published version by default
		if !publishedVersion.Valid {
			return &configv1.ScopeConfig{VersionInfo: cv, CurrentVersion: 0, Fields: []*configv1.ConfigField{}}, nil
		}
		versionToFetch = cv.PublishedVersion
	}

	// Build the query for config fields
	fieldQuery := `SELECT path, value FROM config_field WHERE config_version_id = $1 AND version = $2`
	args := []interface{}{cv.Id, versionToFetch}

	if req.Path != "" {
		fieldQuery += " AND path = $3"
		args = append(args, req.Path)
	}

	rows, err := s.db.QueryContext(ctx, fieldQuery, args...)
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
		CurrentVersion: versionToFetch,
		Fields:         fields,
	}, nil
}

// GetConfig retrieves the published configuration for a given identifier.
func (s *server) GetConfig(ctx context.Context, req *configv1.GetConfigRequest) (*configv1.ScopeConfig, error) {
	return s.getConfig(ctx, req, 0) // 0 indicates to fetch the published version.
}

// GetLatestConfig retrieves the latest configuration for a given identifier.
func (s *server) GetLatestConfig(ctx context.Context, req *configv1.GetConfigRequest) (*configv1.ScopeConfig, error) {
	return s.getConfig(ctx, req, -1) // -1 indicates to fetch the latest version.
}

// GetConfigByVersion retrieves a specific version of a configuration.
func (s *server) GetConfigByVersion(ctx context.Context, req *configv1.GetConfigByVersionRequest) (*configv1.ScopeConfig, error) {
	getRequest := &configv1.GetConfigRequest{Identifier: req.Identifier, Path: req.Path}
	return s.getConfig(ctx, getRequest, req.Version)
}

// UpdateConfig creates a new version of a configuration.
func (s *server) UpdateConfig(ctx context.Context, req *configv1.UpdateConfigRequest) (*configv1.ScopeConfig, error) {
	scope, scopeID, err := getIdentifier(req.Identifier)
	if err != nil {
		return nil, err
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
        WHERE service_name = $1 AND scope = $2 AND scope_id = $3 AND group_id = $4 FOR UPDATE`,
		req.Identifier.ServiceName, scope.String(), scopeID, req.Identifier.GroupId)

	if err = row.Scan(&configVersionID, &latestVersion); err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `INSERT INTO config_version (service_name, scope, scope_id, group_id, latest_version, created_by, updated_by)
            VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
			req.Identifier.ServiceName, scope.String(), scopeID, req.Identifier.GroupId, newVersion, req.User, req.User,
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
			Identifier:    req.Identifier,
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
	scope, scopeID, err := getIdentifier(req.Identifier)
	if err != nil {
		return nil, err
	}

	var cv configv1.ConfigVersion
	cv.Identifier = req.Identifier
	var createdAt, updatedAt sql.NullTime

	query := `UPDATE config_version SET published_version = $1, updated_at = NOW(), updated_by = $2
			  WHERE service_name = $3 AND scope = $4 AND scope_id = $5 AND group_id = $6
			  RETURNING id, latest_version, published_version, created_at, updated_at`

	err = s.db.QueryRowContext(ctx, query, req.VersionToPublish, req.User, req.Identifier.ServiceName, scope.String(), scopeID, req.Identifier.GroupId).Scan(
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

// GetConfigHistory retrieves the version history of a configuration.
func (s *server) GetConfigHistory(ctx context.Context, req *configv1.GetConfigHistoryRequest) (*configv1.GetConfigHistoryResponse, error) {
	scope, scopeID, err := getIdentifier(req.Identifier)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, latest_version, published_version, created_at, updated_at, updated_by FROM config_version
              WHERE service_name = $1 AND scope = $2 AND scope_id = $3 AND group_id = $4
              ORDER BY updated_at DESC`

	rows, err := s.db.QueryContext(ctx, query, req.Identifier.ServiceName, scope.String(), scopeID, req.Identifier.GroupId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query config history: %v", err)
	}
	defer rows.Close()

	var versions []*configv1.ConfigVersion
	for rows.Next() {
		cv := &configv1.ConfigVersion{Identifier: req.Identifier}
		var publishedVersion sql.NullInt32
		var createdAt, updatedAt sql.NullTime
		var updatedBy sql.NullString

		if err := rows.Scan(&cv.Id, &cv.LatestVersion, &publishedVersion, &createdAt, &updatedAt, &updatedBy); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan config version: %v", err)
		}
		if publishedVersion.Valid {
			cv.PublishedVersion = publishedVersion.Int32
		}
		if createdAt.Valid {
			cv.CreatedAt = timestamppb.New(createdAt.Time)
		}
		if updatedAt.Valid {
			cv.UpdatedAt = timestamppb.New(updatedAt.Time)
		}
		if updatedBy.Valid {
			cv.UpdatedBy = updatedBy.String
		}
		versions = append(versions, cv)
	}

	return &configv1.GetConfigHistoryResponse{Versions: versions}, nil
}

// DeleteConfig deletes a configuration.
func (s *server) DeleteConfig(ctx context.Context, req *configv1.DeleteConfigRequest) (*emptypb.Empty, error) {
	scope, scopeID, err := getIdentifier(req.Identifier)
	if err != nil {
		return nil, err
	}

	query := `DELETE FROM config_version WHERE service_name = $1 AND scope = $2 AND scope_id = $3 AND group_id = $4`
	result, err := s.db.ExecContext(ctx, query, req.Identifier.ServiceName, scope.String(), scopeID, req.Identifier.GroupId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete config: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "config not found")
	}

	return &emptypb.Empty{}, nil
}
