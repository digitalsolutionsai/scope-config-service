package scopeconfig

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Standard sentinel errors for common failure scenarios.
var (
	// ErrConfigNotFound indicates that the requested configuration was not found.
	ErrConfigNotFound = errors.New("config not found")

	// ErrTemplateNotFound indicates that the requested template was not found.
	ErrTemplateNotFound = errors.New("template not found")

	// ErrServerUnavailable indicates that the server is unavailable.
	ErrServerUnavailable = errors.New("server unavailable")

	// ErrInvalidArgument indicates that an invalid argument was provided.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrPermissionDenied indicates that permission was denied for the operation.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrAlreadyExists indicates that the resource already exists.
	ErrAlreadyExists = errors.New("resource already exists")
)

// ConfigError represents an error that occurred during a configuration operation.
type ConfigError struct {
	// Method is the name of the method that failed.
	Method string
	// Code is the gRPC status code.
	Code codes.Code
	// Message is the error message from the server.
	Message string
	// Err is the underlying error, if any.
	Err error
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Method, e.Message)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Method, e.Err)
	}
	return fmt.Sprintf("%s failed", e.Method)
}

// Unwrap returns the underlying error.
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// Is checks if the error matches a target error.
func (e *ConfigError) Is(target error) bool {
	switch target {
	case ErrConfigNotFound, ErrTemplateNotFound:
		return e.Code == codes.NotFound
	case ErrServerUnavailable:
		return e.Code == codes.Unavailable
	case ErrInvalidArgument:
		return e.Code == codes.InvalidArgument
	case ErrPermissionDenied:
		return e.Code == codes.PermissionDenied
	case ErrAlreadyExists:
		return e.Code == codes.AlreadyExists
	}
	return false
}

// IsNotFound returns true if the error indicates a resource was not found.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrConfigNotFound) || errors.Is(err, ErrTemplateNotFound)
}

// IsServerUnavailable returns true if the error indicates the server is unavailable.
func IsServerUnavailable(err error) bool {
	return errors.Is(err, ErrServerUnavailable)
}

// IsInvalidArgument returns true if the error indicates an invalid argument.
func IsInvalidArgument(err error) bool {
	return errors.Is(err, ErrInvalidArgument)
}

// IsPermissionDenied returns true if the error indicates permission was denied.
func IsPermissionDenied(err error) bool {
	return errors.Is(err, ErrPermissionDenied)
}

// IsAlreadyExists returns true if the error indicates the resource already exists.
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// newConfigError creates a new ConfigError from a gRPC error.
func newConfigError(method string, err error) *ConfigError {
	if err == nil {
		return nil
	}

	configErr := &ConfigError{
		Method: method,
		Err:    err,
	}

	st, ok := status.FromError(err)
	if ok {
		configErr.Code = st.Code()
		configErr.Message = st.Message()
	}

	return configErr
}

// GRPCCode returns the gRPC status code from the error, if available.
func GRPCCode(err error) codes.Code {
	var configErr *ConfigError
	if errors.As(err, &configErr) {
		return configErr.Code
	}
	st, ok := status.FromError(err)
	if ok {
		return st.Code()
	}
	return codes.Unknown
}
