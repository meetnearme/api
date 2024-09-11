import { Kysely, sql } from 'kysely';

export async function up(db) {
  // Create the purchasables table with check constraints
  await db.schema
    .createTable("purchasables")
    .addColumn("id", "uuid", (col) => col.primaryKey())
    .addColumn("user_id", "uuid", (col) => col.notNull())
    .addColumn("name", "varchar(255)", (col) => col.notNull())
    .addColumn("item_type", "varchar(50)", (col) => col.notNull())
    .addColumn("cost", "numeric", (col) => col.notNull())
    .addColumn("currency", "varchar(3)", (col) => col.notNull()) // ISO 4217 currency code
    .addColumn("donation_ratio", "numeric")
    .addColumn("inventory", "integer")
    .addColumn("charge_recurrence_interval", "varchar(20)")
    .addColumn("charge_recurrence_interval_count", "integer")
    .addColumn("charge_recurrence_end_date", "timestamp")
    .addColumn("created_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn("updated_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addCheckConstraint(
      "purchasables_item_type_check",
      sql`item_type IN ('ticket', 'membership', 'donation', 'partialDonation', 'merchandise')`
    )
    .addCheckConstraint(
      "purchasables_charge_recurrence_interval_check",
      sql`charge_recurrence_interval IN ('day', 'week', 'month', 'year')`
    )
    .execute();

  // Create an index on item_type for better query performance
  await db.schema
    .createIndex("purchasables_item_type_index")
    .on("purchasables")
    .column("item_type")
    .execute();

  await db.schema
    .createIndex("purchasable_user_id_index")
    .on("purchasables")
    .column("user_id")
    .execute();
}

export async function down(db) {
  // Drop the index before dropping the table
  await db.schema.dropIndex("purchasables_item_type_index").execute();

  // Drop the purchasables table
  await db.schema.dropTable("purchasables").execute();
}

