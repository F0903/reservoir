package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"reservoir/config/flags"
	"reservoir/utils"
	"reservoir/utils/bytesize"
	"reservoir/utils/duration"
	"time"
)

const configVersion = 7

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
	ProxyListen             ConfigProp[string]            `json:"proxy_listen"`                // The address and port that the proxy will listen on.
	CaCert                  ConfigProp[string]            `json:"ca_cert"`                     // Path to CA certificate file.
	CaKey                   ConfigProp[string]            `json:"ca_key"`                      // Path to CA private key file.
	UpstreamDefaultHttps    ConfigProp[bool]              `json:"upstream_default_https"`      // If true, the proxy will always send HTTPS instead of HTTP to the upstream server.
	RetryOnRange416         ConfigProp[bool]              `json:"retry_on_range_416"`          // If true, the proxy will retry a request without the Range header if the upstream responds with a 416 Range Not Satisfiable.
	WebserverListen         ConfigProp[string]            `json:"webserver_listen"`            // The address and port that the webserver (dashboard and API) will listen on.
	DashboardDisabled       ConfigProp[bool]              `json:"dashboard_disabled"`          // If true, the dashboard will be disabled. The API must also be enabled if the dashboard is enabled.
	ApiDisabled             ConfigProp[bool]              `json:"api_disabled"`                // If true, the API will be disabled.
	CacheDir                ConfigProp[string]            `json:"cache_dir"`                   // The directory where cached files will be stored.
	IgnoreCacheControl      ConfigProp[bool]              `json:"ignore_cache_control"`        // If true, the proxy will ignore Cache-Control headers from the upstream response.
	MaxCacheSize            ConfigProp[bytesize.ByteSize] `json:"max_cache_size"`              // The maximum size of the cache in bytes. If the cache exceeds this size, entries will be evicted.
	DefaultCacheMaxAge      ConfigProp[duration.Duration] `json:"default_cache_max_age"`       // The default cache max age to use if the upstream response does not specify a Cache-Control or Expires header.
	ForceDefaultCacheMaxAge ConfigProp[bool]              `json:"force_default_cache_max_age"` // If true, always use the default cache max age even if the upstream response has a Cache-Control or Expires header.
	CacheCleanupInterval    ConfigProp[duration.Duration] `json:"cache_cleanup_interval"`      // The interval at which the cache will be cleaned up to remove expired entries.
	LogLevel                ConfigProp[slog.Level]        `json:"log_level"`                   // The log level to use for the application.
	LogFile                 ConfigProp[string]            `json:"log_file"`                    // The path to the log file. If empty, no file logging will be done.
	LogFileMaxSize          ConfigProp[bytesize.ByteSize] `json:"log_file_max_size"`           // The maximum size of the log file.
	LogFileMaxBackups       ConfigProp[int]               `json:"log_file_max_backups"`        // The maximum number of old log files to retain.
	LogFileCompress         ConfigProp[bool]              `json:"log_file_compress"`           // If true, old log files will be compressed.
	LogToStdout             ConfigProp[bool]              `json:"log_to_stdout"`               // If true, log messages will be written to stdout.
}

// Marks all properties that require a restart to take effect as needing a restart.
func (c *Config) setRestartNeededProps() {
	c.ProxyListen.SetRequiresRestart()
	c.CaCert.SetRequiresRestart()
	c.CaKey.SetRequiresRestart()
	c.RetryOnRange416.SetRequiresRestart()
	c.WebserverListen.SetRequiresRestart()
	c.DashboardDisabled.SetRequiresRestart()
	c.ApiDisabled.SetRequiresRestart()
	c.CacheDir.SetRequiresRestart()
	c.DefaultCacheMaxAge.SetRequiresRestart()
	c.LogFile.SetRequiresRestart()
	c.LogFileMaxSize.SetRequiresRestart()
	c.LogFileMaxBackups.SetRequiresRestart()
	c.LogFileCompress.SetRequiresRestart()
	c.LogToStdout.SetRequiresRestart()
}

