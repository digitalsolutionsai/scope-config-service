package httpgateway

import (
	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
)

// TemplateResponse represents a clean JSON response for a template.
type TemplateResponse struct {
	ServiceName  string                      `json:"serviceName"`
	ServiceLabel string                      `json:"serviceLabel"`
	GroupId      string                      `json:"groupId"`
	GroupLabel   string                      `json:"groupLabel"`
	Description  string                      `json:"description"`
	Fields       []TemplateFieldResponse     `json:"fields"`
}

// TemplateFieldResponse represents a single field in the template.
type TemplateFieldResponse struct {
	Path         string              `json:"path"`
	Label        string              `json:"label"`
	Description  string              `json:"description"`
	Type         string              `json:"type"`
	DefaultValue string              `json:"defaultValue"`
	DisplayOn    []string            `json:"displayOn"`
	Options      []ValueOptionResponse `json:"options,omitempty"`
}

// ValueOptionResponse represents a value option.
type ValueOptionResponse struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ConfigResponse represents a clean JSON response for a configuration.
type ConfigResponse struct {
	ServiceName      string                 `json:"serviceName"`
	Scope            string                 `json:"scope"`
	GroupId          string                 `json:"groupId"`
	ProjectId        string                 `json:"projectId,omitempty"`
	StoreId          string                 `json:"storeId,omitempty"`
	UserId           string                 `json:"userId,omitempty"`
	CurrentVersion   int32                  `json:"currentVersion"`
	LatestVersion    int32                  `json:"latestVersion"`
	PublishedVersion int32                  `json:"publishedVersion,omitempty"`
	Fields           map[string]string      `json:"fields"`
	CreatedAt        string                 `json:"createdAt,omitempty"`
	UpdatedAt        string                 `json:"updatedAt,omitempty"`
}

// HistoryResponse represents a clean JSON response for version history.
type HistoryResponse struct {
	History []HistoryEntryResponse `json:"history"`
}

// HistoryEntryResponse represents a single history entry.
type HistoryEntryResponse struct {
	Version   int32  `json:"version"`
	CreatedAt string `json:"createdAt"`
	CreatedBy string `json:"createdBy"`
}

// VersionResponse represents a clean JSON response for a config version.
type VersionResponse struct {
	ServiceName      string `json:"serviceName"`
	Scope            string `json:"scope"`
	GroupId          string `json:"groupId"`
	ProjectId        string `json:"projectId,omitempty"`
	StoreId          string `json:"storeId,omitempty"`
	UserId           string `json:"userId,omitempty"`
	LatestVersion    int32  `json:"latestVersion"`
	PublishedVersion int32  `json:"publishedVersion"`
	CreatedAt        string `json:"createdAt,omitempty"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
}

// convertTemplateToJSON converts a gRPC template response to a clean JSON format.
func convertTemplateToJSON(template *configv1.ConfigTemplate) TemplateResponse {
	fields := make([]TemplateFieldResponse, len(template.Fields))
	for i, field := range template.Fields {
		displayOn := make([]string, len(field.DisplayOn))
		for j, scope := range field.DisplayOn {
			displayOn[j] = scope.String()
		}

		options := make([]ValueOptionResponse, len(field.Options))
		for j, opt := range field.Options {
			options[j] = ValueOptionResponse{
				Value: opt.Value,
				Label: opt.Label,
			}
		}

		fields[i] = TemplateFieldResponse{
			Path:         field.Path,
			Label:        field.Label,
			Description:  field.Description,
			Type:         field.Type.String(),
			DefaultValue: field.DefaultValue,
			DisplayOn:    displayOn,
			Options:      options,
		}
	}

	return TemplateResponse{
		ServiceName:  template.Identifier.ServiceName,
		ServiceLabel: template.ServiceLabel,
		GroupId:      template.Identifier.GroupId,
		GroupLabel:   template.GroupLabel,
		Description:  template.GroupDescription,
		Fields:       fields,
	}
}

// convertConfigToJSON converts a gRPC config response to a clean JSON format.
func convertConfigToJSON(config *configv1.ScopeConfig) ConfigResponse {
	fields := make(map[string]string)
	for _, field := range config.Fields {
		fields[field.Path] = field.Value
	}

	response := ConfigResponse{
		CurrentVersion: config.CurrentVersion,
		Fields:         fields,
	}

	if config.VersionInfo != nil {
		response.LatestVersion = config.VersionInfo.LatestVersion
		response.PublishedVersion = config.VersionInfo.PublishedVersion

		if config.VersionInfo.Identifier != nil {
			response.ServiceName = config.VersionInfo.Identifier.ServiceName
			response.Scope = config.VersionInfo.Identifier.Scope.String()
			response.GroupId = config.VersionInfo.Identifier.GroupId
			response.ProjectId = config.VersionInfo.Identifier.ProjectId
			response.StoreId = config.VersionInfo.Identifier.StoreId
			response.UserId = config.VersionInfo.Identifier.UserId
		}

		if config.VersionInfo.CreatedAt != nil {
			response.CreatedAt = config.VersionInfo.CreatedAt.AsTime().Format("2006-01-02T15:04:05Z")
		}
		if config.VersionInfo.UpdatedAt != nil {
			response.UpdatedAt = config.VersionInfo.UpdatedAt.AsTime().Format("2006-01-02T15:04:05Z")
		}
	}

	return response
}

// convertHistoryToJSON converts a gRPC history response to a clean JSON format.
func convertHistoryToJSON(history *configv1.GetConfigHistoryResponse) HistoryResponse {
	entries := make([]HistoryEntryResponse, len(history.History))
	for i, entry := range history.History {
		createdAt := ""
		if entry.CreatedAt != nil {
			createdAt = entry.CreatedAt.AsTime().Format("2006-01-02T15:04:05Z")
		}

		entries[i] = HistoryEntryResponse{
			Version:   entry.Version,
			CreatedAt: createdAt,
			CreatedBy: entry.CreatedBy,
		}
	}

	return HistoryResponse{
		History: entries,
	}
}

// convertVersionToJSON converts a gRPC version response to a clean JSON format.
func convertVersionToJSON(version *configv1.ConfigVersion) VersionResponse {
	response := VersionResponse{
		LatestVersion:    version.LatestVersion,
		PublishedVersion: version.PublishedVersion,
	}

	if version.Identifier != nil {
		response.ServiceName = version.Identifier.ServiceName
		response.Scope = version.Identifier.Scope.String()
		response.GroupId = version.Identifier.GroupId
		response.ProjectId = version.Identifier.ProjectId
		response.StoreId = version.Identifier.StoreId
		response.UserId = version.Identifier.UserId
	}

	if version.CreatedAt != nil {
		response.CreatedAt = version.CreatedAt.AsTime().Format("2006-01-02T15:04:05Z")
	}
	if version.UpdatedAt != nil {
		response.UpdatedAt = version.UpdatedAt.AsTime().Format("2006-01-02T15:04:05Z")
	}

	return response
}

// TemplateListResponse represents a list of templates.
type TemplateListResponse struct {
	Templates []TemplateResponse `json:"templates"`
}

// convertTemplateListToJSON converts a gRPC template list response to a clean JSON format.
func convertTemplateListToJSON(list *configv1.ListConfigTemplatesResponse) TemplateListResponse {
	templates := make([]TemplateResponse, len(list.Templates))
	for i, tmpl := range list.Templates {
		templates[i] = convertTemplateToJSON(tmpl)
	}
	return TemplateListResponse{Templates: templates}
}
