-- Add is_active column to config_template table to allow filtering templates
ALTER TABLE config_template ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true NOT NULL;

-- Create index for efficient filtering by is_active
CREATE INDEX IF NOT EXISTS idx_config_template_is_active ON config_template(is_active);
