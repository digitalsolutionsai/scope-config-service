CREATE TABLE IF NOT EXISTS config_version (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(50) NOT NULL,
    project_id VARCHAR(50),
    store_id VARCHAR(50),
    group_id VARCHAR(50),
    scope VARCHAR(20) NOT NULL,
    latest_version INTEGER NOT NULL DEFAULT 0,
    published_version INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(50),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(50),
    UNIQUE (service_name, project_id, store_id, group_id, scope)
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