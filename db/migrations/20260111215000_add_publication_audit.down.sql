-- Revert size of existing audit fields
ALTER TABLE config_version ALTER COLUMN created_by TYPE VARCHAR(50);
ALTER TABLE config_version ALTER COLUMN updated_by TYPE VARCHAR(50);
ALTER TABLE config_version_history ALTER COLUMN created_by TYPE VARCHAR(50);

-- Remove publication audit fields from config_version_history table
ALTER TABLE config_version_history 
DROP COLUMN IF EXISTS published_by,
DROP COLUMN IF EXISTS published_at;
