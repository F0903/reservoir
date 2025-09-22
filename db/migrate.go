package db

import "fmt"

func MigrateDatabases() error {
	if err := migrateMainDatabase(); err != nil {
		return fmt.Errorf("failed to migrate main database: %w", err)
	}

	// Add future database migrations here

	return nil
}