func newDefault() *Config {
	cfg := &Config{
		ConfigVersion:           NewConfigProp(configVersion),
		ProxyListen:             NewConfigProp(":9999"),
		CaCert:                  NewConfigProp("ssl/ca.crt"),
		CaKey:                   NewConfigProp("ssl/ca.key"),
		UpstreamDefaultHttps:    NewConfigProp(true),
		RetryOnRange416:         NewConfigProp(true),
		WebserverListen:         NewConfigProp("localhost:8080"),
		DashboardDisabled:       NewConfigProp(false),
		ApiDisabled:             NewConfigProp(false),
		CacheDir:                NewConfigProp("var/cache/"),
		IgnoreCacheControl:      NewConfigProp(true), // This this is primarily targeted at caching package managers, we want to cache aggressively by default.
		MaxCacheSize:            NewConfigProp(bytesize.ParseUnchecked("10G")),
		DefaultCacheMaxAge:      NewConfigProp(duration.Duration(1 * time.Hour)),
		ForceDefaultCacheMaxAge: NewConfigProp(true), // Since this is again primarily targeted at caching apt repositories, we want to cache aggressively by default.
		CacheCleanupInterval:    NewConfigProp(duration.Duration(90 * time.Minute)),
		LogLevel:                NewConfigProp(slog.LevelInfo),
		LogFile:                 NewConfigProp("var/proxy.log"),
		LogFileMaxSize:          NewConfigProp(bytesize.ParseUnchecked("500M")),
		LogFileMaxBackups:       NewConfigProp(3),
		LogFileCompress:         NewConfigProp(true),
		LogToStdout:             NewConfigProp(false),
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

func (c *Config) verify() error {
	// Could perhaps add more validation in the future.
	if c.ConfigVersion.Read() != configVersion {
		return ErrConfigVersionMismatch
	}
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
		return nil, fmt.Errorf("%w: %v", ErrConfigVersionMismatch, err)
	}

	cfg.setRestartNeededProps()

	slog.Info("Successfully loaded config", "path", path)
	return &cfg, nil
}

func loadOrDefault(path string) (*Config, error) {
	cfg, err := load(path)
	if err != nil {
		slog.Warn("Failed to load config, using defaults", "path", path, "error", err)
		cfg = newDefault()
		if err := cfg.persist(); err != nil {
			slog.Error("Failed to persist default config", "error", err)
			return nil, fmt.Errorf("%w: failed to persist default config", ErrConfigPersist)
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

			valueBytes, err := json.Marshal(value)
			if err != nil {
				slog.Error("Failed to marshal value to JSON", "field", fieldT.Name, "error", err)
				continue
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

func (c *Config) OverrideFromFlags() {
	fl := flags.New()
	fl.AddString("listen", ":9999", "The address and port that the proxy will listen on").OnSet(func(val flags.FlagValue) { c.ProxyListen.Overwrite(val.AsString()) })
	fl.AddString("ca-cert", "ssl/ca.crt", "Path to CA certificate file").OnSet(func(val flags.FlagValue) { c.CaCert.Overwrite(val.AsString()) })
	fl.AddString("ca-key", "ssl/ca.key", "Path to CA private key file").OnSet(func(val flags.FlagValue) { c.CaKey.Overwrite(val.AsString()) })
	fl.AddString("cache-dir", "var/cache/", "Path to cache directory").OnSet(func(val flags.FlagValue) { c.CacheDir.Overwrite(val.AsString()) })
	fl.AddString("webserver-listen", "localhost:8080", "The address and port that the webserver (dashboard and API) will listen on").OnSet(func(val flags.FlagValue) { c.WebserverListen.Overwrite(val.AsString()) })
	fl.AddBool("no-dashboard", false, "Disable the dashboard. The API must also be enabled if the dashboard is enabled.").OnSet(func(val flags.FlagValue) { c.DashboardDisabled.Overwrite(val.AsBool()) })
	fl.AddBool("no-api", false, "Disable the API").OnSet(func(val flags.FlagValue) { c.ApiDisabled.Overwrite(val.AsBool()) })
	fl.AddString("log-level", "", "Set the logging level (DEBUG, INFO, WARN, ERROR)").OnSet(func(val flags.FlagValue) {
		level := utils.StringToLogLevel(val.AsString())
		c.LogLevel.Overwrite(level)
	})
	fl.AddString("log-file", "", "Set the log file path").OnSet(func(val flags.FlagValue) { c.LogFile.Overwrite(val.AsString()) })
	fl.AddString("log-file-max-size", "", "Set the log file max size").OnSet(func(val flags.FlagValue) { c.LogFileMaxSize.Overwrite(val.AsBytesize()) })
	fl.AddInt("log-file-max-backups", 3, "Set the log file max backups").OnSet(func(val flags.FlagValue) { c.LogFileMaxBackups.Overwrite(val.AsInt()) })
	fl.AddBool("log-file-compress", true, "Set the log file compress").OnSet(func(val flags.FlagValue) { c.LogFileCompress.Overwrite(val.AsBool()) })
	fl.AddBool("log-to-stdout", false, "Enable logging to stdout").OnSet(func(val flags.FlagValue) { c.LogToStdout.Overwrite(val.AsBool()) })
	fl.Parse()
}

func ParseFlags() {
	UpdateAndVerify(func(cfg *Config) {
		cfg.OverrideFromFlags()
	})
}
