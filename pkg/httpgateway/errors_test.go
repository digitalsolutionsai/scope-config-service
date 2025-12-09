package httpgateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGrpcCodeToHTTP(t *testing.T) {
	tests := []struct {
		name         string
		grpcCode     codes.Code
		expectedHTTP int
	}{
		{"NotFound", codes.NotFound, http.StatusNotFound},
		{"InvalidArgument", codes.InvalidArgument, http.StatusBadRequest},
		{"PermissionDenied", codes.PermissionDenied, http.StatusForbidden},
		{"AlreadyExists", codes.AlreadyExists, http.StatusConflict},
		{"Internal", codes.Internal, http.StatusInternalServerError},
		{"Unavailable", codes.Unavailable, http.StatusServiceUnavailable},
		{"Unauthenticated", codes.Unauthenticated, http.StatusUnauthorized},
		{"Unimplemented", codes.Unimplemented, http.StatusNotImplemented},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := status.Error(tt.grpcCode, "test error")
			httpCode := grpcCodeToHTTP(err)
			if httpCode != tt.expectedHTTP {
				t.Errorf("grpcCodeToHTTP(%v) = %d, want %d", tt.grpcCode, httpCode, tt.expectedHTTP)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "gRPC NotFound error",
			err:            status.Error(codes.NotFound, "config not found"),
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "config not found",
		},
		{
			name:           "gRPC InvalidArgument error",
			err:            status.Error(codes.InvalidArgument, "invalid scope"),
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid scope",
		},
		{
			name:           "generic error",
			err:            errors.New("something went wrong"),
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, tt.err)

			if w.Code != tt.expectedStatus {
				t.Errorf("WriteError status = %d, want %d", w.Code, tt.expectedStatus)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if errResp.Message != tt.expectedMsg {
				t.Errorf("Error message = %q, want %q", errResp.Message, tt.expectedMsg)
			}

			if errResp.Code != tt.expectedStatus {
				t.Errorf("Error code = %d, want %d", errResp.Code, tt.expectedStatus)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	err := WriteJSON(w, http.StatusOK, data)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("WriteJSON status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("Response data = %v, want %v", result, data)
	}
}
