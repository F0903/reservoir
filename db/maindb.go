package db

import "fmt"

const mainDbPath = "var/database.db"
const mainDbBusyTimeout = 5000 // ms

func OpenMainDatabase() (Database, error) {
	return Open(mainDbPath, mainDbBusyTimeout)
}

func migrateMainDatabase() error {
	db, err := OpenMainDatabase()
	if err != nil {
		return fmt.Errorf("failed to open main database: %w", err)
	}
	if err := db.Migrate(); err != nil {
		return err
	}
	db.Close()
	return nil
}
