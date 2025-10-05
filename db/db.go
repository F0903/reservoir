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

func (db *Database) Exec(query string, args ...any) error {
	_, err := db.raw.Exec(query, args...)
	return err
}

func (db *Database) Migrate() error {
	slog.Debug("Migrating database...")
	files, _ := migrationFs.ReadDir("migrations")

	tx, err := db.raw.Begin()
	if err != nil {
		slog.Error("Failed to begin database transaction", "error", err)
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		mig, _ := migrationFs.ReadFile("migrations/" + file.Name())
		migStr := string(mig)

		_, err := tx.Exec(migStr)
		if err != nil {
			slog.Error("Failed to execute migration. Rolling back...", "file", file.Name(), "error", err)
			_ = tx.Rollback()
			return fmt.Errorf("%w: %v", ErrMigrationFailed, err)
		}
	}

	slog.Debug("Database migration completed successfully. Committing...")
	return tx.Commit()
}

func (db *Database) Close() error {
	return db.raw.Close()
}
