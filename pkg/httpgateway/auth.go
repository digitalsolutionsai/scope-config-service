package httpgateway

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// UserInfo contains the authenticated user information extracted from JWT.
type UserInfo struct {
	Email            string
	PreferredUsername string
	Name             string
	GivenName        string
	FamilyName       string
	Sub              string
	Roles            []string
}

// contextKey is used for storing values in context.
type contextKey string

const (
	userInfoKey contextKey = "userInfo"
)

// KeycloakClaims represents the claims from a Keycloak JWT token.
type KeycloakClaims struct {
	jwt.RegisteredClaims
	PreferredUsername string                            `json:"preferred_username"`
	Email             string                            `json:"email"`
	EmailVerified     bool                              `json:"email_verified"`
	Name              string                            `json:"name"`
	GivenName         string                            `json:"given_name"`
	FamilyName        string                            `json:"family_name"`
	ResourceAccess    map[string]map[string]interface{} `json:"resource_access"`
}

// AuthMiddleware validates JWT tokens and checks for required roles.
type AuthMiddleware struct {
	publicKey       *rsa.PublicKey
	requiredClient  string   // e.g., "dsai-console"
	requiredRoles   []string // e.g., ["ROLE_ADMIN"]
	skipAuth        bool     // For development/testing
}

// NewAuthMiddleware creates a new authentication middleware.
// publicKeyPEM should be the PEM-encoded RSA public key from Keycloak.
func NewAuthMiddleware(publicKeyPEM, requiredClient string, requiredRoles []string) (*AuthMiddleware, error) {
	// If no public key provided, skip auth (for development)
	if publicKeyPEM == "" {
		return &AuthMiddleware{
			skipAuth: true,
		}, nil
	}

	publicKey, err := parseRSAPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return &AuthMiddleware{
		publicKey:      publicKey,
		requiredClient: requiredClient,
		requiredRoles:  requiredRoles,
		skipAuth:       false,
	}, nil
}

// Middleware returns an HTTP middleware function that validates JWT tokens.
func (a *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication if configured
		if a.skipAuth {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			WriteError(w, &authError{message: "missing Authorization header"})
			return
		}

		// Check for Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			WriteError(w, &authError{message: "invalid Authorization header format"})
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		claims, err := a.validateToken(tokenString)
		if err != nil {
			WriteError(w, &authError{message: fmt.Sprintf("invalid token: %v", err)})
			return
		}

		// Check roles
		if !a.hasRequiredRole(claims) {
			WriteError(w, &authError{message: "insufficient permissions"})
			return
		}

		// Extract user info and add to context
		userInfo := a.extractUserInfo(claims)
		ctx := context.WithValue(r.Context(), userInfoKey, userInfo)

		// Continue to next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateToken parses and validates the JWT token.
func (a *AuthMiddleware) validateToken(tokenString string) (*KeycloakClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &KeycloakClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	claims, ok := token.Claims.(*KeycloakClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// hasRequiredRole checks if the user has the required role in the required client.
func (a *AuthMiddleware) hasRequiredRole(claims *KeycloakClaims) bool {
	if claims.ResourceAccess == nil {
		return false
	}

	clientAccess, ok := claims.ResourceAccess[a.requiredClient]
	if !ok {
		return false
	}

	rolesInterface, ok := clientAccess["roles"]
	if !ok {
		return false
	}

	roles, ok := rolesInterface.([]interface{})
	if !ok {
		return false
	}

	// Check if user has any of the required roles
	for _, role := range roles {
		roleStr, ok := role.(string)
		if !ok {
			continue
		}

		for _, requiredRole := range a.requiredRoles {
			if roleStr == requiredRole {
				return true
			}
		}
	}

	return false
}

// extractUserInfo extracts user information from JWT claims.
func (a *AuthMiddleware) extractUserInfo(claims *KeycloakClaims) *UserInfo {
	// Extract all roles from the required client
	var roles []string
	if claims.ResourceAccess != nil {
		if clientAccess, ok := claims.ResourceAccess[a.requiredClient]; ok {
			if rolesInterface, ok := clientAccess["roles"].([]interface{}); ok {
				for _, role := range rolesInterface {
					if roleStr, ok := role.(string); ok {
						roles = append(roles, roleStr)
					}
				}
			}
		}
	}

	return &UserInfo{
		Email:             claims.Email,
		PreferredUsername: claims.PreferredUsername,
		Name:              claims.Name,
		GivenName:         claims.GivenName,
		FamilyName:        claims.FamilyName,
		Sub:               claims.Subject,
		Roles:             roles,
	}
}

// GetUserInfo extracts user information from the request context.
func GetUserInfo(ctx context.Context) (*UserInfo, bool) {
	userInfo, ok := ctx.Value(userInfoKey).(*UserInfo)
	return userInfo, ok
}

// parseRSAPublicKeyFromPEM parses an RSA public key from PEM format.
func parseRSAPublicKeyFromPEM(pemStr string) (*rsa.PublicKey, error) {
	// Try PEM format first
	block, _ := pem.Decode([]byte(pemStr))
	if block != nil {
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA public key")
		}
		return rsaPub, nil
	}

	// Try base64-encoded DER format (Keycloak format)
	der, err := base64.StdEncoding.DecodeString(pemStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

// authError represents an authentication error.
type authError struct {
	message string
}

func (e *authError) Error() string {
	return e.message
}

// WriteError is extended to handle auth errors with 401 status.
func (e *authError) HTTPStatus() int {
	return http.StatusUnauthorized
}

// Helper function to get user email from context (for audit purposes).
func GetUserEmail(ctx context.Context) string {
	userInfo, ok := GetUserInfo(ctx)
	if !ok || userInfo == nil {
		return ""
	}
	if userInfo.Email != "" {
		return userInfo.Email
	}
	return userInfo.PreferredUsername
}

// ParseKeycloakToken is a utility function to parse and print token claims (for debugging).
func ParseKeycloakToken(tokenString string) (*KeycloakClaims, error) {
	// Parse without validation (for debugging)
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &KeycloakClaims{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*KeycloakClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

// PrintTokenClaims prints the token claims in a formatted way (for debugging).
func PrintTokenClaims(claims *KeycloakClaims) {
	jsonBytes, _ := json.MarshalIndent(claims, "", "  ")
	fmt.Println(string(jsonBytes))
}
