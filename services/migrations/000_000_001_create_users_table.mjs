import { Kysely, sql } from 'kysely';

export async function up(db) {
  // Create the users table with constraints directly
  await db.schema
    .createTable("users")
    .addColumn("id", "uuid", (col) => col.primaryKey())
    .addColumn("name", "varchar(255)", (col) => col.notNull())
    .addColumn("email", "varchar(255)", (col) => col.notNull().unique())
    .addColumn("address", "varchar(255)")
    .addColumn("phone", "varchar(20)")
    .addColumn("profile_picture_url", "varchar(255)")
    .addColumn('category_preferences', 'varchar(510)')
    .addColumn("created_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn("updated_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn("role", "varchar(50)", (col) => col.notNull())
    .addCheckConstraint(
      "users_role_check",
      sql`role IN ('standard_user', 'organization_user', 'suborganization_user')`
    )
    .execute();

  // Create an index on the email column for better query performance
  await db.schema
    .createIndex("users_email_index")
    .on("users")
    .column("email")
    .execute();
}

export async function down(db) {
  // Drop the index before dropping the table
  await db.schema.dropIndex("users_email_index").execute();

  // Drop the users table
  await db.schema.dropTable("users").execute();
}

