package httpgateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"github.com/go-chi/chi/v5"
)

// AdminGateway extends Gateway with direct DB access for admin operations
// that don't warrant a full gRPC round-trip (e.g. toggling is_active).
type AdminGateway struct {
	*Gateway
	db *sql.DB
}

// NewAdminGateway creates an AdminGateway.
func NewAdminGateway(client configv1.ConfigServiceClient, db *sql.DB) *AdminGateway {
	return &AdminGateway{
		Gateway: NewGateway(client),
		db:      db,
	}
}

// ── Import Template ───────────────────────────────────────────────────────────

// ImportTemplateRequest is the JSON body for POST /api/v1/config/templates.
// It mirrors the YAML file format so the admin UI can parse YAML client-side
// and POST the resulting JSON.
type ImportTemplateRequest struct {
	Service  ImportServiceInfo   `json:"service"`
	Groups   []ImportGroupInfo   `json:"groups"`
	UserName string              `json:"userName"`
}

type ImportServiceInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type ImportGroupInfo struct {
	ID          string              `json:"id"`
	Label       string              `json:"label"`
	Description string              `json:"description"`
	SortOrder   int32               `json:"sortOrder"`
	Fields      []ImportFieldInfo   `json:"fields"`
}

type ImportFieldInfo struct {
	Path         string        `json:"path"`
	Label        string        `json:"label"`
	Description  string        `json:"description"`
	Type         string        `json:"type"`
	DefaultValue string        `json:"defaultValue"`
	DisplayOn    []string      `json:"displayOn"`
	SortOrder    int32         `json:"sortOrder"`
	Options      []ImportOption `json:"options,omitempty"`
}

type ImportOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ImportTemplateResult is returned per-group.
type ImportTemplateResult struct {
	ServiceName string `json:"serviceName"`
	GroupID     string `json:"groupId"`
	Status      string `json:"status"` // "ok" | "error"
	Error       string `json:"error,omitempty"`
}

// ImportTemplate handles POST /api/v1/config/templates
//
// @Summary Import / upsert configuration templates
// @Description Applies one or more configuration template groups from a JSON body (mirrors YAML format).
// @Description Each group is upserted independently; results are returned per group.
// @Tags Templates
// @Accept json
// @Produce json
// @Param body body ImportTemplateRequest true "Template import request"
// @Success 200 {object} map[string]interface{} "Per-group import results"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Router /config/templates [post]
func (ag *AdminGateway) ImportTemplate(w http.ResponseWriter, r *http.Request) {
	var req ImportTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, &validationError{message: "invalid JSON body: " + err.Error()})
		return
	}

	if req.Service.ID == "" {
		WriteError(w, &validationError{message: "service.id is required"})
		return
	}
	if len(req.Groups) == 0 {
		WriteError(w, &validationError{message: "at least one group is required"})
		return
	}

	// Resolve userName (header takes precedence, then body, then context)
	userName := r.Header.Get("X-User-Name")
	if userName == "" {
		userName = req.UserName
	}
	if userName == "" {
		userName = GetUserEmail(r.Context())
	}
	if userName == "" {
		WriteError(w, &validationError{message: "userName is required"})
		return
	}

	results := make([]ImportTemplateResult, 0, len(req.Groups))

	for _, grp := range req.Groups {
		if grp.ID == "" {
			results = append(results, ImportTemplateResult{
				ServiceName: req.Service.ID,
				GroupID:     "(missing)",
				Status:      "error",
				Error:       "group.id is required",
			})
			continue
		}

		fields, err := convertImportFields(grp.Fields)
		if err != nil {
			results = append(results, ImportTemplateResult{
				ServiceName: req.Service.ID,
				GroupID:     grp.ID,
				Status:      "error",
				Error:       err.Error(),
			})
			continue
		}

		template := &configv1.ConfigTemplate{
			Identifier: &configv1.ConfigIdentifier{
				ServiceName: req.Service.ID,
				GroupId:     grp.ID,
			},
			ServiceLabel:     req.Service.Label,
			GroupLabel:       grp.Label,
			GroupDescription: grp.Description,
			SortOrder:        grp.SortOrder,
			Fields:           fields,
		}

		_, grpcErr := ag.client.ApplyConfigTemplate(r.Context(), &configv1.ApplyConfigTemplateRequest{
			Template: template,
			User:     userName,
		})

		if grpcErr != nil {
			results = append(results, ImportTemplateResult{
				ServiceName: req.Service.ID,
				GroupID:     grp.ID,
				Status:      "error",
				Error:       grpcErr.Error(),
			})
		} else {
			results = append(results, ImportTemplateResult{
				ServiceName: req.Service.ID,
				GroupID:     grp.ID,
				Status:      "ok",
			})
		}
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

func convertImportFields(fields []ImportFieldInfo) ([]*configv1.ConfigFieldTemplate, error) {
	out := make([]*configv1.ConfigFieldTemplate, 0, len(fields))
	for _, f := range fields {
		if f.Path == "" {
			return nil, &validationError{message: "field.path is required"}
		}

		// Map type string → proto enum
		ft, ok := configv1.FieldType_value[f.Type]
		if !ok || ft == 0 {
			// Default to STRING for unknown types
			ft = int32(configv1.FieldType_STRING)
		}

		// Map displayOn strings → proto scopes
		displayOn := make([]configv1.Scope, 0, len(f.DisplayOn))
		for _, s := range f.DisplayOn {
			if v, ok := configv1.Scope_value[s]; ok && v != 0 {
				displayOn = append(displayOn, configv1.Scope(v))
			}
		}

		// Map options
		options := make([]*configv1.ValueOption, 0, len(f.Options))
		for _, o := range f.Options {
			options = append(options, &configv1.ValueOption{Value: o.Value, Label: o.Label})
		}

		out = append(out, &configv1.ConfigFieldTemplate{
			Path:         f.Path,
			Label:        f.Label,
			Description:  f.Description,
			Type:         configv1.FieldType(ft),
			DefaultValue: f.DefaultValue,
			DisplayOn:    displayOn,
			SortOrder:    f.SortOrder,
			Options:      options,
		})
	}
	return out, nil
}

// ── Set Template Active ───────────────────────────────────────────────────────

// SetTemplateActiveRequest is the JSON body for PATCH /api/v1/config/templates/{svc}/{grp}/active
type SetTemplateActiveRequest struct {
	Active bool `json:"active"`
}

// SetTemplateActive handles PATCH /api/v1/config/templates/{serviceName}/{groupId}/active
//
// @Summary Enable or disable a configuration template
// @Description Toggles the is_active flag on a template. Inactive templates are hidden from
// @Description the normal config UI and not returned by GET /api/v1/config/templates (unless includeInactive=true).
// @Tags Templates
// @Accept json
// @Produce json
// @Param serviceName path string true "Service name"
// @Param groupId path string true "Group ID"
// @Param body body SetTemplateActiveRequest true "Active state to set"
// @Success 200 {object} map[string]interface{} "Updated active state"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Template not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /config/templates/{serviceName}/{groupId}/active [patch]
func (ag *AdminGateway) SetTemplateActive(w http.ResponseWriter, r *http.Request) {
	if ag.db == nil {
		WriteError(w, &validationError{message: "database not available for this operation"})
		return
	}

	serviceName := chi.URLParam(r, "serviceName")
	groupId := chi.URLParam(r, "groupId")

	if serviceName == "" || groupId == "" {
		WriteError(w, &validationError{message: "serviceName and groupId are required"})
		return
	}

	var req SetTemplateActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, &validationError{message: "invalid JSON body"})
		return
	}

	result, err := ag.db.ExecContext(
		context.Background(),
		`UPDATE config_template SET is_active = $1, updated_at = CURRENT_TIMESTAMP WHERE service_name = $2 AND group_id = $3`,
		req.Active, serviceName, groupId,
	)
	if err != nil {
		WriteError(w, err)
		return
	}

	rows, err := result.RowsAffected()
	if err != nil {
		WriteError(w, err)
		return
	}
	if rows == 0 {
		http.Error(w, `{"message":"template not found"}`, http.StatusNotFound)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"serviceName": serviceName,
		"groupId":     groupId,
		"active":      req.Active,
	})
}

