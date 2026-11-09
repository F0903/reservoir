package config

func (c *Config) applyMigrations() bool {
	migrated := false
	defaults := NewDefault()

	if !c.Database.File.IsSet() {
		c.Database.File = defaults.Database.File
		migrated = true
	}

	// The old value is dropped on purpose: its default of true is saved in every
	// existing config, and carrying it over would break hosts that only serve HTTP.
	if c.Proxy.LegacyUpstreamDefaultHttps != nil {
		c.Proxy.LegacyUpstreamDefaultHttps = nil
		migrated = true
	}
	if !c.Proxy.UpgradeHttpToHttps.IsSet() {
		c.Proxy.UpgradeHttpToHttps = defaults.Proxy.UpgradeHttpToHttps
		migrated = true
	}

	if !c.Webserver.TrustedProxies.IsSet() {
		c.Webserver.TrustedProxies = defaults.Webserver.TrustedProxies
		migrated = true
	}

	return migrated
}
