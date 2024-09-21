-- Drop the indexes and constraints before dropping the table
DROP INDEX IF EXISTS event_rsvps_user_id_index;
DROP INDEX IF EXISTS event_rsvps_event_id_index;

-- Drop the event_rsvps table
DROP TABLE IF EXISTS event_rsvps;

