package service

import (
	"context"
	"database/sql"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/lib/pq"
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
        INSERT INTO config_template (service_name, group_id, created_by, updated_by)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (service_name, group_id) DO UPDATE
        SET updated_at = NOW(), updated_by = $4
        RETURNING id`

	err = tx.QueryRowContext(ctx, upsertQuery,
		template.Identifier.ServiceName,
		template.Identifier.GroupId,
		req.User,
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
        INSERT INTO config_template_field (config_template_id, path, label, description, type, default_value, display_on)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`)
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

		_, err := stmt.ExecContext(ctx, templateID, field.Path, field.Label, field.Description, field.Type.String(), field.DefaultValue, pq.Array(displayOn))
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

	// Find the template ID.
	var templateID int32
	query := `SELECT id FROM config_template WHERE service_name = $1 AND group_id = $2`
	err := s.db.QueryRowContext(ctx, query, identifier.ServiceName, identifier.GroupId).Scan(&templateID)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "template not found for service '%s' and group '%s'", identifier.ServiceName, identifier.GroupId)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query for template: %v", err)
	}

	// Fetch the fields for the found template.
	rows, err := s.db.QueryContext(ctx, `
		SELECT path, label, description, type, default_value, display_on
		FROM config_template_field WHERE config_template_id = $1 ORDER BY path ASC`, templateID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query template fields: %v", err)
	}
	defer rows.Close()

	var fields []*configv1.ConfigFieldTemplate
	for rows.Next() {
		field := &configv1.ConfigFieldTemplate{}
		var fieldType string
		var displayOn []string

		if err := rows.Scan(&field.Path, &field.Label, &field.Description, &fieldType, &field.DefaultValue, pq.Array(&displayOn)); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan template field: %v", err)
		}

		// Convert string type from DB to enum type for proto.
		if val, ok := configv1.FieldType_value[fieldType]; ok {
			field.Type = configv1.FieldType(val)
		}

		// Convert string array from DB to repeated enum scope for proto.
		field.DisplayOn = make([]configv1.Scope, len(displayOn))
		for i, s := range displayOn {
			if val, ok := configv1.Scope_value[s]; ok {
				field.DisplayOn[i] = configv1.Scope(val)
			}
		}

		fields = append(fields, field)
	}

	return &configv1.ConfigTemplate{
		Identifier: identifier,
		Fields:     fields,
	}, nil
}
