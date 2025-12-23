package vn.dsai.scopeconfig;

import vn.dsai.config.v1.ConfigField;
import vn.dsai.config.v1.FieldType;

/**
 * Utility builder for creating ConfigField objects with proper type handling.
 *
 * Example:
 * <pre>{@code
 * ConfigField field = ConfigFieldBuilder.create("database.host")
 *         .stringValue("localhost")
 *         .build();
 *
 * ConfigField intField = ConfigFieldBuilder.create("database.port")
 *         .intValue(5432)
 *         .build();
 * }</pre>
 */
public class ConfigFieldBuilder {

    private final ConfigField.Builder builder;

    private ConfigFieldBuilder(String path) {
        this.builder = ConfigField.newBuilder().setPath(path);
    }

    // Creates a new builder with the specified field path (e.g., "database.host")
    public static ConfigFieldBuilder create(String path) {
        if (path == null || path.isEmpty()) {
            throw new IllegalArgumentException("Field path is required");
        }
        return new ConfigFieldBuilder(path);
    }

    // Sets a string value
    public ConfigFieldBuilder stringValue(String value) {
        builder.setValue(value).setType(FieldType.STRING);
        return this;
    }

    // Sets an integer value
    public ConfigFieldBuilder intValue(int value) {
        builder.setValue(String.valueOf(value)).setType(FieldType.INT);
        return this;
    }

    // Sets a long integer value
    public ConfigFieldBuilder longValue(long value) {
        builder.setValue(String.valueOf(value)).setType(FieldType.INT);
        return this;
    }

    // Sets a float value
    public ConfigFieldBuilder floatValue(float value) {
        builder.setValue(String.valueOf(value)).setType(FieldType.FLOAT);
        return this;
    }

    // Sets a double value
    public ConfigFieldBuilder doubleValue(double value) {
        builder.setValue(String.valueOf(value)).setType(FieldType.FLOAT);
        return this;
    }

    // Sets a boolean value
    public ConfigFieldBuilder booleanValue(boolean value) {
        builder.setValue(String.valueOf(value)).setType(FieldType.BOOLEAN);
        return this;
    }

    // Sets a JSON value
    public ConfigFieldBuilder jsonValue(String jsonValue) {
        builder.setValue(jsonValue).setType(FieldType.JSON);
        return this;
    }

    // Sets a raw value with explicit type
    public ConfigFieldBuilder value(String value, FieldType type) {
        builder.setValue(value).setType(type);
        return this;
    }

    // Builds and returns the ConfigField
    public ConfigField build() {
        return builder.build();
    }
}
