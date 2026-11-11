package config

// applyMigrations reports whether it changed the config. Settings missing from an
// old file need no migration: load decodes the file on top of the defaults.
func (c *Config) applyMigrations() bool {
	// The old value is dropped on purpose: its default of true is saved in every
	// existing config, and carrying it over would break hosts that only serve HTTP.
	if c.Proxy.LegacyUpstreamDefaultHttps != nil {
		c.Proxy.LegacyUpstreamDefaultHttps = nil
		return true
	}
	return false
}
