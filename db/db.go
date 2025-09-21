package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func Open(path string, busyTimeout int) (*sqlx.DB, error) {
	// Windows-friendly file URI. Relative path is fine too.
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(%d)&_pragma=foreign_keys(1)", path, busyTimeout)
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Pool settings (SQLite = 1 writer; keep small)
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
		return nil, err
	}
	return db, nil
}
