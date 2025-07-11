-- Create enum only if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'scrape_status') THEN
        CREATE TYPE scrape_status AS ENUM ('HEALTHY', 'WARNING', 'FAILING');
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'scrape_source') THEN
        CREATE TYPE scrape_source AS ENUM ('MEETUP', 'EVENTBRITE', 'other structured');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS seshujobs (
    normalized_url_key TEXT PRIMARY KEY,
    location_latitude DOUBLE PRECISION,
    location_longitude DOUBLE PRECISION,
    location_address TEXT,
    scheduled_scrape_time TIMESTAMP NOT NULL,
    target_name_css_path TEXT NOT NULL,
    target_location_css_path TEXT NOT NULL,
    target_start_time_css_path TEXT NOT NULL,
    target_description_css_path TEXT,
    target_href_css_path TEXT,
    status scrape_status NOT NULL,
    last_scrape_success TIMESTAMP,
    last_scrape_failure TIMESTAMP,
    last_scrape_failure_count INTEGER NOT NULL DEFAULT 0,
    owner_id TEXT NOT NULL,
    known_scrape_source scrape_source NOT NULL
);
