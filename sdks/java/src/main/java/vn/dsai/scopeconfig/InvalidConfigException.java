package vn.dsai.scopeconfig;

import io.grpc.Status;

// Exception thrown when a configuration request contains invalid data
public class InvalidConfigException extends ConfigServiceException {

    public InvalidConfigException(String message) {
        super(message, Status.Code.INVALID_ARGUMENT);
    }

    public InvalidConfigException(String message, Throwable cause) {
        super(message, Status.Code.INVALID_ARGUMENT, cause);
    }
}
