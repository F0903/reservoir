package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
)

var (
	ErrConfigFileOpen      = errors.New("config file open failed")
	ErrConfigFileWrite     = errors.New("config file write failed")
	ErrConfigFileRead      = errors.New("config file read failed")
	ErrConfigInvalidFormat = errors.New("config invalid format")
	ErrConfigPersist       = errors.New("config persist failed")
)

type Config struct {
	Proxy     ProxyConfig     `json:"proxy"`
	Webserver WebserverConfig `json:"webserver"`
	Cache     CacheConfig     `json:"cache"`
	Logging   LogConfig       `json:"logging"`
}

// Marks all properties that require a restart to take effect as needing a restart.
func (c *Config) setRestartNeededProps() {
	c.Proxy.setRestartNeededProps()
	c.Webserver.setRestartNeededProps()
	c.Cache.setRestartNeededProps()
	c.Logging.setRestartNeededProps()
}

func newDefault() *Config {
	cfg := &Config{
		Proxy:     defaultProxyConfig(),
		Webserver: defaultWebserverConfig(),
		Cache:     defaultCacheConfig(),
		Logging:   defaultLogConfig(),
	}
	cfg.setRestartNeededProps()
	return cfg
}

// Writes the configuration to disk.
func (c *Config) persist() error {
	f, err := os.Create(configPath.Path)
	if err != nil {
		slog.Error("Failed to create config file", "path", configPath.Path, "error", err)
		return fmt.Errorf("%w: failed to open config file for writing '%s'", ErrConfigFileOpen, configPath.Path)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ") // Pretty print the JSON output
	if err := enc.Encode(c); err != nil {
		slog.Error("Failed to encode config to JSON", "path", configPath.Path, "error", err)
		return fmt.Errorf("%w: failed to write config to file '%s'", ErrConfigFileWrite, configPath.Path)
	}

	slog.Info("Successfully persisted config", "path", configPath.Path)
	return nil
}

func load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		slog.Error("Failed to open config file", "path", path, "error", err)
		return nil, fmt.Errorf("%w: failed to open config file '%s'", ErrConfigFileRead, path)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		slog.Error("Failed to decode config JSON", "path", path, "error", err)
		return nil, fmt.Errorf("%w: failed to decode config from '%s'", ErrConfigInvalidFormat, path)
	}

	if err := cfg.verify(); err != nil {
		slog.Error("Unable to verify config", "path", path, "error", err)
		return nil, err
	}

	cfg.setRestartNeededProps()

	slog.Info("Successfully loaded config", "path", path)
	return &cfg, nil
}

func loadOrDefault(path string) (*Config, error) {
	cfg, err := load(path)
	if err != nil {
		slog.Error("Config load failed. Resetting to defaults.", "path", path, "error", err)
		cfg = newDefault()
		if err := cfg.persist(); err != nil {
			slog.Error("Failed to persist default config after reset", "error", err)
			return nil, fmt.Errorf("%w: failed to persist default config", ErrConfigPersist)
		}
		slog.Warn("Config file has been reset to defaults due to error.")
	}
	return cfg, nil
}
