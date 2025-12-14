package httpgateway

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
)

// Gateway represents the HTTP gateway that wraps the gRPC client.
type Gateway struct {
	client configv1.ConfigServiceClient
}

// NewGateway creates a new HTTP gateway.
func NewGateway(client configv1.ConfigServiceClient) *Gateway {
	return &Gateway{client: client}
}

// ListTemplates handles GET /api/v1/templates
// Query parameters:
//   - serviceName (optional): Filter by service name
func (g *Gateway) ListTemplates(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("serviceName")

	resp, err := g.client.ListConfigTemplates(r.Context(), &configv1.ListConfigTemplatesRequest{
		ServiceName: serviceName,
	})
	if err != nil {
		WriteError(w, err)
		return
	}

	response := convertTemplateListToJSON(resp)
	WriteJSON(w, http.StatusOK, response)
}

// GetTemplate handles GET /api/v1/config/{serviceName}/template
//
// @Summary Get configuration template
// @Description Retrieves the template (schema) for a specific service and group. Essential for building dynamic UI forms.
// @Description Returns field definitions including types, labels, default values, and display options.
// @Tags Templates
// @Accept json
// @Produce json
// @Param serviceName path string true "Service name (e.g., payment-service)"
// @Param groupId query string true "Configuration group ID (e.g., stripe)"
// @Success 200 {object} map[string]interface{} "Template metadata with field definitions"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 404 {object} map[string]interface{} "Template not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /config/{serviceName}/template [get]
func (g *Gateway) GetTemplate(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "serviceName")
	groupId := r.URL.Query().Get("groupId")

	if serviceName == "" {
		WriteError(w, &validationError{message: "serviceName is required"})
		return
	}

	if groupId == "" {
		WriteError(w, &validationError{message: "groupId query parameter is required"})
		return
	}

	identifier := &configv1.ConfigIdentifier{
		ServiceName: serviceName,
		GroupId:     groupId,
	}

	template, err := g.client.GetConfigTemplate(r.Context(), &configv1.GetConfigTemplateRequest{
		Identifier: identifier,
	})
	if err != nil {
		WriteError(w, err)
		return
	}

	// Convert to a clean JSON format
	response := convertTemplateToJSON(template)
	WriteJSON(w, http.StatusOK, response)
}

// GetConfig handles GET /api/v1/config/{serviceName}/scope/{scope}
//
// @Summary Get published configuration
// @Description Retrieves the published (active) configuration for a specific service, group, and scope.
// @Description Returns merged values (explicit + defaults) and metadata.
// @Tags Configuration
// @Accept json
// @Produce json
// @Param serviceName path string true "Service name"
// @Param scope path string true "Scope level: SYSTEM, PROJECT, STORE, or USER"
// @Param groupId query string true "Configuration group ID"
// @Param projectId query string false "Project ID (required for PROJECT, STORE, USER scopes)"
// @Param storeId query string false "Store ID (required for STORE, USER scopes)"
// @Param userId query string false "User ID (required for USER scope)"
// @Success 200 {object} map[string]interface{} "Configuration with merged values and metadata"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 404 {object} map[string]interface{} "Configuration not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /config/{serviceName}/scope/{scope} [get]
func (g *Gateway) GetConfig(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "serviceName")
	scopeStr := chi.URLParam(r, "scope")

	identifier, err := buildIdentifier(serviceName, scopeStr, r)
	if err != nil {
		WriteError(w, err)
		return
	}

	config, err := g.client.GetConfig(r.Context(), &configv1.GetConfigRequest{
		Identifier: identifier,
	})
	if err != nil {
		WriteError(w, err)
		return
	}

	response := convertConfigToJSON(config)
	WriteJSON(w, http.StatusOK, response)
}

