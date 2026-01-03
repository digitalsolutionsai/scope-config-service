package scopeconfig

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestConfigError(t *testing.T) {
	err := &ConfigError{
		Method:  "GetConfig",
		Code:    codes.NotFound,
		Message: "configuration not found",
	}

	if err.Error() != "GetConfig: configuration not found" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestConfigErrorWithUnderlying(t *testing.T) {
	underlying := errors.New("connection failed")
	err := &ConfigError{
		Method: "GetConfig",
		Err:    underlying,
	}

	if err.Unwrap() != underlying {
		t.Error("Expected to unwrap to underlying error")
	}
}

func TestConfigErrorIs(t *testing.T) {
	tests := []struct {
		name   string
		code   codes.Code
		target error
		want   bool
	}{
		{"NotFound matches ErrConfigNotFound", codes.NotFound, ErrConfigNotFound, true},
		{"NotFound matches ErrTemplateNotFound", codes.NotFound, ErrTemplateNotFound, true},
		{"Unavailable matches ErrServerUnavailable", codes.Unavailable, ErrServerUnavailable, true},
		{"InvalidArgument matches ErrInvalidArgument", codes.InvalidArgument, ErrInvalidArgument, true},
		{"PermissionDenied matches ErrPermissionDenied", codes.PermissionDenied, ErrPermissionDenied, true},
		{"AlreadyExists matches ErrAlreadyExists", codes.AlreadyExists, ErrAlreadyExists, true},
		{"NotFound does not match ErrServerUnavailable", codes.NotFound, ErrServerUnavailable, false},
		{"Unknown does not match any", codes.Unknown, ErrConfigNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ConfigError{Code: tt.code}
			if errors.Is(err, tt.target) != tt.want {
				t.Errorf("ConfigError{Code: %v}.Is(%v) = %v, want %v",
					tt.code, tt.target, errors.Is(err, tt.target), tt.want)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	notFoundErr := &ConfigError{Code: codes.NotFound}
	unavailableErr := &ConfigError{Code: codes.Unavailable}

	if !IsNotFound(notFoundErr) {
		t.Error("Expected IsNotFound to return true for NotFound error")
	}
	if IsNotFound(unavailableErr) {
		t.Error("Expected IsNotFound to return false for Unavailable error")
	}
}

func TestIsServerUnavailable(t *testing.T) {
	unavailableErr := &ConfigError{Code: codes.Unavailable}
	notFoundErr := &ConfigError{Code: codes.NotFound}

	if !IsServerUnavailable(unavailableErr) {
		t.Error("Expected IsServerUnavailable to return true for Unavailable error")
	}
	if IsServerUnavailable(notFoundErr) {
		t.Error("Expected IsServerUnavailable to return false for NotFound error")
	}
}

func TestIsInvalidArgument(t *testing.T) {
	invalidErr := &ConfigError{Code: codes.InvalidArgument}
	otherErr := &ConfigError{Code: codes.NotFound}

	if !IsInvalidArgument(invalidErr) {
		t.Error("Expected IsInvalidArgument to return true")
	}
	if IsInvalidArgument(otherErr) {
		t.Error("Expected IsInvalidArgument to return false")
	}
}

func TestIsPermissionDenied(t *testing.T) {
	deniedErr := &ConfigError{Code: codes.PermissionDenied}
	otherErr := &ConfigError{Code: codes.NotFound}

	if !IsPermissionDenied(deniedErr) {
		t.Error("Expected IsPermissionDenied to return true")
	}
	if IsPermissionDenied(otherErr) {
		t.Error("Expected IsPermissionDenied to return false")
	}
}

func TestIsAlreadyExists(t *testing.T) {
	existsErr := &ConfigError{Code: codes.AlreadyExists}
	otherErr := &ConfigError{Code: codes.NotFound}

	if !IsAlreadyExists(existsErr) {
		t.Error("Expected IsAlreadyExists to return true")
	}
	if IsAlreadyExists(otherErr) {
		t.Error("Expected IsAlreadyExists to return false")
	}
}

func TestNewConfigError(t *testing.T) {
	// Test with nil error
	if newConfigError("Test", nil) != nil {
		t.Error("Expected nil result for nil error")
	}

	// Test with gRPC error
	grpcErr := status.Error(codes.NotFound, "not found")
	configErr := newConfigError("GetConfig", grpcErr)

	if configErr == nil {
		t.Fatal("Expected non-nil config error")
	}
	if configErr.Method != "GetConfig" {
		t.Errorf("Expected method 'GetConfig', got '%s'", configErr.Method)
	}
	if configErr.Code != codes.NotFound {
		t.Errorf("Expected code NotFound, got %v", configErr.Code)
	}
	if configErr.Message != "not found" {
		t.Errorf("Expected message 'not found', got '%s'", configErr.Message)
	}

	// Test with non-gRPC error
	plainErr := errors.New("plain error")
	configErr = newConfigError("Update", plainErr)

	if configErr.Code != codes.OK { // codes.OK is the zero value
		t.Errorf("Expected zero code for non-gRPC error, got %v", configErr.Code)
	}
}

func TestGRPCCode(t *testing.T) {
	// Test with ConfigError
	configErr := &ConfigError{Code: codes.InvalidArgument}
	if GRPCCode(configErr) != codes.InvalidArgument {
		t.Error("Expected InvalidArgument code")
	}

	// Test with gRPC status error
	grpcErr := status.Error(codes.Unavailable, "unavailable")
	if GRPCCode(grpcErr) != codes.Unavailable {
		t.Error("Expected Unavailable code")
	}

	// Test with plain error
	plainErr := errors.New("plain error")
	if GRPCCode(plainErr) != codes.Unknown {
		t.Error("Expected Unknown code for plain error")
	}
}

func TestSentinelErrors(t *testing.T) {
	// Verify sentinel errors are distinct
	errors := []error{
		ErrConfigNotFound,
		ErrTemplateNotFound,
		ErrServerUnavailable,
		ErrInvalidArgument,
		ErrPermissionDenied,
		ErrAlreadyExists,
	}

	for i, e1 := range errors {
		for j, e2 := range errors {
			if i != j && e1 == e2 {
				t.Errorf("Sentinel errors %d and %d should be distinct", i, j)
			}
		}
	}
}