// ListAllTemplates handles GET /api/v1/config/templates
//
// @Summary List configuration templates
// @Description Retrieves a list of configuration templates. By default, only active templates are returned.
// @Description Use ?includeInactive=true to return both active and inactive templates (useful for admin UI).
// @Tags Templates
// @Accept json
// @Produce json
// @Param serviceName query string false "Filter by service name"
// @Param includeInactive query boolean false "Include inactive templates"
// @Success 200 {object} map[string]interface{} "List of templates"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /config/templates [get]
func (ag *AdminGateway) ListAllTemplates(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("serviceName")
	includeInactive := r.URL.Query().Get("includeInactive") == "true"

	var req *configv1.ListConfigTemplatesRequest
	if includeInactive {
		// No is_active filter → returns all templates
		req = &configv1.ListConfigTemplatesRequest{
			ServiceName: serviceName,
		}
	} else {
		isActive := true
		req = &configv1.ListConfigTemplatesRequest{
			ServiceName: serviceName,
			IsActive:    &isActive,
		}
	}

	resp, err := ag.client.ListConfigTemplates(r.Context(), req)
	if err != nil {
		WriteError(w, err)
		return
	}

	// We need to return is_active status too; query the DB for that.
	// Build a map of (service_name, group_id) → is_active
	activeMap := map[string]bool{}
	if ag.db != nil {
		rows, dbErr := ag.db.QueryContext(r.Context(),
			`SELECT service_name, group_id, is_active FROM config_template`)
		if dbErr == nil {
			defer rows.Close()
			for rows.Next() {
				var svc, grp string
				var active bool
				if rows.Scan(&svc, &grp, &active) == nil {
					activeMap[svc+"/"+grp] = active
				}
			}
		}
	}

	type TemplateWithActive struct {
		ServiceName  string `json:"serviceName"`
		ServiceLabel string `json:"serviceLabel"`
		GroupId      string `json:"groupId"`
		GroupLabel   string `json:"groupLabel"`
		Description  string `json:"description"`
		SortOrder    int32  `json:"sortOrder"`
		IsActive     bool   `json:"isActive"`
	}

	templates := make([]TemplateWithActive, 0, len(resp.Templates))
	for _, t := range resp.Templates {
		key := t.Identifier.ServiceName + "/" + t.Identifier.GroupId
		isActive, exists := activeMap[key]
		if !exists {
			isActive = true // default if not in DB (shouldn't happen)
		}
		templates = append(templates, TemplateWithActive{
			ServiceName:  t.Identifier.ServiceName,
			ServiceLabel: t.ServiceLabel,
			GroupId:      t.Identifier.GroupId,
			GroupLabel:   t.GroupLabel,
			Description:  t.GroupDescription,
			SortOrder:    t.SortOrder,
			IsActive:     isActive,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"templates": templates,
	})
}
