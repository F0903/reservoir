package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var (
	ErrMigrationFailed = errors.New("database migration failed")
)

//go:embed migrations/*.sql
var migrationFs embed.FS

type Database struct {
	raw *sqlx.DB
}

func Open(path string, busyTimeout int) (Database, error) {
	// Windows-friendly file URI. Relative path is fine too.
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(%d)&_pragma=foreign_keys(1)", path, busyTimeout)
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return Database{}, err
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Improve read concurrency and durability balance
	if _, err := db.ExecContext(ctx, `
    PRAGMA journal_mode = WAL;
    PRAGMA synchronous = NORMAL;
  `); err != nil {
		_ = db.Close()
		return Database{}, err
	}

	// Return by value for now since we are just wrapping a pointer
	return Database{raw: db}, nil
}

func IsResponseEmpty(err error) bool {
	// If other db backends are added, this can be made part of a Database interface
	return errors.Is(err, sql.ErrNoRows)
}

func (db *Database) Get(dest any, query string, args ...any) error {
	return db.raw.Get(dest, query, args...)
}

func (db *Database) Select(dest any, query string, args ...any) error {
	return db.raw.Select(dest, query, args...)
}

func (db *Database) Exec(query string, args ...any) error {
	_, err := db.raw.Exec(query, args...)
	return err
}

func (db *Database) ExecResult(query string, args ...any) (sql.Result, error) {
	return db.raw.Exec(query, args...)
}

func (db *Database) Migrate() error {
	slog.Debug("Migrating database...")
	files, err := migrationFs.ReadDir("migrations")
	if err != nil {
		slog.Error("Failed to read database migrations", "error", err)
		return fmt.Errorf("%w: %v", ErrMigrationFailed, err)
	}

	tx, err := db.raw.Begin()
	if err != nil {
		slog.Error("Failed to begin database transaction", "error", err)
		return err
	}

	if err := ensureSchemaMigrations(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if err := applyMigration(tx, file.Name()); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	slog.Debug("Database migration completed successfully. Committing...")
	return tx.Commit()
}

func ensureSchemaMigrations(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   TEXT     PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		slog.Error("Failed to create schema migrations table", "error", err)
		return fmt.Errorf("%w: %v", ErrMigrationFailed, err)
	}
	return nil
}

func applyMigration(tx *sql.Tx, filename string) error {
	applied, err := migrationApplied(tx, filename)
	if err != nil {
		return err
	}
	if applied {
		slog.Debug("Skipping already applied migration", "file", filename)
		return nil
	}

	migration, err := migrationFs.ReadFile("migrations/" + filename)
	if err != nil {
		slog.Error("Failed to read migration", "file", filename, "error", err)
		return fmt.Errorf("%w: %v", ErrMigrationFailed, err)
	}

	if _, err := tx.Exec(string(migration)); err != nil {
		slog.Error("Failed to execute migration. Rolling back...", "file", filename, "error", err)
		return fmt.Errorf("%w: %v", ErrMigrationFailed, err)
	}

	if _, err := tx.Exec("INSERT INTO schema_migrations (filename) VALUES (?)", filename); err != nil {
		slog.Error("Failed to record migration", "file", filename, "error", err)
		return fmt.Errorf("%w: %v", ErrMigrationFailed, err)
	}
	return nil
}

func migrationApplied(tx *sql.Tx, filename string) (bool, error) {
	var applied int
	if err := tx.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE filename = ?", filename).Scan(&applied); err != nil {
		slog.Error("Failed to check migration status", "file", filename, "error", err)
		return false, fmt.Errorf("%w: %v", ErrMigrationFailed, err)
	}
	return applied > 0, nil
}

func (db *Database) Close() error {
	return db.raw.Close()
}
