# Database Migrations

This directory contains database migration files that are automatically executed
in numerical order by the `run_migrations` Go binary during application startup.

## How It Fits Into the Architecture

1. **Main App** (`functions/gateway/main.go`) starts independently
2. **Migration Binary** (`cmd/run-migrations/main.go`) discovers and executes
   files from this directory
3. **SQL Files** in this directory are executed manually when needed via
   `npm run docker:migrations:run`

## Naming Convention

Migrations must follow this naming pattern:

```
{sequence_number}_{description}.sql
```

### Examples:

- `001_add_location_timezone.sql`
- `002_add_user_preferences.sql`
- `003_update_event_schema.sql`

## Rules

1. **Sequence Numbers**: Must be sequential integers starting from 001
2. **Descriptions**: Use lowercase with underscores, descriptive of the change
3. **File Extension**: Must end with `.sql`
4. **Ordering**: Files are executed in numerical order (001, 002, 003, etc.)

## Adding New Migrations

1. Create a new file with the next available sequence number
2. Use a descriptive name that explains what the migration does
3. Make the migration idempotent (safe to run multiple times)
4. Test the migration on a copy of production data before deploying

## Example Migration

```sql
-- Migration: 001_add_location_timezone.sql
-- Description: Adds location_timezone column to seshujobs table

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'seshujobs'
        AND column_name = 'location_timezone'
    ) THEN
        ALTER TABLE seshujobs ADD COLUMN location_timezone TEXT;
        RAISE NOTICE 'Added location_timezone column to seshujobs table';
    ELSE
        RAISE NOTICE 'Column location_timezone already exists in seshujobs table';
    END IF;
END$$;
```

## Important Notes

- **Never delete or rename existing migration files** - this will break the
  sequence
- **Always test migrations** before deploying to production
- **Use transactions** when possible to ensure rollback capability
- **Keep migrations small and focused** on single schema changes
- **Use `IF NOT EXISTS` checks** to make migrations idempotent

## Execution Flow

```
Manual Migration Execution
       ↓
npm run docker:migrations:run
       ↓
Migration binary scans this directory
       ↓
Files sorted by sequence number
       ↓
Each .sql file executed in order
       ↓
Migration complete - app continues running
```

## Troubleshooting

- **Migration not running**: Check that the file follows the naming convention
- **Sequence errors**: Ensure files are numbered sequentially without gaps
- **Execution failures**: Check SQL syntax and database permissions
- **Missing files**: Verify the migrations directory is properly mounted in
  Docker
