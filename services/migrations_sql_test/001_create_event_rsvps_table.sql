-- Create the event_rsvps table with check constraints
CREATE TABLE event_rsvps (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  event_id UUID NOT NULL,
  event_source_id UUID NOT NULL,
  event_source_type VARCHAR(50) NOT NULL,
  status VARCHAR(50) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  CONSTRAINT event_rsvps_status_check CHECK (status IN ('Yes', 'Maybe', 'Interested, cannot make it!')),
  CONSTRAINT event_rsvps_event_source_type_check CHECK (event_source_type IN ('internalRecurrence', 'internalSingle', 'seshuJob'))
);

-- Create indexes for foreign keys for better query performance
CREATE INDEX event_rsvps_user_id_index ON event_rsvps (user_id);
CREATE INDEX event_rsvps_event_id_index ON event_rsvps (event_id);

