package config

import (
	"apt_cacher_go/utils/assertedpath"
	"apt_cacher_go/utils/bytesize"
	"apt_cacher_go/utils/duration"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
)

//TODO: add file watcher to reload config on changes

const configVersion = 1

var (
	ErrConfigFileOpen        = errors.New("config file open failed")
	ErrConfigFileWrite       = errors.New("config file write failed")
	ErrConfigFileRead        = errors.New("config file read failed")
	ErrConfigVersionMismatch = errors.New("config version mismatch")
	ErrConfigInvalidFormat   = errors.New("config invalid format")
	ErrConfigPersist         = errors.New("config persist failed")
)

var configPath = assertedpath.Assert("var/config.json")

var Global *Config = func() *Config {
	cfg, err := LoadOrDefault(configPath.Path)
	if err != nil {
		slog.Error("Failed to load global config", "error", err)
		panic(err)
	}
	return cfg
}()

type Config struct {
	ConfigVersion           int               // Version of the config file format, used for future migrations to ensure compatibility.
	AlwaysCache             bool              // If true, the proxy will always cache responses, even if the upstream response requests the opposite.
	MaxCacheSize            bytesize.ByteSize // The maximum size of the cache in bytes. If the cache exceeds this size, entries will be evicted.
	DefaultCacheMaxAge      duration.Duration // The default cache max age to use if the upstream response does not specify a Cache-Control or Expires header.
	ForceDefaultCacheMaxAge bool              // If true, always use the default cache max age even if the upstream response has a Cache-Control or Expires header.
	CacheCleanupInterval    duration.Duration // The interval at which the cache will be cleaned up to remove expired entries.
	UpstreamDefaultHttps    bool              // If true, the proxy will always send HTTPS instead of HTTP to the upstream server.
}

func Default() *Config {
	return &Config{
		ConfigVersion:           configVersion,
		AlwaysCache:             true, // This this is primarily targeted at caching apt repositories, we want to cache aggressively by default.
		MaxCacheSize:            bytesize.ParseUnchecked("10G"),
		DefaultCacheMaxAge:      duration.Duration(1 * time.Hour),
		ForceDefaultCacheMaxAge: true, // Since this is again primarily targeted at caching apt repositories, we want to cache aggressively by default.
		CacheCleanupInterval:    duration.Duration(90 * time.Minute),
		UpstreamDefaultHttps:    true,
	}
}

// Writes the configuration to disk.
func (c *Config) Persist() error {
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

func Load(path string) (*Config, error) {
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

	if cfg.ConfigVersion != configVersion {
		slog.Error("Config version mismatch", "path", path, "expected", configVersion, "got", cfg.ConfigVersion)
		return nil, fmt.Errorf("%w: expected %d, got %d", ErrConfigVersionMismatch, configVersion, cfg.ConfigVersion)
	}

	slog.Info("Successfully loaded config", "path", path)
	return &cfg, nil
}

func LoadOrDefault(path string) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		slog.Warn("Failed to load config, using defaults", "path", path, "error", err)
		cfg = Default()
		if err := cfg.Persist(); err != nil {
			slog.Error("Failed to persist default config", "error", err)
			return nil, fmt.Errorf("%w: failed to persist default config", ErrConfigPersist)
		}
		slog.Info("Created default config file", "path", path)
	}
	return cfg, nil
}
