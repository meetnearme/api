import { Kysely, sql } from 'kysely'

export async function up(db: Kysely<any>): Promise<void> {
  // Create the organizations table
  await db.schema
    .createTable("organizations")
    .addColumn("id", "uuid", (col) => col.primaryKey())
    .addColumn("name", "varchar(255)", (col) => col.notNull())
    .addColumn("description", "text")
    .addColumn("is_sub_organization", "boolean", (col) => col.notNull())
    .addColumn("parent_organization_id", "uuid", (col) =>
      col.references("organizations.id").onDelete("set null"))
    .addColumn("address_street", "varchar(255)")
    .addColumn("address_city", "varchar(255)")
    .addColumn("address_zip_code", "varchar(20)")
    .addColumn("address_country", "varchar(255)")
    .addColumn("email", "varchar(255)")
    .addColumn("website", "varchar(255)")
    .addColumn("phone", "varchar(20)")
    .addColumn("logo_url", "varchar(255)")
    .addColumn("created_at", "timestamp", (col) =>
      col.notNull().defaultTo(sql`now()`))
    .addColumn("updated_at", "timestamp", (col) =>
      col.notNull().defaultTo(sql`now()`))
    .execute();

  // Create an index on parent_organization_id for better query performance
  await db.schema
    .createIndex("organizations_parent_id_index")
    .on("organizations")
    .column("parent_organization_id")
    .execute();
}

export async function down(db: Kysely<any>): Promise<void> {
  // Drop the index before dropping the table
  await db.schema.dropIndex("organizations_parent_id_index").execute();

  // Drop the organizations table
  await db.schema.dropTable("organizations").execute();
}

