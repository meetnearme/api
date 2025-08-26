package startup

import (
	"fmt"
	"log"
	"os"
)

// StartupTask represents a startup task function
type StartupTask struct {
	Name string
	Task func() error
}

// Registry holds all startup tasks
var Registry []StartupTask

// Register adds a startup task to the registry
func Register(name string, task func() error) {
	Registry = append(Registry, StartupTask{
		Name: name,
		Task: task,
	})
}

// RunAll executes all registered startup tasks
func RunAll() error {
	log.Printf("Running %d startup tasks...", len(Registry))

	for _, task := range Registry {
		log.Printf("Running startup task: %s", task.Name)

		if err := task.Task(); err != nil {
			return fmt.Errorf("startup task '%s' failed: %w", task.Name, err)
		}

		log.Printf("Startup task '%s' completed successfully", task.Name)
	}

	log.Printf("All startup tasks completed successfully")
	return nil
}

// Auto-register startup tasks
func init() {
	// Skip registration in test environment
	if os.Getenv("GO_ENV") == "test" {
		log.Println("Skipping startup task registration in test environment")
		return
	}

	// Register all startup tasks
	Register("Database Migrations", InitMigrations)
	Register("Weaviate Schema Setup", InitWeaviate)
}
