package service

import (
	"context"
	"database/sql"
	"encoding/json"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApplyConfigTemplate applies a schema to a configuration.
func (s *server) ApplyConfigTemplate(ctx context.Context, req *configv1.ApplyConfigTemplateRequest) (*configv1.ConfigTemplate, error) {
	template := req.Template
	if template == nil || template.Identifier == nil {
		return nil, status.Error(codes.InvalidArgument, "template and identifier cannot be nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback() // Rollback on any error.

	var templateID int32
	upsertQuery := `
		INSERT INTO config_template (service_name, group_id, service_label, group_label, group_description, sort_order, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (service_name, group_id) DO UPDATE
		SET service_label = $3, group_label = $4, group_description = $5, sort_order = $6, updated_at = CURRENT_TIMESTAMP, updated_by = $7
		RETURNING id`

	err = tx.QueryRowContext(ctx, upsertQuery,
		template.Identifier.ServiceName,
		template.Identifier.GroupId,
		template.ServiceLabel,
		template.GroupLabel,
		template.GroupDescription,
		template.SortOrder,
		req.User,
	).Scan(&templateID)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upsert config template: %v", err)
	}

	// Delete old fields to ensure a clean slate.
	_, err = tx.ExecContext(ctx, "DELETE FROM config_template_field WHERE config_template_id = $1", templateID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete old template fields: %v", err)
	}

	// Prepare statement for inserting new fields.
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO config_template_field (config_template_id, path, label, description, type, default_value, display_on, options, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to prepare field template insert: %v", err)
	}
	defer stmt.Close()

	for _, field := range template.Fields {
		// Convert the repeated Scope enum to a string array for the pq driver.
		displayOn := make([]string, len(field.DisplayOn))
		for i, scope := range field.DisplayOn {
			displayOn[i] = scope.String()
		}

		// Marshal options to JSONB
		var optionsJSON []byte
		if len(field.Options) > 0 {
			optionsJSON, err = json.Marshal(field.Options)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to marshal options for field '%s': %v", field.Path, err)
			}
		} else {
			optionsJSON = []byte("null")
		}

		arrayVal, err := arrayParam(s.dialect, displayOn)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to encode display_on for field '%s': %v", field.Path, err)
		}

		_, err = stmt.ExecContext(ctx, templateID, field.Path, field.Label, field.Description, field.Type.String(), field.DefaultValue, arrayVal, optionsJSON, field.SortOrder)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to insert template field '%s': %v", field.Path, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return template, nil
}

// GetConfigTemplate retrieves a configuration template.
func (s *server) GetConfigTemplate(ctx context.Context, req *configv1.GetConfigTemplateRequest) (*configv1.ConfigTemplate, error) {
	identifier := req.Identifier
	if identifier == nil {
		return nil, status.Error(codes.InvalidArgument, "identifier cannot be nil")
	}

	template := &configv1.ConfigTemplate{
		Identifier: identifier,
	}
	var templateID int32

	// Find the template and its metadata
	query := `SELECT id, service_label, group_label, group_description, sort_order FROM config_template WHERE service_name = $1 AND group_id = $2`
	err := s.db.QueryRowContext(ctx, query, identifier.ServiceName, identifier.GroupId).Scan(
		&templateID,
		&template.ServiceLabel,
		&template.GroupLabel,
		&template.GroupDescription,
		&template.SortOrder,
	)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "template not found for service '%s' and group '%s'", identifier.ServiceName, identifier.GroupId)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query for template: %v", err)
	}

	// Fetch the fields for the found template.
	rows, err := s.db.QueryContext(ctx, `
		SELECT path, label, description, type, default_value, display_on, options, sort_order
		FROM config_template_field WHERE config_template_id = $1 ORDER BY sort_order ASC, path ASC`, templateID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query template fields: %v", err)
	}
	defer rows.Close()

	var fields []*configv1.ConfigFieldTemplate
	for rows.Next() {
		field := &configv1.ConfigFieldTemplate{}
		var fieldType string
		var displayOn []string
		var optionsJSON sql.NullString // Use sql.NullString to handle NULL JSONB

		if err := rows.Scan(&field.Path, &field.Label, &field.Description, &fieldType, &field.DefaultValue, newArrayScanner(s.dialect, &displayOn), &optionsJSON, &field.SortOrder); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan template field: %v", err)
		}

		// Unmarshal options from JSONB
		if optionsJSON.Valid {
			if err := json.Unmarshal([]byte(optionsJSON.String), &field.Options); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to unmarshal options for field '%s': %v", field.Path, err)
			}
		}

		// Convert string type from DB to enum type for proto.
		if val, ok := configv1.FieldType_value[fieldType]; ok {
			field.Type = configv1.FieldType(val)
		}

		// Convert string array from DB to repeated enum scope for proto.
		if displayOn != nil {
			field.DisplayOn = make([]configv1.Scope, len(displayOn))
			for i, s := range displayOn {
				if val, ok := configv1.Scope_value[s]; ok {
					field.DisplayOn[i] = configv1.Scope(val)
				}
			}
		} else {
			field.DisplayOn = make([]configv1.Scope, 0)
		}

		fields = append(fields, field)
	}
	template.Fields = fields

	return template, nil
}

// ListConfigTemplates retrieves a list of configuration templates.
func (s *server) ListConfigTemplates(ctx context.Context, req *configv1.ListConfigTemplatesRequest) (*configv1.ListConfigTemplatesResponse, error) {
	query := `SELECT service_name, group_id, service_label, group_label, group_description, sort_order 
              FROM config_template`

	var args []interface{}
	if req.ServiceName != "" {
		query += ` WHERE service_name = $1`
		args = append(args, req.ServiceName)
	}

	query += ` ORDER BY service_name, sort_order ASC, group_id`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query templates: %v", err)
	}
	defer rows.Close()

	var templates []*configv1.ConfigTemplate
	for rows.Next() {
		var serviceName, groupID string
		var serviceLabel, groupLabel, groupDescription sql.NullString
		var sortOrder int32

		if err := rows.Scan(&serviceName, &groupID, &serviceLabel, &groupLabel, &groupDescription, &sortOrder); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan template row: %v", err)
		}

		// ConfigIdentifier requires Scope, but for a template list it's generic.
		// We set it to UNSPECIFIED or leave default (0).
		identifier := &configv1.ConfigIdentifier{
			ServiceName: serviceName,
			GroupId:     groupID,
			Scope:       configv1.Scope_SCOPE_UNSPECIFIED,
		}

		templates = append(templates, &configv1.ConfigTemplate{
			Identifier:       identifier,
			ServiceLabel:     serviceLabel.String,
			GroupLabel:       groupLabel.String,
			GroupDescription: groupDescription.String,
			SortOrder:        sortOrder,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "rows iteration error: %v", err)
	}

	return &configv1.ListConfigTemplatesResponse{
		Templates: templates,
	}, nil
}
