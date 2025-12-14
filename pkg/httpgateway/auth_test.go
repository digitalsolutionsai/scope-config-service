package httpgateway

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// generateRSAKeyPair generates an RSA key pair for testing.
func generateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

// generateTestToken creates a test JWT token.
func generateTestToken(privateKey *rsa.PrivateKey, claims *KeycloakClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

// publicKeyToPEM converts an RSA public key to PEM format.
func publicKeyToPEM(publicKey *rsa.PublicKey) (string, error) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}

	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}

	return string(pem.EncodeToMemory(pemBlock)), nil
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	// Generate test keys
	privateKey, publicKey, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	publicKeyPEM, err := publicKeyToPEM(publicKey)
	if err != nil {
		t.Fatalf("Failed to convert public key to PEM: %v", err)
	}

	// Create auth middleware
	authMiddleware, err := NewAuthMiddleware(publicKeyPEM, "dsai-console", []string{"ROLE_ADMIN"})
	if err != nil {
		t.Fatalf("Failed to create auth middleware: %v", err)
	}

	// Create test claims with valid role
	claims := &KeycloakClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "https://auth.dsai.vn/realms/sso",
		},
		Email:             "test@example.com",
		PreferredUsername: "testuser",
		Name:              "Test User",
		ResourceAccess: map[string]map[string]interface{}{
			"dsai-console": {
				"roles": []interface{}{"ROLE_ADMIN"},
			},
		},
	}

	// Generate token
	tokenString, err := generateTestToken(privateKey, claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify user info is in context
		userInfo, ok := GetUserInfo(r.Context())
		if !ok {
			t.Error("User info not found in context")
		}
		if userInfo.Email != "test@example.com" {
			t.Errorf("Email = %s, want test@example.com", userInfo.Email)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with auth middleware
	protectedHandler := authMiddleware.Middleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	// Execute request
	protectedHandler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	_, publicKey, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	publicKeyPEM, err := publicKeyToPEM(publicKey)
	if err != nil {
		t.Fatalf("Failed to convert public key to PEM: %v", err)
	}

	authMiddleware, err := NewAuthMiddleware(publicKeyPEM, "dsai-console", []string{"ROLE_ADMIN"})
	if err != nil {
		t.Fatalf("Failed to create auth middleware: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protectedHandler := authMiddleware.Middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	// No Authorization header
	w := httptest.NewRecorder()

	protectedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	_, publicKey, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	publicKeyPEM, err := publicKeyToPEM(publicKey)
	if err != nil {
		t.Fatalf("Failed to convert public key to PEM: %v", err)
	}

	authMiddleware, err := NewAuthMiddleware(publicKeyPEM, "dsai-console", []string{"ROLE_ADMIN"})
	if err != nil {
		t.Fatalf("Failed to create auth middleware: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protectedHandler := authMiddleware.Middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	protectedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InsufficientPermissions(t *testing.T) {
	privateKey, publicKey, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	publicKeyPEM, err := publicKeyToPEM(publicKey)
	if err != nil {
		t.Fatalf("Failed to convert public key to PEM: %v", err)
	}

	authMiddleware, err := NewAuthMiddleware(publicKeyPEM, "dsai-console", []string{"ROLE_ADMIN"})
	if err != nil {
		t.Fatalf("Failed to create auth middleware: %v", err)
	}

	// Create claims with wrong role
	claims := &KeycloakClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Email: "test@example.com",
		ResourceAccess: map[string]map[string]interface{}{
			"dsai-console": {
				"roles": []interface{}{"ROLE_USER"}, // Wrong role
			},
		},
	}

	tokenString, err := generateTestToken(privateKey, claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protectedHandler := authMiddleware.Middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	protectedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_SkipAuth(t *testing.T) {
	// Create middleware with no public key (skip auth mode)
	authMiddleware, err := NewAuthMiddleware("", "dsai-console", []string{"ROLE_ADMIN"})
	if err != nil {
		t.Fatalf("Failed to create auth middleware: %v", err)
	}

	if !authMiddleware.skipAuth {
		t.Error("Expected skipAuth to be true when no public key provided")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protectedHandler := authMiddleware.Middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	// No Authorization header
	w := httptest.NewRecorder()

	protectedHandler.ServeHTTP(w, req)

	// Should pass without authentication
	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d (auth should be skipped)", w.Code, http.StatusOK)
	}
}

func TestGetUserEmail(t *testing.T) {
	privateKey, publicKey, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	publicKeyPEM, err := publicKeyToPEM(publicKey)
	if err != nil {
		t.Fatalf("Failed to convert public key to PEM: %v", err)
	}

	authMiddleware, err := NewAuthMiddleware(publicKeyPEM, "dsai-console", []string{"ROLE_ADMIN"})
	if err != nil {
		t.Fatalf("Failed to create auth middleware: %v", err)
	}

	claims := &KeycloakClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Email:             "loivo@dsai.vn",
		PreferredUsername: "loivo",
		Name:              "Loi Vo",
		ResourceAccess: map[string]map[string]interface{}{
			"dsai-console": {
				"roles": []interface{}{"ROLE_ADMIN"},
			},
		},
	}

	tokenString, err := generateTestToken(privateKey, claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email := GetUserEmail(r.Context())
		if email != "loivo@dsai.vn" {
			t.Errorf("GetUserEmail() = %s, want loivo@dsai.vn", email)
		}
		w.WriteHeader(http.StatusOK)
	})

	protectedHandler := authMiddleware.Middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	protectedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}
