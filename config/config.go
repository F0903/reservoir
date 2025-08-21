package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"reservoir/utils/bytesize"
	"reservoir/utils/duration"
	"time"
)

const configVersion = 2

var (
	ErrConfigFileOpen        = errors.New("config file open failed")
	ErrConfigFileWrite       = errors.New("config file write failed")
	ErrConfigFileRead        = errors.New("config file read failed")
	ErrConfigVersionMismatch = errors.New("config version mismatch")
	ErrConfigInvalidFormat   = errors.New("config invalid format")
	ErrConfigPersist         = errors.New("config persist failed")
)

type Config struct {
	ConfigVersion           ConfigProp[int]               `json:"config_version"`              // Version of the config file format, used for future migrations to ensure compatibility.
	DashboardEnabled        ConfigProp[bool]              `json:"dashboard_enabled"`           // If true, the dashboard will be enabled.
	ApiEnabled              ConfigProp[bool]              `json:"api_enabled"`                 // If true, the API will be enabled. This will always be enabled if the dashboard is enabled.
	AlwaysCache             ConfigProp[bool]              `json:"always_cache"`                // If true, the proxy will always cache responses, even if the upstream response requests the opposite.
	MaxCacheSize            ConfigProp[bytesize.ByteSize] `json:"max_cache_size"`              // The maximum size of the cache in bytes. If the cache exceeds this size, entries will be evicted.
	DefaultCacheMaxAge      ConfigProp[duration.Duration] `json:"default_cache_max_age"`       // The default cache max age to use if the upstream response does not specify a Cache-Control or Expires header.
	ForceDefaultCacheMaxAge ConfigProp[bool]              `json:"force_default_cache_max_age"` // If true, always use the default cache max age even if the upstream response has a Cache-Control or Expires header.
	CacheCleanupInterval    ConfigProp[duration.Duration] `json:"cache_cleanup_interval"`      // The interval at which the cache will be cleaned up to remove expired entries.
	UpstreamDefaultHttps    ConfigProp[bool]              `json:"upstream_default_https"`      // If true, the proxy will always send HTTPS instead of HTTP to the upstream server.
	LogLevel                ConfigProp[slog.Level]        `json:"log_level"`                   // The log level to use for the application.
	LogFile                 ConfigProp[string]            `json:"log_file"`                    // The path to the log file. If empty, logging will only be done to stdout.
}

func newDefault() Config {
	return Config{
		ConfigVersion:           NewConfigProp(configVersion),
		DashboardEnabled:        NewConfigProp(true),
		ApiEnabled:              NewConfigProp(true),
		AlwaysCache:             NewConfigProp(true), // This this is primarily targeted at caching apt repositories, we want to cache aggressively by default.
		MaxCacheSize:            NewConfigProp(bytesize.ParseUnchecked("10G")),
		DefaultCacheMaxAge:      NewConfigProp(duration.Duration(1 * time.Hour)),
		ForceDefaultCacheMaxAge: NewConfigProp(true), // Since this is again primarily targeted at caching apt repositories, we want to cache aggressively by default.
		CacheCleanupInterval:    NewConfigProp(duration.Duration(90 * time.Minute)),
		UpstreamDefaultHttps:    NewConfigProp(true),
		LogLevel:                NewConfigProp(slog.LevelInfo),
		LogFile:                 NewConfigProp("var/proxy.log"),
	}
}

// Writes the configuration to disk.
func (c Config) persist() error {
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

func (c Config) verify() error {
	if c.ConfigVersion.Read() != configVersion {
		return ErrConfigVersionMismatch
	}
	return nil
}

func load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		slog.Error("Failed to open config file", "path", path, "error", err)
		return Config{}, fmt.Errorf("%w: failed to open config file '%s'", ErrConfigFileRead, path)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		slog.Error("Failed to decode config JSON", "path", path, "error", err)
		return Config{}, fmt.Errorf("%w: failed to decode config from '%s'", ErrConfigInvalidFormat, path)
	}

	if err := cfg.verify(); err != nil {
		slog.Error("Unable to verify config", "path", path, "error", err)
		return Config{}, fmt.Errorf("%w: %v", ErrConfigVersionMismatch, err)
	}

	slog.Info("Successfully loaded config", "path", path)
	return cfg, nil
}

func loadOrDefault(path string) (Config, error) {
	cfg, err := load(path)
	if err != nil {
		slog.Warn("Failed to load config, using defaults", "path", path, "error", err)
		cfg = newDefault()
		if err := cfg.persist(); err != nil {
			slog.Error("Failed to persist default config", "error", err)
			return Config{}, fmt.Errorf("%w: failed to persist default config", ErrConfigPersist)
		}
		slog.Info("Created default config file", "path", path)
	}
	return cfg, nil
}

// Dynamically sets the properties of the Config struct based on the provided map.
// This allows for partial updates to the config without needing to know the exact structure of the Config struct.
func setPropsFromMap(cfg *Config, updates map[string]any) {
	t := reflect.TypeOf(*cfg)
	v := reflect.ValueOf(cfg).Elem()

	// This could definitely be optimized, but for the current usage, this should be more than sufficient.
	for key, value := range updates {
		slog.Debug("Processing update", "key", key, "value", value)

		for i := 0; i < t.NumField(); i++ {
			fieldT := t.Field(i)

			fieldJsonName, ok := fieldT.Tag.Lookup("json")
			if !ok || fieldJsonName != key {
				slog.Debug("Skipping field, was not match", "field", fieldT.Name, "json_name", fieldJsonName)
				continue
			}
			fieldV := v.Field(i)

			slog.Debug("Found matching field", "field", fieldT.Name, "type", fieldT.Type, "field_value", fieldV)

			if !fieldV.CanSet() {
				slog.Error("Cannot set field", "field", fieldT.Name, "type", fieldT.Type, "field_value", fieldV)
				continue
			}

			if !fieldV.CanAddr() {
				slog.Error("Cannot get address of field", "field", fieldT.Name, "type", fieldT.Type, "field_value", fieldV)
				continue
			}
			fieldVAddr := fieldV.Addr()

			var valueBytes []byte
			switch v := value.(type) {
			case string:
				// If the value is a string, we need to add quotes around it to parse it correctly.
				valueBytes = fmt.Appendf(valueBytes, "\"%s\"", v)
			default:
				valueBytes = fmt.Appendf(valueBytes, "%v", v)
			}

			unmarshalJson := fieldVAddr.MethodByName("UnmarshalJSON")
			if unmarshalJson.IsZero() {
				slog.Error("UnmarshalJSON method was not found!", "field", fieldT.Name, "type", fieldT.Type, "field_value", fieldV)
				continue
			}
			returns := unmarshalJson.Call([]reflect.Value{reflect.ValueOf(valueBytes)})
			result := returns[0]
			if !result.IsNil() {
				err := result.Interface().(error)
				slog.Error("UnmarshalJSON failed", "field", fieldT.Name, "error", err)
				continue
			}

			slog.Debug("Field updated successfully", "field", fieldT.Name, "type", fieldT.Type, "new_value", value)
			break // We found the field, no need to continue
		}
	}
}
