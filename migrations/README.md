# Database Migrations

This directory contains SQL migration files that are automatically executed when
the application starts.

## Migration Files

Migration files must follow the naming convention: `NNN_description.sql`

- `NNN`: A three-digit sequence number (e.g., `001`, `002`, `003`)
- `description`: A brief description of what the migration does
- `.sql`: The file extension

Example: `001_add_location_timezone.sql`

## Dynamic Table Names

To keep database table names in sync across the application, migrations can use
dynamic table names via PostgreSQL session variables.

### Available Session Variables

- `app.seshu_jobs_table_name`: The name of the seshu jobs table (default:
  `seshujobs`)

### How It Works

1. **Environment Variable**: Set `SESHU_JOBS_TABLE_NAME` in your environment
   (`.env` file or Docker Compose)
2. **Go Migration Runner**: The migration runner reads the environment variable
   and sets it as a PostgreSQL session variable
3. **SQL Migration**: Use `current_setting('app.seshu_jobs_table_name', true)`
   to access the table name in your SQL

### Example Migration with Dynamic Table Name

```sql
DO $$
DECLARE
    table_name_var TEXT;
BEGIN
    -- Get table name from session variable (set by Go migration runner)
    table_name_var := current_setting('app.seshu_jobs_table_name', true);

    -- Fallback to default if not set
    IF table_name_var IS NULL OR table_name_var = '' THEN
        table_name_var := 'seshujobs';
    END IF;

    -- Use dynamic SQL with format() to safely quote the table name
    EXECUTE format('ALTER TABLE %I ADD COLUMN new_column TEXT', table_name_var);
END$$;
```

### Configuration Locations

The table name configuration is defined in multiple places to ensure
consistency:

1. **Environment Variables** (`.env` file or Docker Compose):

   ```bash
   SESHU_JOBS_TABLE_NAME=seshujobs
   ```

2. **Docker Compose** (`docker-compose.yml`):

   ```yaml
   environment:
     SESHU_JOBS_TABLE_NAME: ${SESHU_JOBS_TABLE_NAME:-seshujobs}
   ```

3. **Go Constants** (`functions/gateway/helpers/constants.go`):

   ```go
   const SESHU_JOBS_TABLE_NAME = "seshujobs" // Default value
   ```

4. **Migration Runner** (`functions/gateway/startup/run_migrations.go`):
   - Reads environment variable
   - Sets PostgreSQL session variable
   - Provides default fallback

### Benefits

- **Single Source of Truth**: Change the table name in one place (environment
  variable)
- **Type Safety**: Go code uses constants, not magic strings
- **Flexibility**: Easy to use different table names for testing or different
  environments
- **Backward Compatible**: Defaults to original table name if environment
  variable is not set

## Creating New Migrations

1. Create a new file with the next sequence number
2. Use the idempotent pattern with `IF NOT EXISTS` checks
3. Use dynamic table names when referencing application tables
4. Test the migration in a local environment first

## Migration Execution

Migrations are automatically executed:

- On application startup (except in test environment)
- In sequence order (sorted by number)
- With retry logic for database connectivity
- With proper error handling and logging

To manually run migrations:

```bash
# Set environment variables
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5433
export POSTGRES_DB=postgres
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export POSTGRES_MIGRATIONS_DIR=./migrations
export SESHU_JOBS_TABLE_NAME=seshujobs

# Run the application (migrations run on startup)
go run functions/gateway/main.go
```

## Troubleshooting

### Migration fails with "relation does not exist"

Check that:

1. The initial table creation script (`seshujobs_init.sql`) has been executed by
   Docker
2. The table name in the environment variable matches the actual table name
3. The PostgreSQL session variable is being set correctly (check logs)

### Migration runs multiple times

Migrations use idempotent patterns with `IF NOT EXISTS` checks, so running them
multiple times should be safe. However, if you see unexpected behavior:

1. Check that the migration logic properly handles existing state
2. Verify the migration sequence numbers are unique
3. Consider adding a migration tracking table in the future

### Table name mismatch

If you see errors about table names not matching:

1. Check the `SESHU_JOBS_TABLE_NAME` environment variable in `.env`
2. Verify Docker Compose is passing the environment variable correctly
3. Restart the Docker containers to pick up environment changes
4. Check the application logs for warnings about missing session variables
