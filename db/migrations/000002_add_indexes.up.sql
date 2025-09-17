CREATE INDEX IF NOT EXISTS idx_config_version_service_name ON config_version (service_name);
CREATE INDEX IF NOT EXISTS idx_config_version_project_id ON config_version (project_id);
CREATE INDEX IF NOT EXISTS idx_config_version_store_id ON config_version (store_id);
CREATE INDEX IF NOT EXISTS idx_config_version_group_id ON config_version (group_id);
