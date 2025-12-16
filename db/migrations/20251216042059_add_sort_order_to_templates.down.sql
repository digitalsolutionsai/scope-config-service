-- Remove indexes
DROP INDEX IF EXISTS idx_config_template_field_sort_order;
DROP INDEX IF EXISTS idx_config_template_sort_order;

-- Remove sort_order column from config_template_field table
ALTER TABLE config_template_field DROP COLUMN IF EXISTS sort_order;

-- Remove sort_order column from config_template table
ALTER TABLE config_template DROP COLUMN IF EXISTS sort_order;
