-- Migration 002: Add child selector columns and recursion flag to seshujobs table
-- This migration introduces optional child CSS selector fields and a recursion flag to match the updated SeshuJob struct.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'seshujobs'
        AND column_name = 'target_child_name_css_path'
    ) THEN
        ALTER TABLE seshujobs ADD COLUMN target_child_name_css_path TEXT;
        RAISE NOTICE 'Added target_child_name_css_path column to seshujobs table';
    ELSE
        RAISE NOTICE 'Column target_child_name_css_path already exists in seshujobs table';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'seshujobs'
        AND column_name = 'target_child_location_css_path'
    ) THEN
        ALTER TABLE seshujobs ADD COLUMN target_child_location_css_path TEXT;
        RAISE NOTICE 'Added target_child_location_css_path column to seshujobs table';
    ELSE
        RAISE NOTICE 'Column target_child_location_css_path already exists in seshujobs table';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'seshujobs'
        AND column_name = 'target_child_start_time_css_path'
    ) THEN
        ALTER TABLE seshujobs ADD COLUMN target_child_start_time_css_path TEXT;
        RAISE NOTICE 'Added target_child_start_time_css_path column to seshujobs table';
    ELSE
        RAISE NOTICE 'Column target_child_start_time_css_path already exists in seshujobs table';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'seshujobs'
        AND column_name = 'target_child_end_time_css_path'
    ) THEN
        ALTER TABLE seshujobs ADD COLUMN target_child_end_time_css_path TEXT;
        RAISE NOTICE 'Added target_child_end_time_css_path column to seshujobs table';
    ELSE
        RAISE NOTICE 'Column target_child_end_time_css_path already exists in seshujobs table';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'seshujobs'
        AND column_name = 'target_child_description_css_path'
    ) THEN
        ALTER TABLE seshujobs ADD COLUMN target_child_description_css_path TEXT;
        RAISE NOTICE 'Added target_child_description_css_path column to seshujobs table';
    ELSE
        RAISE NOTICE 'Column target_child_description_css_path already exists in seshujobs table';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'seshujobs'
        AND column_name = 'is_recursive'
    ) THEN
        ALTER TABLE seshujobs ADD COLUMN is_recursive BOOLEAN NOT NULL DEFAULT FALSE;
        RAISE NOTICE 'Added is_recursive column to seshujobs table';
    ELSE
        RAISE NOTICE 'Column is_recursive already exists in seshujobs table';
    END IF;
END$$;
