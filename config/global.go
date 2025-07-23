package config

import (
	"errors"
	"fmt"
	"log/slog"
	"reservoir/utils/assertedpath"
	"reservoir/utils/writesynced"
)

var (
	ErrUpdateFailed = errors.New("failed to update global config")
)

var configPath = assertedpath.Assert("var/config.json")

var global *writesynced.WriteSynced[Config] = func() *writesynced.WriteSynced[Config] {
	cfg, err := loadOrDefault(configPath.Path)
	if err != nil {
		slog.Error("Failed to load global config", "error", err)
		panic(err)
	}
	return writesynced.New(cfg)
}()

// Returns a copy of the current global config
func Get() Config {
	return global.Get()
}

// Update applies the provided function to the global config, verifies it, and persists it.
// If verification or persistence fails, it reverts to the old config.
func Update(f func(*Config)) error {
	if f == nil {
		slog.Error("Update function is nil, skipping config update")
		return nil
	}

	old := global.Get()
	new := global.UpdateAndGet(func(global_config *Config) {
		f(global_config)
	})

	if err := new.verify(); err != nil {
		slog.Error("Updated global config failed verification", "error", err)
		global.Set(old) // Revert to old config on error
		return fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	if err := new.persist(); err != nil {
		slog.Error("Failed to persist updated global config", "error", err)
		global.Set(old) // Revert to old config on error
		return fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	slog.Info("Global config updated successfully", "new_config", new, "old_config", old)
	return nil
}
