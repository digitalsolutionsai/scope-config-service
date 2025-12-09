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

// GetTemplate handles GET /api/v1/templates/{serviceName}
// Query parameters:
//   - groupId (required): The group ID for the template
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
// Query parameters:
//   - groupId (required): The group ID
//   - projectId (optional): Project ID for PROJECT scope
//   - storeId (optional): Store ID for STORE scope
//   - userId (optional): User ID for USER scope
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
// Query parameters: same as GetConfig
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
// Query parameters: same as GetConfig, plus:
//   - limit (optional): Maximum number of history entries to return
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
// Request body:
//   {
//     "version": 3,
//     "userName": "John Doe",
//     "projectId": "project-123",  // optional
//     "storeId": null,              // optional
//     "userId": null                // optional
//   }
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
