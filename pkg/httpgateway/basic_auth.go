package httpgateway

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
)

// BasicAuthMiddleware provides HTTP Basic Authentication.
type BasicAuthMiddleware struct {
	username              string
	password              string
	credentialsConfigured bool   // false when AUTH_USER or AUTH_PASSWORD is empty
	realm                 string // WWW-Authenticate realm, default "ScopeConfig"
}

// NewBasicAuthMiddleware creates a BasicAuthMiddleware.
// If username or password is empty, credentialsConfigured is set to false
// and the middleware will pass through all requests (internal/gateway mode).
func NewBasicAuthMiddleware(username, password string) *BasicAuthMiddleware {
	return &BasicAuthMiddleware{
		username:              username,
		password:              password,
		credentialsConfigured: username != "" && password != "",
		realm:                 "ScopeConfig",
	}
}

// Handler returns an http.Handler middleware function compatible with chi.Use().
func (b *BasicAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pass-through if credentials not configured (internal/gateway mode)
		if !b.credentialsConfigured {
			next.ServeHTTP(w, r)
			return
		}

		// Extract Basic Auth from request
		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, b.realm))
			writeBasicAuthError(w, "missing or invalid Authorization header")
			return
		}

		// Constant-time comparison to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(b.username)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(b.password)) == 1

		if !usernameMatch || !passwordMatch {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, b.realm))
			writeBasicAuthError(w, "invalid credentials")
			return
		}

		// Credentials valid — proceed
		next.ServeHTTP(w, r)
	})
}

// writeBasicAuthError writes a JSON error response with HTTP 401 status.
func writeBasicAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(http.StatusUnauthorized),
		Message: message,
		Code:    http.StatusUnauthorized,
	})
}
