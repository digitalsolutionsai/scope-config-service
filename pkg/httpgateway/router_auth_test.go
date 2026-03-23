package httpgateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
	"google.golang.org/grpc"
)

// routerTestClient implements configv1.ConfigServiceClient for router integration tests.
// All methods return empty successful responses so we can test auth behavior.
type routerTestClient struct {
	configv1.ConfigServiceClient
}

func (c *routerTestClient) GetConfigTemplate(ctx context.Context, req *configv1.GetConfigTemplateRequest, opts ...grpc.CallOption) (*configv1.ConfigTemplate, error) {
	return &configv1.ConfigTemplate{
		Identifier: &configv1.ConfigIdentifier{ServiceName: "test", GroupId: "test"},
	}, nil
}

func (c *routerTestClient) ListConfigTemplates(ctx context.Context, req *configv1.ListConfigTemplatesRequest, opts ...grpc.CallOption) (*configv1.ListConfigTemplatesResponse, error) {
	return &configv1.ListConfigTemplatesResponse{}, nil
}

func TestRouter_SwaggerPublicWithBasicAuth(t *testing.T) {
	router := NewRouterWithConfig(RouterConfig{
		Client:              &routerTestClient{},
		BasicAuthMiddleware: NewBasicAuthMiddleware("admin", "secret"),
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/swagger/index.html")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("swagger with basic auth configured: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestRouter_ProtectedRoutesRequireBasicAuth(t *testing.T) {
	router := NewRouterWithConfig(RouterConfig{
		Client:              &routerTestClient{},
		BasicAuthMiddleware: NewBasicAuthMiddleware("admin", "secret"),
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	routes := []string{
		"/admin",
		"/api/v1/config/templates",
		"/api/v1/config/myservice/template?groupId=test",
	}

	for _, route := range routes {
		resp, err := http.Get(ts.URL + route)
		if err != nil {
			t.Fatalf("request to %s failed: %v", route, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s without credentials: got %d, want %d", route, resp.StatusCode, http.StatusUnauthorized)
		}
	}
}

func TestRouter_ProtectedRoutesAccessibleWithBasicAuth(t *testing.T) {
	router := NewRouterWithConfig(RouterConfig{
		Client:              &routerTestClient{},
		BasicAuthMiddleware: NewBasicAuthMiddleware("admin", "secret"),
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	routes := []string{
		"/admin",
		"/api/v1/config/templates",
	}

	client := &http.Client{}
	for _, route := range routes {
		req, _ := http.NewRequest("GET", ts.URL+route, nil)
		req.SetBasicAuth("admin", "secret")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request to %s failed: %v", route, err)
		}
		resp.Body.Close()

		// Should not be 401 — the middleware should let it through
		if resp.StatusCode == http.StatusUnauthorized {
			t.Errorf("%s with valid credentials: got 401, expected non-401", route)
		}
	}
}

func TestRouter_NoAuthConfigured(t *testing.T) {
	router := NewRouterWithConfig(RouterConfig{
		Client: &routerTestClient{},
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	routes := []string{
		"/admin",
		"/api/v1/config/templates",
	}

	for _, route := range routes {
		resp, err := http.Get(ts.URL + route)
		if err != nil {
			t.Fatalf("request to %s failed: %v", route, err)
		}
		resp.Body.Close()

		// Should not be 401 when no auth is configured
		if resp.StatusCode == http.StatusUnauthorized {
			t.Errorf("%s without any auth configured: got 401, expected non-401", route)
		}
	}
}
