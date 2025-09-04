-- Migration 002: Update NULL location_timezone values to empty strings
-- This migration updates any existing NULL values in the location_timezone column

UPDATE seshujobs
SET location_timezone = ''
WHERE location_timezone IS NULL;

-- Log the migration
DO $$
BEGIN
    RAISE NOTICE 'Updated NULL location_timezone values to empty strings';
END$$;
