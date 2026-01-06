package vn.dsai.scopeconfig;

import io.grpc.Status;
import io.grpc.StatusRuntimeException;

// Base exception for all ConfigService-related errors
public class ConfigServiceException extends Exception {

    private final Status.Code statusCode;

    public ConfigServiceException(String message, Status.Code statusCode) {
        super(message);
        this.statusCode = statusCode;
    }

    public ConfigServiceException(String message, Status.Code statusCode, Throwable cause) {
        super(message, cause);
        this.statusCode = statusCode;
    }

    public Status.Code getStatusCode() {
        return statusCode;
    }

    // Creates a ConfigServiceException from a gRPC StatusRuntimeException
    public static ConfigServiceException fromGrpcStatus(String method, StatusRuntimeException e) {
        Status status = e.getStatus();
        String message = status.getDescription() != null ? status.getDescription() : e.getMessage();

        return switch (status.getCode()) {
            case NOT_FOUND -> new ConfigNotFoundException(
                    String.format("%s: resource not found - %s", method, message),
                    e
            );
            case INVALID_ARGUMENT -> new InvalidConfigException(
                    String.format("%s: invalid argument - %s", method, message),
                    e
            );
            case ALREADY_EXISTS -> new ConfigServiceException(
                    String.format("%s: resource already exists - %s", method, message),
                    Status.Code.ALREADY_EXISTS,
                    e
            );
            case PERMISSION_DENIED -> new ConfigServiceException(
                    String.format("%s: permission denied - %s", method, message),
                    Status.Code.PERMISSION_DENIED,
                    e
            );
            case UNAVAILABLE -> new ConfigServiceException(
                    String.format("%s: service unavailable - %s", method, message),
                    Status.Code.UNAVAILABLE,
                    e
            );
            case UNAUTHENTICATED -> new ConfigServiceException(
                    String.format("%s: authentication required - %s", method, message),
                    Status.Code.UNAUTHENTICATED,
                    e
            );
            default -> new ConfigServiceException(
                    String.format("%s failed with %s: %s", method, status.getCode(), message),
                    status.getCode(),
                    e
            );
        };
    }
}
