import { Kysely, sql } from 'kysely'

export async function up(db) {
  // Create the users table
  await db.schema
    .createTable("users")
    .addColumn("id", "uuid", (col) => col.primaryKey())
    .addColumn("name", "varchar(255)", (col) => col.notNull())
    .addColumn("email", "varchar(255)", (col) => col.notNull().unique())
    .addColumn("address", "varchar(255)")
    .addColumn("phone", "varchar(20)")
    .addColumn("profile_picture_url", "varchar(255)")
    .addColumn("created_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn("updated_at", "timestamp", (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn("role", "varchar(50)", (col) => col.notNull())
    .addColumn("organization_user_id", "uuid") // Foreign key for suborganization role
    .execute();

  // Create an index on the email column for better query performance
  await db.schema
    .createIndex("users_email_index")
    .on("users")
    .column("email")
    .execute();

  // Optionally, add a check constraint for role to mimic enum behavior
  // await db.schema
  //   .alterTable("users")
  //   .addConstraint(
  //     "users_role_check",
  //     sql`CHECK (role IN ('standard_user', 'organization_user', 'suborganization_user'))`
  //   )
  //   .execute();

  // Add foreign key constraint
  // await db.schema
  //   .alterTable("users")
  //   .addForeignKeyConstraint("users_organization_user_fk", {
  //     columns: ["organization_user_id"],
  //     referencedTable: "users",
  //     referencedColumns: ["id"],
  //     onDelete: "SET NULL", // Adjust behavior as needed
  //     condition: sql`role = 'suborganization_user'`
  //   })
  //   .execute();
}

export async function down(db) {
  // Drop the foreign key constraint before dropping the column
  // await db.schema.dropForeignKeyConstraint("users_organization_user_fk").execute();

  // Drop the index before dropping the table
  await db.schema.dropIndex("users_email_index").execute();

  // Drop the users table
  await db.schema.dropTable("users").execute();
}

