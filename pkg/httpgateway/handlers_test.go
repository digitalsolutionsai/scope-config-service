package httpgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mockConfigServiceClient is a mock implementation of the gRPC client for testing.
type mockConfigServiceClient struct {
	configv1.ConfigServiceClient
	GetConfigTemplateFunc    func(ctx context.Context, req *configv1.GetConfigTemplateRequest) (*configv1.ConfigTemplate, error)
	GetConfigFunc            func(ctx context.Context, req *configv1.GetConfigRequest) (*configv1.ScopeConfig, error)
	GetLatestConfigFunc      func(ctx context.Context, req *configv1.GetConfigRequest) (*configv1.ScopeConfig, error)
	GetConfigHistoryFunc     func(ctx context.Context, req *configv1.GetConfigHistoryRequest) (*configv1.GetConfigHistoryResponse, error)
	PublishVersionFunc       func(ctx context.Context, req *configv1.PublishVersionRequest) (*configv1.ConfigVersion, error)
}

func (m *mockConfigServiceClient) GetConfigTemplate(ctx context.Context, req *configv1.GetConfigTemplateRequest, opts ...grpc.CallOption) (*configv1.ConfigTemplate, error) {
	if m.GetConfigTemplateFunc != nil {
		return m.GetConfigTemplateFunc(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockConfigServiceClient) GetConfig(ctx context.Context, req *configv1.GetConfigRequest, opts ...grpc.CallOption) (*configv1.ScopeConfig, error) {
	if m.GetConfigFunc != nil {
		return m.GetConfigFunc(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockConfigServiceClient) GetLatestConfig(ctx context.Context, req *configv1.GetConfigRequest, opts ...grpc.CallOption) (*configv1.ScopeConfig, error) {
	if m.GetLatestConfigFunc != nil {
		return m.GetLatestConfigFunc(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockConfigServiceClient) GetConfigHistory(ctx context.Context, req *configv1.GetConfigHistoryRequest, opts ...grpc.CallOption) (*configv1.GetConfigHistoryResponse, error) {
	if m.GetConfigHistoryFunc != nil {
		return m.GetConfigHistoryFunc(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockConfigServiceClient) PublishVersion(ctx context.Context, req *configv1.PublishVersionRequest, opts ...grpc.CallOption) (*configv1.ConfigVersion, error) {
	if m.PublishVersionFunc != nil {
		return m.PublishVersionFunc(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestGetTemplate(t *testing.T) {
	tests := []struct {
		name           string
		serviceName    string
		queryParams    string
		mockResponse   *configv1.ConfigTemplate
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful template retrieval",
			serviceName: "payment-service",
			queryParams: "?groupId=stripe",
			mockResponse: &configv1.ConfigTemplate{
				Identifier: &configv1.ConfigIdentifier{
					ServiceName: "payment-service",
					GroupId:     "stripe",
				},
				ServiceLabel:     "Payment Service",
				GroupLabel:       "Stripe Configuration",
				GroupDescription: "Stripe payment gateway settings",
				Fields: []*configv1.ConfigFieldTemplate{
					{
						Path:         "stripe.apiKey",
						Label:        "API Key",
						Description:  "Stripe API key",
						Type:         configv1.FieldType_STRING,
						DefaultValue: "",
						DisplayOn:    []configv1.Scope{configv1.Scope_SYSTEM},
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing groupId parameter",
			serviceName:    "payment-service",
			queryParams:    "",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "template not found",
			serviceName:    "unknown-service",
			queryParams:    "?groupId=unknown",
			mockError:      status.Error(codes.NotFound, "template not found"),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockConfigServiceClient{
				GetConfigTemplateFunc: func(ctx context.Context, req *configv1.GetConfigTemplateRequest) (*configv1.ConfigTemplate, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			gateway := NewGateway(mockClient)

			req := httptest.NewRequest("GET", "/api/v1/templates/"+tt.serviceName+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Set up chi context
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("serviceName", tt.serviceName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			gateway.GetTemplate(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("GetTemplate status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response TemplateResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if response.ServiceName != tt.mockResponse.Identifier.ServiceName {
					t.Errorf("ServiceName = %q, want %q", response.ServiceName, tt.mockResponse.Identifier.ServiceName)
				}
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name           string
		serviceName    string
		scope          string
		queryParams    string
		mockResponse   *configv1.ScopeConfig
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful config retrieval - PROJECT scope",
			serviceName: "payment-service",
			scope:       "PROJECT",
			queryParams: "?groupId=stripe&projectId=proj-123",
			mockResponse: &configv1.ScopeConfig{
				VersionInfo: &configv1.ConfigVersion{
					Identifier: &configv1.ConfigIdentifier{
						ServiceName: "payment-service",
						Scope:       configv1.Scope_PROJECT,
						GroupId:     "stripe",
						ProjectId:   "proj-123",
					},
					LatestVersion:    2,
					PublishedVersion: 1,
				},
				CurrentVersion: 1,
				Fields: []*configv1.ConfigField{
					{Path: "stripe.apiKey", Value: "sk_test_123"},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing projectId for PROJECT scope",
			serviceName:    "payment-service",
			scope:          "PROJECT",
			queryParams:    "?groupId=stripe",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "invalid scope",
			serviceName:    "payment-service",
			scope:          "INVALID",
			queryParams:    "?groupId=stripe",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockConfigServiceClient{
				GetConfigFunc: func(ctx context.Context, req *configv1.GetConfigRequest) (*configv1.ScopeConfig, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			gateway := NewGateway(mockClient)

			req := httptest.NewRequest("GET", "/api/v1/config/"+tt.serviceName+"/scope/"+tt.scope+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Set up chi context
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("serviceName", tt.serviceName)
			rctx.URLParams.Add("scope", tt.scope)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			gateway.GetConfig(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("GetConfig status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response ConfigResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if response.ServiceName != tt.mockResponse.VersionInfo.Identifier.ServiceName {
					t.Errorf("ServiceName = %q, want %q", response.ServiceName, tt.mockResponse.VersionInfo.Identifier.ServiceName)
				}
			}
		})
	}
}

func TestPublishConfig(t *testing.T) {
	tests := []struct {
		name           string
		serviceName    string
		scope          string
		queryParams    string
		requestBody    PublishRequest
		mockResponse   *configv1.ConfigVersion
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful publish",
			serviceName: "payment-service",
			scope:       "PROJECT",
			queryParams: "?groupId=stripe",
			requestBody: PublishRequest{
				Version:   2,
				UserName:  "John Doe",
				ProjectId: "proj-123",
			},
			mockResponse: &configv1.ConfigVersion{
				Identifier: &configv1.ConfigIdentifier{
					ServiceName: "payment-service",
					Scope:       configv1.Scope_PROJECT,
					GroupId:     "stripe",
					ProjectId:   "proj-123",
				},
				LatestVersion:    2,
				PublishedVersion: 2,
				CreatedAt:        timestamppb.Now(),
				UpdatedAt:        timestamppb.Now(),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "missing userName",
			serviceName: "payment-service",
			scope:       "PROJECT",
			queryParams: "?groupId=stripe",
			requestBody: PublishRequest{
				Version:   2,
				ProjectId: "proj-123",
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "invalid version",
			serviceName: "payment-service",
			scope:       "PROJECT",
			queryParams: "?groupId=stripe",
			requestBody: PublishRequest{
				Version:   0,
				UserName:  "John Doe",
				ProjectId: "proj-123",
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockConfigServiceClient{
				PublishVersionFunc: func(ctx context.Context, req *configv1.PublishVersionRequest) (*configv1.ConfigVersion, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			gateway := NewGateway(mockClient)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/config/"+tt.serviceName+"/scope/"+tt.scope+"/publish"+tt.queryParams, bytes.NewReader(body))
			w := httptest.NewRecorder()

			// Set up chi context
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("serviceName", tt.serviceName)
			rctx.URLParams.Add("scope", tt.scope)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			gateway.PublishConfig(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("PublishConfig status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response VersionResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if response.PublishedVersion != tt.mockResponse.PublishedVersion {
					t.Errorf("PublishedVersion = %d, want %d", response.PublishedVersion, tt.mockResponse.PublishedVersion)
				}
			}
		})
	}
}

func TestParseScope(t *testing.T) {
	tests := []struct {
		input    string
		expected configv1.Scope
		wantErr  bool
	}{
		{"SYSTEM", configv1.Scope_SYSTEM, false},
		{"system", configv1.Scope_SYSTEM, false},
		{"PROJECT", configv1.Scope_PROJECT, false},
		{"project", configv1.Scope_PROJECT, false},
		{"STORE", configv1.Scope_STORE, false},
		{"store", configv1.Scope_STORE, false},
		{"USER", configv1.Scope_USER, false},
		{"user", configv1.Scope_USER, false},
		{"INVALID", configv1.Scope_SCOPE_UNSPECIFIED, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseScope(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseScope(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("parseScope(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
