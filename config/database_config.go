package config

import "fmt"

type DatabaseConfig struct {
	File ConfigProp[string] `json:"file"` // Path to the SQLite database file.
}

func (c *DatabaseConfig) setRestartNeededProps() {
	c.File.SetRequiresRestart()
}

func (c *DatabaseConfig) verify() error {
	if c.File.Read() == "" {
		return fmt.Errorf("database.file cannot be empty")
	}
	return nil
}

func defaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		File: NewConfigProp("var/database.db"),
	}
}
