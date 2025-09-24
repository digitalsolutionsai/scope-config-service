-- The scope for which a configuration applies.
CREATE TYPE scope_enum AS ENUM ('SYSTEM', 'PROJECT', 'STORE', 'USER');

--------------------------------------------------------------------------------
-- CONFIGURATION INSTANCES & VALUES
--------------------------------------------------------------------------------

-- Tracks the current and published versions for a specific configuration instance.
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

-- This separates the version timeline from the current state in config_version.
CREATE TABLE IF NOT EXISTS config_version_history (
    id SERIAL PRIMARY KEY,
    config_version_id INTEGER NOT NULL REFERENCES config_version(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(50),
    UNIQUE (config_version_id, version)
);

CREATE TABLE IF NOT EXISTS config_field (
    id SERIAL PRIMARY KEY,
    config_version_id INTEGER NOT NULL REFERENCES config_version(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    path VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    type VARCHAR(20) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    UNIQUE (config_version_id, version, path)
);

--------------------------------------------------------------------------------
-- CONFIGURATION TEMPLATES
--------------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS config_template (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(255) NOT NULL,
    service_label VARCHAR(255) NOT NULL, -- Added
    group_id VARCHAR(255) NOT NULL,
    group_label VARCHAR(255) NOT NULL, -- Added
    group_description TEXT, -- Added
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    UNIQUE (service_name, group_id)
);

-- Defines the schema for each field within a configuration template.
CREATE TABLE IF NOT EXISTS config_template_field (
    id SERIAL PRIMARY KEY,
    config_template_id INT NOT NULL REFERENCES config_template(id) ON DELETE CASCADE,
    path VARCHAR(255) NOT NULL,
    label VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- Corresponds to the FieldType enum (STRING, INT, etc.)
    default_value TEXT,
    display_on scope_enum[], -- Changed to scope_enum array for type safety
    options JSONB, -- Stores value-label pairs for dropdowns or enum-like strings
    UNIQUE (config_template_id, path)
);

--------------------------------------------------------------------------------
-- INDEXES
--------------------------------------------------------------------------------

-- For quick lookups of a configuration instance.
CREATE INDEX IF NOT EXISTS idx_config_version_identifier ON config_version (service_name, scope, scope_id, group_id);

-- (NEW) For efficient querying of a version's history.
CREATE INDEX IF NOT EXISTS idx_config_version_history_version_id ON config_version_history(config_version_id);

-- For quick retrieval of all fields belonging to a template.
CREATE INDEX IF NOT EXISTS idx_config_template_field_template_id ON config_template_field(config_template_id);