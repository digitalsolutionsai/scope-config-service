package httpgateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// okHandler is a simple handler that returns 200 OK.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestNewBasicAuthMiddleware_CredentialsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		pass     string
		wantConf bool
	}{
		{"both set", "admin", "secret", true},
		{"empty user", "", "secret", false},
		{"empty pass", "admin", "", false},
		{"both empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewBasicAuthMiddleware(tt.user, tt.pass)
			if m.credentialsConfigured != tt.wantConf {
				t.Errorf("credentialsConfigured = %v, want %v", m.credentialsConfigured, tt.wantConf)
			}
			if m.realm != "ScopeConfig" {
				t.Errorf("realm = %q, want %q", m.realm, "ScopeConfig")
			}
		})
	}
}

func TestBasicAuth_PassThroughMode(t *testing.T) {
	m := NewBasicAuthMiddleware("", "")
	handler := m.Handler(okHandler)

	// Without Authorization header
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("pass-through without header: got %d, want %d", w.Code, http.StatusOK)
	}

	// With Authorization header (should still pass through)
	req = httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("anyone", "anything")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("pass-through with header: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestBasicAuth_ValidCredentials(t *testing.T) {
	m := NewBasicAuthMiddleware("admin", "secret")
	handler := m.Handler(okHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("admin", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("valid credentials: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestBasicAuth_InvalidCredentials(t *testing.T) {
	m := NewBasicAuthMiddleware("admin", "secret")
	handler := m.Handler(okHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("admin", "wrong")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid credentials: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if got := w.Header().Get("WWW-Authenticate"); got == "" {
		t.Error("expected WWW-Authenticate header to be set")
	}
}

func TestBasicAuth_MissingHeader(t *testing.T) {
	m := NewBasicAuthMiddleware("admin", "secret")
	handler := m.Handler(okHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing header: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if got := w.Header().Get("WWW-Authenticate"); got == "" {
		t.Error("expected WWW-Authenticate header to be set")
	}
}

func TestBasicAuth_MalformedHeader(t *testing.T) {
	m := NewBasicAuthMiddleware("admin", "secret")
	handler := m.Handler(okHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("malformed header: got %d, want %d", w.Code, http.StatusUnauthorized)
	}

	// Also test with garbage value
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "NotBasic garbage")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("garbage header: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestBasicAuth_ResponseIsJSON(t *testing.T) {
	m := NewBasicAuthMiddleware("admin", "secret")
	handler := m.Handler(okHandler)

	// Missing header
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if errResp.Message == "" {
		t.Error("expected non-empty error message")
	}
	if errResp.Code != http.StatusUnauthorized {
		t.Errorf("error code = %d, want %d", errResp.Code, http.StatusUnauthorized)
	}

	// Wrong credentials
	req = httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("admin", "wrong")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var errResp2 ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp2); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if errResp2.Message == "" {
		t.Error("expected non-empty error message for invalid credentials")
	}
}
