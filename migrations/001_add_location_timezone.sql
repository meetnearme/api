-- Migration 001: Add location_timezone column to seshujobs table
-- This migration adds the new location_timezone column for storing derived timezone information

-- Check if the column already exists to make this migration idempotent
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'seshujobs'
        AND column_name = 'location_timezone'
    ) THEN
        -- Add the new column
        ALTER TABLE seshujobs ADD COLUMN location_timezone TEXT;

        -- Log the migration
        RAISE NOTICE 'Added location_timezone column to seshujobs table';
    ELSE
        RAISE NOTICE 'Column location_timezone already exists in seshujobs table';
    END IF;
END$$;
