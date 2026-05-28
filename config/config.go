package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reservoir/utils/assertedpath"
)

var (
	ErrConfigFileOpen      = errors.New("config file open failed")
	ErrConfigFileWrite     = errors.New("config file write failed")
	ErrConfigFileRead      = errors.New("config file read failed")
	ErrConfigInvalidFormat = errors.New("config invalid format")
	ErrConfigPersist       = errors.New("config persist failed")
)

const DefaultConfigPath = "var/config.json"

type Config struct {
	Proxy     ProxyConfig     `json:"proxy"`
	Webserver WebserverConfig `json:"webserver"`
	Cache     CacheConfig     `json:"cache"`
	Logging   LogConfig       `json:"logging"`
	Database  DatabaseConfig  `json:"database"`

	path string
}

// Marks all properties that require a restart to take effect as needing a restart.
func (c *Config) setRestartNeededProps() {
	c.Proxy.setRestartNeededProps()
	c.Webserver.setRestartNeededProps()
	c.Cache.setRestartNeededProps()
	c.Logging.setRestartNeededProps()
	c.Database.setRestartNeededProps()
}

func NewDefault() *Config {
	cfg := &Config{
		Proxy:     defaultProxyConfig(),
		Webserver: defaultWebserverConfig(),
		Cache:     defaultCacheConfig(),
		Logging:   defaultLogConfig(),
		Database:  defaultDatabaseConfig(),
		path:      DefaultConfigPath,
	}
	cfg.setRestartNeededProps()
	return cfg
}

// Writes the configuration to disk.
func (c *Config) persist() error {
	path := c.path
	if path == "" {
		path = DefaultConfigPath
	}

	assertedPath, err := assertedpath.TryAssert(path)
	if err != nil {
		slog.Error("Failed to assert config file path", "path", path, "error", err)
		return fmt.Errorf("%w: failed to assert config file path '%s'", ErrConfigFileOpen, path)
	}

	f, err := os.Create(assertedPath.Path)
	if err != nil {
		slog.Error("Failed to create config file", "path", assertedPath.Path, "error", err)
		return fmt.Errorf("%w: failed to open config file for writing '%s'", ErrConfigFileOpen, assertedPath.Path)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ") // Pretty print the JSON output
	if err := enc.Encode(c); err != nil {
		slog.Error("Failed to encode config to JSON", "path", assertedPath.Path, "error", err)
		return fmt.Errorf("%w: failed to write config to file '%s'", ErrConfigFileWrite, assertedPath.Path)
	}

	slog.Info("Successfully persisted config", "path", assertedPath.Path)
	return nil
}

func load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		slog.Error("Failed to open config file", "path", path, "error", err)
		return nil, fmt.Errorf("%w: failed to open config file '%s': %w", ErrConfigFileRead, path, err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		slog.Error("Failed to decode config JSON", "path", path, "error", err)
		return nil, fmt.Errorf("%w: failed to decode config from '%s'", ErrConfigInvalidFormat, path)
	}

	migrated := cfg.applyMigrations()
	cfg.path = path
	if err := cfg.verify(); err != nil {
		slog.Error("Unable to verify config", "path", path, "error", err)
		return nil, err
	}

	cfg.setRestartNeededProps()
	if migrated {
		if err := cfg.persist(); err != nil {
			slog.Error("Failed to persist migrated config", "path", path, "error", err)
			return nil, fmt.Errorf("%w: failed to persist migrated config", ErrConfigPersist)
		}
	}

	slog.Info("Successfully loaded config", "path", path)
	return &cfg, nil
}

func LoadOrDefault(path string) (*Config, error) {
	cfg, err := load(path)
	if err == nil {
		return cfg, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		slog.Error("Config load failed. Existing config will not be overwritten.", "path", path, "error", err)
		return nil, err
	}

	slog.Info("Config file does not exist. Creating defaults.", "path", path)
	cfg = NewDefault()
	cfg.path = path
	if err := cfg.persist(); err != nil {
		slog.Error("Failed to persist default config", "error", err)
		return nil, fmt.Errorf("%w: failed to persist default config", ErrConfigPersist)
	}
	return cfg, nil
}
