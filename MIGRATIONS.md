# Database Migrations

This document describes the database migration system for the MeetNearMe API.

## Overview

The migration system automatically runs database schema updates when the
application starts. This ensures that the database schema is always up-to-date
with the latest code changes.

## Architecture Components

### 1. **Main Application** (`functions/gateway/main.go`)

- **Role**: Core HTTP server and business logic
- **Responsibility**: Connects to services, starts HTTP server
- **Migration Integration**: Migrations run manually via npm script

### 2. **Migration Runner Binary** (`cmd/run-migrations/main.go`)

- **Role**: Executes SQL migration files
- **Responsibility**: Discovers, validates, and runs migrations in order
- **Execution**: Called manually via `npm run docker:migrations:run`

### 3. **Utility Binaries** (`cmd/` directory)

- **`weaviate-setup`**: Manual Weaviate schema setup (npm:
  `docker:weaviate:create-schema`)
- **`run-migrations`**: Manual database migration execution (npm:
  `docker:migrations:run`)
- **`seed-weaviate-db`**: Manual data seeding utility
- **`clean-weaviate-db`**: Manual cleanup utility

## Migration Files

- `seshujobs_init.sql` - Initial schema creation (runs on first database
  startup)
- `migrations/` - Directory containing numbered migration files that run in
  order

### Migration Naming Convention

Migrations use the pattern: `{sequence_number}_{description}.sql`

Examples:

- `001_add_location_timezone.sql` - Adds `location_timezone` column to
  `seshujobs` table
- `002_add_user_preferences.sql` - Future migration example
- `003_update_event_schema.sql` - Future migration example

Migrations are automatically discovered and executed in numerical order.

## How It Works

### Development (docker-compose)

1. **Postgres container starts** and runs `seshujobs_init.sql` if it's a fresh
   database
2. **Go app starts** and connects to postgres
3. **Migrations run manually** using `npm run docker:migrations:run` when needed
4. **App serves requests** immediately after startup

### Production (Docker)

1. **Container starts** and runs the main application directly
2. **Migrations run manually** using the `run_migrations` binary when needed
3. **Application starts** immediately without waiting for migrations

### Manual Operations

```bash
# Setup Weaviate schema (one-time setup)
npm run docker:weaviate:create-schema

# Run database migrations (when needed)
npm run docker:migrations:run

# Seed database with data
go run cmd/seed-weaviate-db/main.go --file=path/to/data.json

# Clean database
go run cmd/clean-weaviate-db/main.go
```

## Startup Sequence

```
1. Docker container starts
   ↓
2. startup.sh is created at runtime
   ↓
3. startup.sh runs /go-app/run_migrations
   ↓
4. Migration binary discovers and executes SQL files
   ↓
5. startup.sh starts /go-app/main
   ↓
6. Main app serves requests
```

## Adding New Migrations

1. Create a new migration file in the `migrations/` directory
2. Use the naming pattern: `{next_sequence_number}_{description}.sql`
3. The migration system automatically discovers and runs new migrations
4. Migrations run during Go app startup using the `run_migrations` binary

### Example

To add a new migration for user preferences:

```bash
# Create the new migration file
touch migrations/002_add_user_preferences.sql

# Add your SQL migration code to the file
# The system will automatically run it on next startup
```

## Migration Best Practices

- **Always make migrations idempotent** (safe to run multiple times)
- **Use `IF NOT EXISTS` checks** for adding columns/tables
- **Test migrations on a copy of production data** before deploying
- **Keep migrations small and focused** on single schema changes
- **Use descriptive names for migration files**

## Troubleshooting

### Migration Issues

- Check application logs for migration errors
- Verify migration files are properly numbered
- Ensure database connection parameters are correct

### Utility Binary Issues

- Check environment variables (WEAVIATE_HOST, WEAVIATE_PORT, etc.)
- Verify the binary was built correctly in Docker
- Check npm script configurations in package.json

### Startup Issues

- Check that postgres is healthy before app starts
- Verify migrations directory is properly mounted
- Check that all required binaries are copied to the container
