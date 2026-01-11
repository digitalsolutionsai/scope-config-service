package service

import (
	"context"
	"database/sql"
	"time"

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
		// No config exists
		// Check if the user explicitly disabled template defaults
		if req.UseTemplateDefaults != nil && !*req.UseTemplateDefaults {
			// User explicitly set use_template_defaults=false, return empty config
			return &configv1.ScopeConfig{VersionInfo: cv, Fields: []*configv1.ConfigField{}}, nil
		}
		// Otherwise, try to get template default values (default behavior)
		templateFields, err := s.getTemplateDefaultFields(ctx, req.Identifier.ServiceName, req.Identifier.GroupId, req.Path, scope)
		if err != nil {
			return nil, err
		}
		return &configv1.ScopeConfig{VersionInfo: cv, Fields: templateFields}, nil
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
	var shouldUseTemplateFallback bool
	if version > 0 {
		versionToFetch = version
	} else if version == -1 { // Fetch latest version
		versionToFetch = cv.LatestVersion
	} else { // Fetch published version by default
		if !publishedVersion.Valid {
			// No published version exists, we'll use template fallback if available
			shouldUseTemplateFallback = true
		} else {
			versionToFetch = cv.PublishedVersion
		}
	}

	var fields []*configv1.ConfigField

	// If we should use template fallback, try to get template default values
	if shouldUseTemplateFallback {
		// Check if the user explicitly disabled template defaults
		if req.UseTemplateDefaults != nil && !*req.UseTemplateDefaults {
			// User explicitly set use_template_defaults=false, return empty fields
			fields = []*configv1.ConfigField{}
		} else {
			// Otherwise, try to get template default values (default behavior)
			templateFields, err := s.getTemplateDefaultFields(ctx, req.Identifier.ServiceName, req.Identifier.GroupId, req.Path, scope)
			if err != nil {
				return nil, err
			}
			fields = templateFields
		}
	} else {
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

		for rows.Next() {
			var path, value string
			if err := rows.Scan(&path, &value); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to scan config field: %v", err)
			}
			fields = append(fields, &configv1.ConfigField{Path: path, Value: value})
		}
	}

	currentVersion := versionToFetch
	if shouldUseTemplateFallback {
		currentVersion = 0 // Indicate this is using template defaults, not a real version
	}

	return &configv1.ScopeConfig{
		VersionInfo:    cv,
		CurrentVersion: currentVersion,
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

	// Add to history
	_, err = tx.ExecContext(ctx, `INSERT INTO config_version_history (config_version_id, version, created_by) VALUES ($1, $2, $3)`,
		configVersionID, newVersion, req.User)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert into config version history: %v", err)
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	var cv configv1.ConfigVersion
	cv.Identifier = req.Identifier
	var createdAt, updatedAt sql.NullTime

	// 1. Update config_version table to set published_version
	query := `UPDATE config_version SET published_version = $1, updated_at = NOW(), updated_by = $2
			  WHERE service_name = $3 AND scope = $4 AND scope_id = $5 AND group_id = $6
			  RETURNING id, latest_version, published_version, created_at, updated_at`

	err = tx.QueryRowContext(ctx, query, req.VersionToPublish, req.User, req.Identifier.ServiceName, scope.String(), scopeID, req.Identifier.GroupId).Scan(
		&cv.Id, &cv.LatestVersion, &cv.PublishedVersion, &createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "config identifier not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update config version: %v", err)
	}

	// 2. Update config_version_history table to set publication audit for this version
	res, err := tx.ExecContext(ctx, "UPDATE config_version_history SET published_by = $1, published_at = NOW() WHERE config_version_id = $2 AND version = $3",
		req.User, cv.Id, req.VersionToPublish)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update publication history: %v", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "version %d not found in history", req.VersionToPublish)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	// Set response fields
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

	var configVersionID int32
	query := `SELECT id FROM config_version
              WHERE service_name = $1 AND scope = $2 AND scope_id = $3 AND group_id = $4`
	err = s.db.QueryRowContext(ctx, query, req.Identifier.ServiceName, scope.String(), scopeID, req.Identifier.GroupId).Scan(&configVersionID)
	if err == sql.ErrNoRows {
		return &configv1.GetConfigHistoryResponse{}, nil // No history if config doesn't exist
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query config version id: %v", err)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10 // Default limit
	}

	historyQuery := `SELECT version, created_at, created_by, published_at, published_by FROM config_version_history
                     WHERE config_version_id = $1 ORDER BY created_at DESC LIMIT $2`
	rows, err := s.db.QueryContext(ctx, historyQuery, configVersionID, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query config history: %v", err)
	}
	defer rows.Close()

	var history []*configv1.VersionHistoryEntry
	for rows.Next() {
		entry := &configv1.VersionHistoryEntry{}
		var createdAt time.Time
		var pbAt sql.NullTime
		var pbBy sql.NullString
		if err := rows.Scan(&entry.Version, &createdAt, &entry.CreatedBy, &pbAt, &pbBy); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan history entry: %v", err)
		}
		entry.CreatedAt = timestamppb.New(createdAt)
		if pbAt.Valid {
			entry.PublishedAt = timestamppb.New(pbAt.Time)
		}
		if pbBy.Valid {
			entry.PublishedBy = &pbBy.String
		}
		history = append(history, entry)
	}

	return &configv1.GetConfigHistoryResponse{History: history}, nil
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

// getTemplateDefaultFields retrieves default values from config template fields
// when no published configuration exists
func (s *server) getTemplateDefaultFields(ctx context.Context, serviceName, groupId, pathFilter string, scope configv1.Scope) ([]*configv1.ConfigField, error) {
	var templateID int32

	// Check if template exists for this service and group
	query := `SELECT id FROM config_template WHERE service_name = $1 AND group_id = $2`
	err := s.db.QueryRowContext(ctx, query, serviceName, groupId).Scan(&templateID)

	if err == sql.ErrNoRows {
		// No template exists, return empty fields
		return []*configv1.ConfigField{}, nil
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query config template: %v", err)
	}

	// Build the query for template fields with default values
	// Only include fields that are displayable for the current scope
	fieldQuery := `SELECT path, default_value 
	               FROM config_template_field 
	               WHERE config_template_id = $1 
	               AND default_value IS NOT NULL 
	               AND default_value != '' 
	               AND ($2 = ANY(display_on) OR 'SYSTEM' = ANY(display_on))`
	args := []interface{}{templateID, scope.String()}

	if pathFilter != "" {
		fieldQuery += " AND path = $3"
		args = append(args, pathFilter)
	}

	rows, err := s.db.QueryContext(ctx, fieldQuery, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query template fields: %v", err)
	}
	defer rows.Close()

	var fields []*configv1.ConfigField
	for rows.Next() {
		var path, defaultValue string
		if err := rows.Scan(&path, &defaultValue); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan template field: %v", err)
		}
		fields = append(fields, &configv1.ConfigField{Path: path, Value: defaultValue})
	}

	return fields, nil
}
