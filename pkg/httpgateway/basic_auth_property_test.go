package httpgateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
)

// nonEmptyString generates a non-empty printable ASCII string (1–50 chars).
func nonEmptyString() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{1,50}`)
}

// Property 1: Initialization mode is determined by credential presence.
// For any pair of strings (username, password), credentialsConfigured is true
// iff both are non-empty.
func TestProperty_InitializationMode(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		username := rapid.String().Draw(t, "username")
		password := rapid.String().Draw(t, "password")

		m := NewBasicAuthMiddleware(username, password)

		expected := username != "" && password != ""
		if m.credentialsConfigured != expected {
			t.Fatalf("NewBasicAuthMiddleware(%q, %q).credentialsConfigured = %v, want %v",
				username, password, m.credentialsConfigured, expected)
		}
		if m.realm != "ScopeConfig" {
			t.Fatalf("realm = %q, want %q", m.realm, "ScopeConfig")
		}
	})
}

// Property 2: Pass-through mode forwards all requests.
// For any request with any Authorization header value, pass-through mode
// always invokes the next handler and never returns 401.
func TestProperty_PassThroughForwardsAll(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		authHeader := rapid.String().Draw(t, "authHeader")

		m := NewBasicAuthMiddleware("", "")
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		handler := m.Handler(next)

		req := httptest.NewRequest("GET", "/test", nil)
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if !called {
			t.Fatal("next handler was not called in pass-through mode")
		}
		if w.Code == http.StatusUnauthorized {
			t.Fatal("pass-through mode returned 401")
		}
	})
}

// Property 3: Valid credentials grant access.
// For any non-empty credential pair, a request carrying matching Basic Auth
// always reaches the next handler.
func TestProperty_ValidCredentialsGrantAccess(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		username := nonEmptyString().Draw(t, "username")
		password := nonEmptyString().Draw(t, "password")

		m := NewBasicAuthMiddleware(username, password)
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		handler := m.Handler(next)

		req := httptest.NewRequest("GET", "/test", nil)
		req.SetBasicAuth(username, password)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if !called {
			t.Fatalf("next handler not called with matching credentials (%q, %q)", username, password)
		}
		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want 200 with matching credentials", w.Code)
		}
	})
}

// Property 4: Invalid credentials are rejected.
// For any two distinct credential pairs where at least one field differs,
// the middleware returns 401 with WWW-Authenticate header.
func TestProperty_InvalidCredentialsRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		configUser := nonEmptyString().Draw(t, "configUser")
		configPass := nonEmptyString().Draw(t, "configPass")
		reqUser := nonEmptyString().Draw(t, "reqUser")
		reqPass := nonEmptyString().Draw(t, "reqPass")

		// Ensure at least one field differs
		if configUser == reqUser && configPass == reqPass {
			reqPass = reqPass + "x"
		}

		m := NewBasicAuthMiddleware(configUser, configPass)
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		handler := m.Handler(next)

		req := httptest.NewRequest("GET", "/test", nil)
		req.SetBasicAuth(reqUser, reqPass)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if called {
			t.Fatalf("next handler was called with mismatched credentials (config=%q/%q, req=%q/%q)",
				configUser, configPass, reqUser, reqPass)
		}
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("got status %d, want 401 with mismatched credentials", w.Code)
		}
		if got := w.Header().Get("WWW-Authenticate"); got == "" {
			t.Fatal("expected WWW-Authenticate header to be set")
		}

		// Verify JSON response body
		var errResp ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
			t.Fatalf("response body is not valid JSON: %v", err)
		}
		if errResp.Message == "" {
			t.Fatal("expected non-empty error message in JSON response")
		}
	})
}
