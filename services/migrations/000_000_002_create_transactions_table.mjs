import { Kysely, sql } from 'kysely';

export async function up(db) {
  // Create the transactions table
  await db.schema
    .createTable("transactions")
    .addColumn("id", "uuid", (col) => col.primaryKey())
    .addColumn("user_id", "uuid", (col) => col.notNull())
    .addColumn("amount", "numeric", (col) => col.notNull())
    .addColumn("currency", "varchar(3)", (col) => col.notNull()) // ISO 4217 currency code
    .addColumn("transaction_type", "varchar(50)", (col) => col.notNull()) // E.g., 'credit', 'debit'
    .addColumn("transaction_date", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn("status", "varchar(50)", (col) => col.notNull()) // E.g., 'completed', 'pending', 'failed'
    .addColumn("description", "text")
    .addColumn("reference_id", "varchar(255)")
    .addColumn("created_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn("updated_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .execute();

  // Create an index on user_id for better query performance
  await db.schema
    .createIndex("transactions_user_id_index")
    .on("transactions")
    .column("user_id")
    .execute();

  // Optionally, add a check constraint for transaction_type and status to mimic enum behavior
  await db.schema
    .alterTable("transactions")
    .addConstraint(
      "transactions_type_status_check",
      sql`CHECK (transaction_type IN ('credit', 'debit')) AND (status IN ('completed', 'pending', 'failed'))`
    )
    .execute();
}

export async function down(db) {
  // Drop the index before dropping the table
  await db.schema.dropIndex("transactions_user_id_index").execute();

  // Drop the transactions table
  await db.schema.dropTable("transactions").execute();
}

