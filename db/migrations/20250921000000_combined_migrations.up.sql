CREATE TYPE scope_enum AS ENUM ('SYSTEM', 'PROJECT', 'STORE', 'USER');

CREATE TABLE IF NOT EXISTS config_version (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(50) NOT NULL,
    scope scope_enum NOT NULL,
    scope_id VARCHAR(50) NOT NULL,
    group_id VARCHAR(50) NOT NULL,
    latest_version INTEGER NOT NULL DEFAULT 0,
    published_version INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(50),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(50),
    UNIQUE (service_name, scope, scope_id, group_id)
);

CREATE TABLE IF NOT EXISTS config_field (
    id SERIAL PRIMARY KEY,
    config_version_id INTEGER NOT NULL REFERENCES config_version(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    path VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    type VARCHAR(20) NOT NULL,
    default_value TEXT,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    UNIQUE (config_version_id, version, path)
);

-- The name clearly states its purpose: to quickly find a configuration by its unique identifier.
CREATE INDEX IF NOT EXISTS idx_config_version_identifier ON config_version (service_name, scope, scope_id, group_id);


CREATE TABLE IF NOT EXISTS config_template (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(255) NOT NULL,
    group_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    UNIQUE (service_name, group_id)
);

CREATE TABLE IF NOT EXISTS config_template_field (
    id SERIAL PRIMARY KEY,
    config_template_id INT NOT NULL REFERENCES config_template(id) ON DELETE CASCADE,
    path VARCHAR(255) NOT NULL,
    label VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- Corresponds to the FieldType enum (STRING, INT, etc.)
    default_value TEXT,
    display_on TEXT[], -- Array of strings corresponding to the Scope enum (STORE, PROJECT, etc.)
    options JSONB, -- Stores value-label pairs for dropdowns
    UNIQUE (config_template_id, path)
);

CREATE INDEX IF NOT EXISTS idx_config_template_field_template_id ON config_template_field(config_template_id);