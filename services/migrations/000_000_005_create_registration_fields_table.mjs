import { Kysely, sql } from 'kysely'

export async function up(db) {
  // Create the registration_fields table
  await db.schema
    .createTable('registration_fields')
    .addColumn('id', 'uuid', (col) => col.primaryKey().defaultTo(sql`gen_random_uuid()`))
    .addColumn('name', 'varchar(255)', (col) => col.notNull())
    .addColumn('type', 'varchar(50)', (col) => col.notNull())
    .addColumn('options', 'text[]')
    .addColumn('required', 'boolean', (col) => col.notNull().defaultTo(false))
    .addColumn('default', 'varchar(255)')
    .addColumn('placeholder', 'varchar(255)')
    .addColumn('description', 'text')
    .addColumn('created_at', 'timestamp', (col) => col.notNull().defaultTo(sql`now()`))
    .addColumn('updated_at', 'timestamp', (col) => col.notNull().defaultTo(sql`now()`))
    .execute()

  // Optionally, you can add an index for faster lookups if needed
  await db.schema
    .createIndex('registration_fields_name_index')
    .on('registration_fields')
    .column('name')
    .execute()
}

export async function down(db) {
  // Drop the index before dropping the table
  await db.schema.dropIndex('registration_fields_name_index').execute()

  // Drop the registration_fields table
  await db.schema.dropTable('registration_fields').execute()
}