// GetLatestConfig handles GET /api/v1/config/{serviceName}/scope/{scope}/latest
//
// @Summary Get latest configuration
// @Description Retrieves the latest configuration (including unpublished changes) for a specific service, group, and scope.
// @Description Useful for previewing changes before publishing.
// @Tags Configuration
// @Accept json
// @Produce json
// @Param serviceName path string true "Service name"
// @Param scope path string true "Scope level: SYSTEM, PROJECT, STORE, or USER"
// @Param groupId query string true "Configuration group ID"
// @Param projectId query string false "Project ID (required for PROJECT, STORE, USER scopes)"
// @Param storeId query string false "Store ID (required for STORE, USER scopes)"
// @Param userId query string false "User ID (required for USER scope)"
// @Success 200 {object} map[string]interface{} "Latest configuration with merged values and metadata"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 404 {object} map[string]interface{} "Configuration not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /config/{serviceName}/scope/{scope}/latest [get]
func (g *Gateway) GetLatestConfig(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "serviceName")
	scopeStr := chi.URLParam(r, "scope")

	identifier, err := buildIdentifier(serviceName, scopeStr, r)
	if err != nil {
		WriteError(w, err)
		return
	}

	config, err := g.client.GetLatestConfig(r.Context(), &configv1.GetConfigRequest{
		Identifier: identifier,
	})
	if err != nil {
		WriteError(w, err)
		return
	}

	response := convertConfigToJSON(config)
	WriteJSON(w, http.StatusOK, response)
}

// GetConfigHistory handles GET /api/v1/config/{serviceName}/scope/{scope}/history
//
// @Summary Get configuration version history
// @Description Retrieves the version history for a configuration, showing who made changes and when.
// @Description Returns audit trail with version numbers, timestamps, and user information.
// @Tags Configuration
// @Accept json
// @Produce json
// @Param serviceName path string true "Service name"
// @Param scope path string true "Scope level: SYSTEM, PROJECT, STORE, or USER"
// @Param groupId query string true "Configuration group ID"
// @Param projectId query string false "Project ID (required for PROJECT, STORE, USER scopes)"
// @Param storeId query string false "Store ID (required for STORE, USER scopes)"
// @Param userId query string false "User ID (required for USER scope)"
// @Param limit query int false "Maximum number of history entries to return (default: 10)"
// @Success 200 {object} map[string]interface{} "Version history with audit information"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 404 {object} map[string]interface{} "Configuration not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /config/{serviceName}/scope/{scope}/history [get]
func (g *Gateway) GetConfigHistory(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "serviceName")
	scopeStr := chi.URLParam(r, "scope")

	identifier, err := buildIdentifier(serviceName, scopeStr, r)
	if err != nil {
		WriteError(w, err)
		return
	}

	limit := int32(10) // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil || limitInt <= 0 {
			WriteError(w, &validationError{message: "invalid limit parameter"})
			return
		}
		limit = int32(limitInt)
	}

	history, err := g.client.GetConfigHistory(r.Context(), &configv1.GetConfigHistoryRequest{
		Identifier: identifier,
		Limit:      limit,
	})
	if err != nil {
		WriteError(w, err)
		return
	}

	response := convertHistoryToJSON(history)
	WriteJSON(w, http.StatusOK, response)
}

