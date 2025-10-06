-- Migration 001: Add location_timezone column to seshujobs table
-- This migration adds the new location_timezone column for storing derived timezone information
-- Table name is dynamic and read from app.seshu_jobs_table_name session variable

DO $$
DECLARE
    table_name_var TEXT;
BEGIN
    -- Get table name from session variable (set by Go migration runner)
    table_name_var := current_setting('app.seshu_jobs_table_name', true);

    -- Fallback to default if not set
    IF table_name_var IS NULL OR table_name_var = '' THEN
        table_name_var := 'seshujobs';
    END IF;

    -- Check if the column already exists to make this migration idempotent
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = table_name_var
        AND column_name = 'location_timezone'
    ) THEN
        -- Add the new column using dynamic SQL
        EXECUTE format('ALTER TABLE %I ADD COLUMN location_timezone TEXT', table_name_var);

        -- Log the migration
        RAISE NOTICE 'Added location_timezone column to % table', table_name_var;
    ELSE
        RAISE NOTICE 'Column location_timezone already exists in % table', table_name_var;
    END IF;
END$$;
