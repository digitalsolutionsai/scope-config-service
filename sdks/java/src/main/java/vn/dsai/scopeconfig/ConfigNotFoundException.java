package vn.dsai.scopeconfig;

import io.grpc.Status;

// Exception thrown when a requested configuration is not found
public class ConfigNotFoundException extends ConfigServiceException {

    public ConfigNotFoundException(String message) {
        super(message, Status.Code.NOT_FOUND);
    }

    public ConfigNotFoundException(String message, Throwable cause) {
        super(message, Status.Code.NOT_FOUND, cause);
    }
}
