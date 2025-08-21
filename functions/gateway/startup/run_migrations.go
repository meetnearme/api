package startup

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Migration represents a database migration file
type Migration struct {
	Sequence int
	Filename string
	FullPath string
	Content  string
}

// InitMigrations runs database migrations
func InitMigrations() error {
	// Database connection parameters
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbName := os.Getenv("POSTGRES_DB")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")

	// Migration directory
	migrationsDir := os.Getenv("POSTGRES_MIGRATIONS_DIR")

	fmt.Println("Starting database migrations...")

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		fmt.Printf("No migrations directory found at %s\n", migrationsDir)
		return nil // Not an error, just no migrations to run
	}

	// Discover migration files
	migrations, err := discoverMigrations(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to discover migrations: %w", err)
	}

	if len(migrations) == 0 {
		fmt.Println("No migration files found")
		return nil // Not an error, just no migrations to run
	}

	fmt.Printf("Found %d migration(s):\n", len(migrations))
	for _, migration := range migrations {
		fmt.Printf("   - %s\n", migration.Filename)
	}

	// Connect to database
	db, err := connectToDatabase(dbHost, dbPort, dbName, dbUser, dbPassword)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Run migrations
	if err := runMigrations(db, migrations); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Println("All migrations completed successfully!")
	return nil
}

// init function for backward compatibility
func init() {
	if err := InitMigrations(); err != nil {
		log.Fatalf("Migrations failed: %v", err)
	}
}

func discoverMigrations(migrationsDir string) ([]Migration, error) {
	var migrations []Migration

	// Read all .sql files in the migrations directory
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		// Parse sequence number from filename (e.g., "001_description.sql" -> 1)
		var sequence int
		parts := strings.Split(file.Name(), "_")
		if len(parts) > 0 {
			if _, err := fmt.Sscanf(parts[0], "%d", &sequence); err != nil {
				fmt.Printf("Skipping migration with invalid sequence number: %s\n", file.Name())
				continue
			}
		}

		// Read migration content
		fullPath := filepath.Join(migrationsDir, file.Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			fmt.Printf("Failed to read migration file %s: %v\n", file.Name(), err)
			continue
		}

		migrations = append(migrations, Migration{
			Sequence: sequence,
			Filename: file.Name(),
			FullPath: fullPath,
			Content:  string(content),
		})
	}

	// Sort migrations by sequence number
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Sequence < migrations[j].Sequence
	})

	return migrations, nil
}

func connectToDatabase(host, port, dbName, user, password string) (*sql.DB, error) {
	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		host, port, dbName, user, password)

	// Connect to database with retries
	var db *sql.DB
	var err error
	maxRetries := 10
	retryDelay := time.Second * 2

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			fmt.Printf("Failed to open database connection (attempt %d/%d): %v\n", i+1, maxRetries, err)
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				continue
			}
			return nil, fmt.Errorf("failed to open database connection after %d attempts: %w", maxRetries, err)
		}

		// Test the connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = db.PingContext(ctx)
		cancel()

		if err == nil {
			break
		}

		fmt.Printf("Failed to ping database (attempt %d/%d): %v\n", i+1, maxRetries, err)
		db.Close()

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	fmt.Println("Connected to database successfully")
	return db, nil
}

func runMigrations(db *sql.DB, migrations []Migration) error {
	for _, migration := range migrations {
		fmt.Printf("Running migration: %s\n", migration.Filename)

		// Execute the migration
		if _, err := db.Exec(migration.Content); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration.Filename, err)
		}

		fmt.Printf("Migration %s completed successfully\n", migration.Filename)
	}

	return nil
}
