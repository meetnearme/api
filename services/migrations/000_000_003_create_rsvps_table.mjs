import { Kysely, sql } from 'kysely';

export async function up(db) {
  // Create the event_rsvps table with check constraints
  await db.schema
    .createTable("event_rsvps")
    .addColumn("id", "uuid", (col) => col.primaryKey())
    .addColumn("user_id", "uuid", (col) => col.notNull())
    .addColumn("event_id", "uuid", (col) => col.notNull())
    .addColumn("event_source_id", "uuid", (col) => col.notNull())
    .addColumn("event_source_type", "varchar(50)", (col) => col.notNull())
    .addColumn("status", "varchar(50)", (col) => col.notNull())
    .addColumn("created_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn("updated_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addCheckConstraint(
      "event_rsvps_status_check",
      sql`status IN ('Yes', 'Maybe', 'Interested, cannot make it!')`
    )
    .addCheckConstraint(
      "event_rsvps_event_source_type_check",
      sql`event_source_type IN ('internalRecurrence', 'internalSingle', 'seshuJob')` // Replace with actual types
    )
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

  // await db.schema
  //   .createIndex("event_rsvps_event_source_id_index")
  //   .on("event_rsvps")
  //   .column("event_source_id")
  //   .execute();
}

export async function down(db) {
  // Drop the indexes and constraints before dropping the table
  await db.schema.dropIndex("event_rsvps_user_id_index").execute();
  await db.schema.dropIndex("event_rsvps_event_id_index").execute();
  // await db.schema.dropIndex("event_rsvps_event_source_id_index").execute();

  // Drop the event_rsvps table
  await db.schema.dropTable("event_rsvps").execute();
}

