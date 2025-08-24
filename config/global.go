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

var Global *writesynced.WriteSynced[Config] = func() *writesynced.WriteSynced[Config] {
	cfg, err := loadOrDefault(configPath.Path)
	if err != nil {
		slog.Error("Failed to load global config", "error", err)
		panic(err)
	}
	return writesynced.New(*cfg)
}()

// UpdateAndVerify applies the provided function to the global config, verifies it, and persists it.
// If verification or persistence fails, it reverts to the old config.
func UpdateAndVerify(f func(*Config)) error {
	if f == nil {
		slog.Error("Update function is nil, skipping config update")
		return nil
	}

	cfgLock := Global.Mutable()
	cfg := cfgLock.Get()
	defer cfgLock.UnGet()

	old := *cfg
	f(cfg)

	if err := cfg.verify(); err != nil {
		slog.Error("Updated global config failed verification", "error", err)
		cfg = &old // Revert to old config on error
		return fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	if err := cfg.persist(); err != nil {
		slog.Error("Failed to persist updated global config", "error", err)
		cfg = &old // Revert to old config on error
		return fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	slog.Info("Global config updated successfully", "new_config", cfg, "old_config", old)
	return nil
}

func UpdatePartialFromJSON(updates map[string]any) error {
	slog.Debug("Updating global config with partial JSON", "updates", updates)

	if updates == nil {
		slog.Error("UpdatePartialFromJSON called with nil updates")
		return nil
	}

	return UpdateAndVerify(func(cfg *Config) {
		setPropsFromMap(cfg, updates)
	})
}
