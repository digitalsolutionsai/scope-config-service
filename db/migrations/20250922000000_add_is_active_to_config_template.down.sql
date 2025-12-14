-- Remove index
DROP INDEX IF EXISTS idx_config_template_is_active;

-- Remove is_active column from config_template table
ALTER TABLE config_template DROP COLUMN IF EXISTS is_active;
