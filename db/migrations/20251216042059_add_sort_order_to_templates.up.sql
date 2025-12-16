-- Add sort_order column to config_template table for ordering groups
ALTER TABLE config_template ADD COLUMN IF NOT EXISTS sort_order INTEGER DEFAULT 0;

-- Add sort_order column to config_template_field table for ordering fields within a group
ALTER TABLE config_template_field ADD COLUMN IF NOT EXISTS sort_order INTEGER DEFAULT 0;

-- Create indexes for efficient ordering queries
CREATE INDEX IF NOT EXISTS idx_config_template_sort_order ON config_template(service_name, sort_order);
CREATE INDEX IF NOT EXISTS idx_config_template_field_sort_order ON config_template_field(config_template_id, sort_order);
