-- Migration 003: Add SCANNING status to scrape_status enum
-- This migration extends the scrape_status enum to include 'SCANNING' for newly created jobs

DO $$
BEGIN
    -- Check if SCANNING value already exists in the enum
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum
        WHERE enumlabel = 'SCANNING'
        AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'scrape_status')
    ) THEN
        -- Add SCANNING to the scrape_status enum
        ALTER TYPE scrape_status ADD VALUE 'SCANNING';
        RAISE NOTICE 'Added SCANNING value to scrape_status enum';
    ELSE
        RAISE NOTICE 'SCANNING value already exists in scrape_status enum';
    END IF;
END$$;
