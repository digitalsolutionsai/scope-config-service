-- Increase size of existing audit fields to support rich identity "Name (Email)"
ALTER TABLE config_version ALTER COLUMN created_by TYPE VARCHAR(255);
ALTER TABLE config_version ALTER COLUMN updated_by TYPE VARCHAR(255);
ALTER TABLE config_version_history ALTER COLUMN created_by TYPE VARCHAR(255);

-- Add publication audit fields to config_version_history table
-- This allows tracking who published each specific version
ALTER TABLE config_version_history 
ADD COLUMN IF NOT EXISTS published_by VARCHAR(255) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ DEFAULT NULL;

-- Backfill: If a version is currently marked as 'published' in config_version, 
-- we can assume it was published by the person who last updated the config_version record.
UPDATE config_version_history h
SET published_at = cv.updated_at, published_by = cv.updated_by
FROM config_version cv
WHERE h.config_version_id = cv.id AND h.version = cv.published_version;
