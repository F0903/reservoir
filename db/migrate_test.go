package db

import (
	"path/filepath"
	"testing"
)

func TestMigrateIsIdempotent(t *testing.T) {
	databasePath := filepath.ToSlash(filepath.Join(t.TempDir(), "database.db"))
	database, err := Open(databasePath, 5000)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		t.Fatalf("first migration failed: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("second migration failed: %v", err)
	}

	var roleColumns int
	if err := database.Get(&roleColumns, "SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'is_admin'"); err != nil {
		t.Fatalf("failed to inspect users schema: %v", err)
	}
	if roleColumns != 1 {
		t.Fatalf("expected one is_admin column, got %d", roleColumns)
	}

	var appliedMigrations int
	if err := database.Get(&appliedMigrations, "SELECT COUNT(*) FROM schema_migrations"); err != nil {
		t.Fatalf("failed to count applied migrations: %v", err)
	}
	if appliedMigrations != 2 {
		t.Fatalf("expected 2 applied migrations, got %d", appliedMigrations)
	}
}
