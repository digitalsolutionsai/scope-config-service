package httpgateway

import (
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorResponse represents an error response in JSON format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// WriteError writes an error response in JSON format.
func WriteError(w http.ResponseWriter, err error) {
	httpStatus := grpcCodeToHTTP(err)
	grpcStatus, ok := status.FromError(err)
	
	var message string
	if ok {
		message = grpcStatus.Message()
	} else {
		message = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(httpStatus),
		Message: message,
		Code:    httpStatus,
	})
}

// grpcCodeToHTTP converts gRPC error codes to HTTP status codes.
func grpcCodeToHTTP(err error) int {
	grpcStatus, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError
	}

	switch grpcStatus.Code() {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}