// PublishConfig handles POST /api/v1/config/{serviceName}/scope/{scope}/publish
//
// @Summary Publish configuration version
// @Description Publishes a specific version of the configuration, making it the active version for client consumption.
// @Description The userName field is required for audit trail purposes.
// @Tags Configuration
// @Accept json
// @Produce json
// @Param serviceName path string true "Service name"
// @Param scope path string true "Scope level: SYSTEM, PROJECT, STORE, or USER"
// @Param groupId query string true "Configuration group ID"
// @Param body body PublishRequest true "Publish request with version and scope identifiers"
// @Success 200 {object} map[string]interface{} "Published configuration version details"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 404 {object} map[string]interface{} "Configuration not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /config/{serviceName}/scope/{scope}/publish [post]
func (g *Gateway) PublishConfig(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "serviceName")
	scopeStr := chi.URLParam(r, "scope")

	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, &validationError{message: "invalid JSON body"})
		return
	}

	if req.Version <= 0 {
		WriteError(w, &validationError{message: "version must be greater than 0"})
		return
	}

	scope, err := parseScope(scopeStr)
	if err != nil {
		WriteError(w, err)
		return
	}

	groupId := r.URL.Query().Get("groupId")
	if groupId == "" {
		WriteError(w, &validationError{message: "groupId query parameter is required"})
		return
	}

	// Use authenticated user email if userName not provided in request body
	userName := req.UserName
	if userName == "" {
		userName = GetUserEmail(r.Context())
	}
	
	// If still empty (no auth), require it
	if userName == "" {
		WriteError(w, &validationError{message: "userName is required when authentication is disabled"})
		return
	}

	identifier := &configv1.ConfigIdentifier{
		ServiceName: serviceName,
		Scope:       scope,
		GroupId:     groupId,
		ProjectId:   req.ProjectId,
		StoreId:     req.StoreId,
		UserId:      req.UserId,
	}
	
	publishResp, err := g.client.PublishVersion(r.Context(), &configv1.PublishVersionRequest{
		Identifier:       identifier,
		VersionToPublish: req.Version,
		User:             userName,
	})
	if err != nil {
		WriteError(w, err)
		return
	}

	response := convertVersionToJSON(publishResp)
	WriteJSON(w, http.StatusOK, response)
}

// PublishRequest represents the request body for publishing a configuration.
type PublishRequest struct {
	Version   int32  `json:"version"`
	UserName  string `json:"userName"`
	ProjectId string `json:"projectId,omitempty"`
	StoreId   string `json:"storeId,omitempty"`
	UserId    string `json:"userId,omitempty"`
}

// validationError is used for validation errors.
type validationError struct {
	message string
}

func (e *validationError) Error() string {
	return e.message
}

// buildIdentifier constructs a ConfigIdentifier from URL parameters and query strings.
func buildIdentifier(serviceName, scopeStr string, r *http.Request) (*configv1.ConfigIdentifier, error) {
	if serviceName == "" {
		return nil, &validationError{message: "serviceName is required"}
	}

	scope, err := parseScope(scopeStr)
	if err != nil {
		return nil, err
	}

	groupId := r.URL.Query().Get("groupId")
	if groupId == "" {
		return nil, &validationError{message: "groupId query parameter is required"}
	}

	identifier := &configv1.ConfigIdentifier{
		ServiceName: serviceName,
		Scope:       scope,
		GroupId:     groupId,
		ProjectId:   r.URL.Query().Get("projectId"),
		StoreId:     r.URL.Query().Get("storeId"),
		UserId:      r.URL.Query().Get("userId"),
	}

	// Validate scope-specific IDs
	switch scope {
	case configv1.Scope_PROJECT:
		if identifier.ProjectId == "" {
			return nil, &validationError{message: "projectId is required for PROJECT scope"}
		}
	case configv1.Scope_STORE:
		if identifier.StoreId == "" {
			return nil, &validationError{message: "storeId is required for STORE scope"}
		}
	case configv1.Scope_USER:
		if identifier.UserId == "" {
			return nil, &validationError{message: "userId is required for USER scope"}
		}
	}

	return identifier, nil
}

// parseScope converts a string scope to the enum value.
func parseScope(scopeStr string) (configv1.Scope, error) {
	switch scopeStr {
	case "SYSTEM", "system":
		return configv1.Scope_SYSTEM, nil
	case "PROJECT", "project":
		return configv1.Scope_PROJECT, nil
	case "STORE", "store":
		return configv1.Scope_STORE, nil
	case "USER", "user":
		return configv1.Scope_USER, nil
	default:
		return configv1.Scope_SCOPE_UNSPECIFIED, &validationError{message: "invalid scope: must be one of SYSTEM, PROJECT, STORE, USER"}
	}
}
