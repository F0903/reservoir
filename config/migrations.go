package config

func (c *Config) applyMigrations() bool {
	migrated := false
	defaults := NewDefault()

	if !c.Database.File.IsSet() {
		c.Database.File = defaults.Database.File
		migrated = true
	}

	return migrated
}
