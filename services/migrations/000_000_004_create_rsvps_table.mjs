import { Kysely, sql } from 'kysely';

export async function up(db) {
  // Create the event_rsvps table
  await db.schema
    .createTable("event_rsvps")
    .addColumn("id", "uuid", (col) => col.primaryKey())
    .addColumn("user_id", "uuid", (col) => col.notNull())
    .addColumn("event_id", "uuid", (col) => col.notNull())
    .addColumn("event_source_id", "uuid", (col) => col.notNull())
    .addColumn("event_source_type", "varchar(50)", (col) => col.notNull()) // Assuming this is an enum type
    .addColumn("status", "varchar(50)", (col) => col.notNull()) // Enum values: 'Yes', 'Maybe', 'Interested, can't make it!'
    .addColumn("created_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .execute();

  // Create indexes for foreign keys for better query performance
  await db.schema
    .createIndex("event_rsvps_user_id_index")
    .on("event_rsvps")
    .column("user_id")
    .execute();

  await db.schema
    .createIndex("event_rsvps_event_id_index")
    .on("event_rsvps")
    .column("event_id")
    .execute();

  await db.schema
    .createIndex("event_rsvps_event_source_id_index")
    .on("event_rsvps")
    .column("event_source_id")
    .execute();

  // Optionally, add a check constraint for status to mimic enum behavior
  await db.schema
    .alterTable("event_rsvps")
    .addConstraint(
      "event_rsvps_status_check",
      sql`CHECK (status IN ('Yes', 'Maybe', 'Interested, can't make it!'))`
    )
    .execute();

  // Optionally, add a check constraint for event_source_type to mimic enum behavior
  await db.schema
    .alterTable("event_rsvps")
    .addConstraint(
      "event_rsvps_event_source_type_check",
      sql`CHECK (event_source_type IN ('Type1', 'Type2', 'Type3'))` // Replace with actual types
    )
    .execute();
}

export async function down(db) {
  // Drop the indexes and constraints before dropping the table
  await db.schema.dropIndex("event_rsvps_user_id_index").execute();
  await db.schema.dropIndex("event_rsvps_event_id_index").execute();
  await db.schema.dropIndex("event_rsvps_event_source_id_index").execute();
  await db.schema.dropConstraint("event_rsvps_status_check").execute();
  await db.schema.dropConstraint("event_rsvps_event_source_type_check").execute();

  // Drop the event_rsvps table
  await db.schema.dropTable("event_rsvps").execute();
}

