import { Kysely, sql } from 'kysely';

export async function up(db) {
  // Create the transactions table with constraints directly
  await db.schema
    .createTable("transactions")
    .addColumn("id", "uuid", (col) => col.primaryKey())
    .addColumn("user_id", "uuid", (col) => col.notNull())
    .addColumn("amount", "numeric", (col) => col.notNull())
    .addColumn("currency", "varchar(3)", (col) => col.notNull()) // ISO 4217 currency code
    .addColumn("transaction_type", "varchar(50)", (col) => col.notNull())
    .addColumn("transaction_date", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn("status", "varchar(50)", (col) => col.notNull())
    .addColumn("description", "text")
    .addColumn("reference_id", "varchar(255)")
    .addColumn("created_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn("updated_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addCheckConstraint(
      "transactions_type_check",
      sql`transaction_type IN ('credit', 'debit')`
    )
    .addCheckConstraint(
      "transactions_status_check",
      sql`status IN ('completed', 'pending', 'failed')`
    )
    .execute();

  // Create an index on user_id for better query performance
  await db.schema
    .createIndex("transactions_user_id_index")
    .on("transactions")
    .column("user_id")
    .execute();
}

export async function down(db) {
  // Drop the index before dropping the table
  await db.schema.dropIndex("transactions_user_id_index").execute();

  // Drop the transactions table
  await db.schema.dropTable("transactions").execute();
}

