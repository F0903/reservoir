package db

import (
	"fmt"
	"log/slog"
)

func MigrateDatabases() error {
	slog.Debug("Migrating all databases...")
	if err := migrateMainDatabase(); err != nil {
		return fmt.Errorf("failed to migrate main database: %w", err)
	}

	// Add future database migrations here

	slog.Debug("All databases migrated successfully")
	return nil
}
