import { Kysely, sql } from 'kysely';

export async function up(db) {
  // Create the purchasables table
  await db.schema
    .createTable("purchasables")
    .addColumn("id", "uuid", (col) => col.primaryKey())
    .addColumn("name", "varchar(255)", (col) => col.notNull())
    .addColumn("item_type", "varchar(50)", (col) => col.notNull()) // Enum values: 'ticket', 'membership', 'donation', 'partialDonation', 'merchandise'
    .addColumn("cost", "numeric", (col) => col.notNull())
    .addColumn("currency", "varchar(3)", (col) => col.notNull()) // ISO 4217 currency code
    .addColumn("donation_ratio", "numeric")
    .addColumn("inventory", "integer")
    .addColumn("charge_recurrence_interval", "varchar(20)") // Enum values: 'day', 'week', 'month', 'year'
    .addColumn("charge_recurrence_interval_count", "integer")
    .addColumn("charge_recurrence_end_date", "timestamp")
    .addColumn("created_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn("updated_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .execute();

  // Create an index on item_type for better query performance
  await db.schema
    .createIndex("purchasables_item_type_index")
    .on("purchasables")
    .column("item_type")
    .execute();

  // Optionally, add a check constraint for item_type to mimic enum behavior
  await db.schema
    .alterTable("purchasables")
    .addConstraint(
      "purchasables_item_type_check",
      sql`CHECK (item_type IN ('ticket', 'membership', 'donation', 'partialDonation', 'merchandise'))`
    )
    .execute();

  // Optionally, add a check constraint for charge_recurrence_interval to mimic enum behavior
  await db.schema
    .alterTable("purchasables")
    .addConstraint(
      "purchasables_charge_recurrence_interval_check",
      sql`CHECK (charge_recurrence_interval IN ('day', 'week', 'month', 'year'))`
    )
    .execute();
}

export async function down(db) {
  // Drop the indexes and constraints before dropping the table
  await db.schema.dropIndex("purchasables_item_type_index").execute();
  await db.schema.dropConstraint("purchasables_item_type_check").execute();
  await db.schema.dropConstraint("purchasables_charge_recurrence_interval_check").execute();

  // Drop the purchasables table
  await db.schema.dropTable("purchasables").execute();
}

