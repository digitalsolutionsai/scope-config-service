-- SQLite-compatible schema for scope-config-service.
-- Equivalent to all PostgreSQL migrations combined.
-- This file is idempotent (uses CREATE TABLE IF NOT EXISTS).

--------------------------------------------------------------------------------
-- CONFIGURATION INSTANCES & VALUES
--------------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS config_version (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_name TEXT NOT NULL,
    scope TEXT NOT NULL CHECK(scope IN ('SYSTEM', 'PROJECT', 'STORE', 'USER')),
    scope_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    latest_version INTEGER NOT NULL DEFAULT 0,
    published_version INTEGER,
    created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    created_by TEXT,
    updated_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    updated_by TEXT,
    UNIQUE (service_name, scope, scope_id, group_id)
);

CREATE TABLE IF NOT EXISTS config_version_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    config_version_id INTEGER NOT NULL REFERENCES config_version(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    created_by TEXT,
    published_by TEXT DEFAULT NULL,
    published_at TEXT DEFAULT NULL,
    UNIQUE (config_version_id, version)
);

CREATE TABLE IF NOT EXISTS config_field (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    config_version_id INTEGER NOT NULL REFERENCES config_version(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    path TEXT NOT NULL,
    value TEXT NOT NULL,
    type TEXT NOT NULL,
    is_active INTEGER DEFAULT 1,
    UNIQUE (config_version_id, version, path)
);

--------------------------------------------------------------------------------
-- CONFIGURATION TEMPLATES
--------------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS config_template (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_name TEXT NOT NULL,
    service_label TEXT NOT NULL,
    group_id TEXT NOT NULL,
    group_label TEXT NOT NULL,
    group_description TEXT,
    sort_order INTEGER DEFAULT 0,
    is_active INTEGER DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    updated_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    created_by TEXT,
    updated_by TEXT,
    UNIQUE (service_name, group_id)
);

CREATE TABLE IF NOT EXISTS config_template_field (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    config_template_id INTEGER NOT NULL REFERENCES config_template(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    label TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    default_value TEXT,
    display_on TEXT,   -- JSON array of scope strings, e.g. '["SYSTEM","PROJECT"]'
    options TEXT,       -- JSON object for value-label pairs
    sort_order INTEGER DEFAULT 0,
    UNIQUE (config_template_id, path)
);

--------------------------------------------------------------------------------
-- INDEXES
--------------------------------------------------------------------------------

CREATE INDEX IF NOT EXISTS idx_config_version_identifier ON config_version (service_name, scope, scope_id, group_id);
CREATE INDEX IF NOT EXISTS idx_config_version_history_version_id ON config_version_history(config_version_id);
CREATE INDEX IF NOT EXISTS idx_config_template_field_template_id ON config_template_field(config_template_id);
CREATE INDEX IF NOT EXISTS idx_config_template_is_active ON config_template(is_active);
CREATE INDEX IF NOT EXISTS idx_config_template_sort_order ON config_template(service_name, sort_order);
CREATE INDEX IF NOT EXISTS idx_config_template_field_sort_order ON config_template_field(config_template_id, sort_order);
