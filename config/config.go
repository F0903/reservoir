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

const configVersion = 10

var (
	ErrConfigFileOpen        = errors.New("config file open failed")
	ErrConfigFileWrite       = errors.New("config file write failed")
	ErrConfigFileRead        = errors.New("config file read failed")
	ErrConfigVersionMismatch = errors.New("config version mismatch")
	ErrConfigInvalidFormat   = errors.New("config invalid format")
	ErrConfigPersist         = errors.New("config persist failed")
)

type CacheType string

var (
	CacheTypeFile   CacheType = "file"
	CacheTypeMemory CacheType = "memory"
)

type Config struct {
	ConfigVersion            ConfigProp[int]               `json:"config_version"`              // Version of the config file format, used for future migrations to ensure compatibility.
	ProxyListen              ConfigProp[string]            `json:"proxy_listen"`                // The address and port that the proxy will listen on.
	CaCert                   ConfigProp[string]            `json:"ca_cert"`                     // Path to CA certificate file.
	CaKey                    ConfigProp[string]            `json:"ca_key"`                      // Path to CA private key file.
	UpstreamDefaultHttps     ConfigProp[bool]              `json:"upstream_default_https"`      // If true, the proxy will always send HTTPS instead of HTTP to the upstream server.
	RetryOnRange416          ConfigProp[bool]              `json:"retry_on_range_416"`          // If true, the proxy will retry a request without the Range header if the upstream responds with a 416 Range Not Satisfiable.
	RetryOnInvalidRange      ConfigProp[bool]              `json:"retry_on_invalid_range"`      // If true, the proxy will retry a request without the Range header if the client sends an invalid Range header. (not recommended)
	WebserverListen          ConfigProp[string]            `json:"webserver_listen"`            // The address and port that the webserver (dashboard and API) will listen on.
	DashboardDisabled        ConfigProp[bool]              `json:"dashboard_disabled"`          // If true, the dashboard will be disabled. The API must also be enabled if the dashboard is enabled.
	ApiDisabled              ConfigProp[bool]              `json:"api_disabled"`                // If true, the API will be disabled.
	IgnoreCacheControl       ConfigProp[bool]              `json:"ignore_cache_control"`        // If true, the proxy will ignore Cache-Control headers from the upstream response.
	MaxCacheSize             ConfigProp[bytesize.ByteSize] `json:"max_cache_size"`              // The maximum size of the cache in bytes. If the cache exceeds this size, entries will be evicted.
	DefaultCacheMaxAge       ConfigProp[duration.Duration] `json:"default_cache_max_age"`       // The default cache max age to use if the upstream response does not specify a Cache-Control or Expires header.
	ForceDefaultCacheMaxAge  ConfigProp[bool]              `json:"force_default_cache_max_age"` // If true, always use the default cache max age even if the upstream response has a Cache-Control or Expires header.
	CacheType                ConfigProp[CacheType]         `json:"cache_type"`                  // The type of cache to use. Supported values are "memory" and "file".
	CacheDir                 ConfigProp[string]            `json:"cache_dir"`                   // The directory where cached files will be stored. (only valid for file-based caches)
	CacheMemoryBudgetPercent ConfigProp[int]               `json:"cache_memory_budget_percent"` // The percentage of total memory to use for the cache. (only valid for memory-based caches)
	CacheCleanupInterval     ConfigProp[duration.Duration] `json:"cache_cleanup_interval"`      // The interval at which the cache will be cleaned up to remove expired entries.
	CacheLockShards          ConfigProp[int]               `json:"cache_lock_shards"`           // The number of shards to use for per-key locking. High values increase concurrency but use more memory.
	LogLevel                 ConfigProp[slog.Level]        `json:"log_level"`                   // The log level to use for the application.
	LogFile                  ConfigProp[string]            `json:"log_file"`                    // The path to the log file. If empty, no file logging will be done.
	LogFileMaxSize           ConfigProp[bytesize.ByteSize] `json:"log_file_max_size"`           // The maximum size of the log file.
	LogFileMaxBackups        ConfigProp[int]               `json:"log_file_max_backups"`        // The maximum number of old log files to retain.
	LogFileCompress          ConfigProp[bool]              `json:"log_file_compress"`           // If true, old log files will be compressed.
	LogToStdout              ConfigProp[bool]              `json:"log_to_stdout"`               // If true, log messages will be written to stdout.
}

// Marks all properties that require a restart to take effect as needing a restart.
func (c *Config) setRestartNeededProps() {
	c.ProxyListen.SetRequiresRestart()
	c.CaCert.SetRequiresRestart()
	c.CaKey.SetRequiresRestart()
	c.WebserverListen.SetRequiresRestart()
	c.DashboardDisabled.SetRequiresRestart()
	c.ApiDisabled.SetRequiresRestart()
	c.CacheType.SetRequiresRestart()
	c.CacheDir.SetRequiresRestart()
	c.CacheLockShards.SetRequiresRestart()
}

func newDefault() *Config {
	cfg := &Config{
		ConfigVersion:            NewConfigProp(configVersion),
		ProxyListen:              NewConfigProp(":9999"),
		CaCert:                   NewConfigProp("ssl/ca.crt"),
		CaKey:                    NewConfigProp("ssl/ca.key"),
		UpstreamDefaultHttps:     NewConfigProp(true),
		RetryOnRange416:          NewConfigProp(true),
		RetryOnInvalidRange:      NewConfigProp(false),
		WebserverListen:          NewConfigProp("localhost:8080"),
		DashboardDisabled:        NewConfigProp(false),
		ApiDisabled:              NewConfigProp(false),
		IgnoreCacheControl:       NewConfigProp(true), // This this is primarily targeted at caching package managers, we want to cache aggressively by default.
		MaxCacheSize:             NewConfigProp(bytesize.ParseUnchecked("10G")),
		DefaultCacheMaxAge:       NewConfigProp(duration.Duration(1 * time.Hour)),
		ForceDefaultCacheMaxAge:  NewConfigProp(true), // Since this is again primarily targeted at caching apt repositories, we want to cache aggressively by default.
		CacheType:                NewConfigProp(CacheTypeMemory),
		CacheDir:                 NewConfigProp("var/cache/"),
		CacheMemoryBudgetPercent: NewConfigProp(75),
		CacheCleanupInterval:     NewConfigProp(duration.Duration(90 * time.Minute)),
		CacheLockShards:          NewConfigProp(1024),
		LogLevel:                 NewConfigProp(slog.LevelInfo),
		LogFile:                  NewConfigProp("var/proxy.log"),
		LogFileMaxSize:           NewConfigProp(bytesize.ParseUnchecked("500M")),
		LogFileMaxBackups:        NewConfigProp(3),
		LogFileCompress:          NewConfigProp(true),
		LogToStdout:              NewConfigProp(false),
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
	if c.ConfigVersion.Read() != configVersion {
		return ErrConfigVersionMismatch
	}

	if c.ProxyListen.Read() == "" {
		return fmt.Errorf("proxy_listen cannot be empty")
	}

	if c.WebserverListen.Read() == "" {
		return fmt.Errorf("webserver_listen cannot be empty")
	}

	if c.MaxCacheSize.Read().Bytes() <= 0 {
		return fmt.Errorf("max_cache_size must be greater than 0")
	}

	if c.CacheMemoryBudgetPercent.Read() < 0 || c.CacheMemoryBudgetPercent.Read() > 100 {
		return fmt.Errorf("cache_memory_budget_percent must be between 0 and 100")
	}

	if c.CacheCleanupInterval.Read().Cast() <= 0 {
		return fmt.Errorf("cache_cleanup_interval must be greater than 0")
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

type stagedProp interface {
	CommitStaged()
}

// Dynamically sets the properties of the Config struct based on the provided map.
// This allows for partial updates to the config without needing to know the exact structure of the Config struct.
func setPropsFromMap(cfg *Config, updates map[string]any) (stagedProps []stagedProp, err error) {
	configT := reflect.TypeOf(*cfg)
	configV := reflect.ValueOf(cfg).Elem()

	stagedProps = make([]stagedProp, 0, 1)

	// This could definitely be optimized, but for the current usage, this should be more than sufficient.
	for key, value := range updates {
		slog.Debug("Processing update", "key", key, "value", value)

		for i := 0; i < configT.NumField(); i++ {
			propT := configT.Field(i)

			// Match the JSON tag to the key
			propJsonName, ok := propT.Tag.Lookup("json")
			if !ok || propJsonName != key {
				slog.Debug("Skipping field, was not match", "field", propT.Name, "json_name", propJsonName)
				continue
			}

			propV := configV.Field(i)
			slog.Debug("Found matching field", "field", propT.Name, "type", propT.Type, "field_value", propV)

			if !propV.CanSet() {
				slog.Error("Cannot set field", "field", propT.Name, "type", propT.Type, "field_value", propV)
				continue
			}

			if !propV.CanAddr() {
				slog.Error("Cannot get address of field", "field", propT.Name, "type", propT.Type, "field_value", propV)
				continue
			}
			propVAddr := propV.Addr()

			unmarshalJsonStaged := propVAddr.MethodByName("UnmarshalJSONStaged")
			if unmarshalJsonStaged.IsZero() {
				slog.Error("UnmarshalJSONStaged method was not found!", "field", propT.Name, "type", propT.Type, "field_value", propV)
				continue
			}

			// Marshal the value back to JSON, so we can unmarshal again to handle all the different conversions.
			valueBytes, err := json.Marshal(value)
			if err != nil {
				slog.Error("Failed to marshal value to JSON", "field", propT.Name, "error", err)
				continue
			}

			returns := unmarshalJsonStaged.Call([]reflect.Value{reflect.ValueOf(valueBytes)})
			result := returns[0]
			if !result.IsNil() {
				err := result.Interface().(error)
				slog.Error("UnmarshalJSONStaged failed", "field", propT.Name, "error", err)
				continue
			}

			stagedProps = append(stagedProps, propVAddr.Interface().(stagedProp))

			slog.Debug("Field updated successfully", "field", propT.Name, "type", propT.Type, "new_value", value)
			break // We found the field, no need to continue
		}
	}

	return stagedProps, nil
}

type UpdateStatus int

const (
	UpdateStatusFailed UpdateStatus = iota
	UpdateStatusSuccess
	UpdateStatusRestartRequired
)

type ConfigSubscriber struct {
	unsubs []func()
}

func (s *ConfigSubscriber) Add(unsub func()) {
	s.unsubs = append(s.unsubs, unsub)
}

func (s *ConfigSubscriber) UnsubscribeAll() {
	for _, unsub := range s.unsubs {
		if unsub != nil {
			unsub()
		}
	}
	s.unsubs = nil
}

func UpdatePartialFromJSON(updates map[string]any) (UpdateStatus, error) {
	slog.Info("Updating global config with partial JSON", "updates", updates)

	if updates == nil {
		slog.Error("UpdatePartialFromJSON called with nil updates")
		return UpdateStatusFailed, nil
	}

	slog.Debug("Setting properties from JSON map...", "updates", updates)
	stagedProps, err := setPropsFromMap(Global, updates)
	if err != nil {
		slog.Error("Failed to set properties from map", "error", err)
		return UpdateStatusFailed, fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	slog.Info("Committing updated properties...", "staged_count", len(stagedProps))
	for _, prop := range stagedProps {
		slog.Debug("Committing property...", "prop", prop)
		prop.CommitStaged()
	}

	if err := Global.verify(); err != nil {
		slog.Error("Updated global config failed verification", "error", err)
		return UpdateStatusFailed, fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	if err := Global.persist(); err != nil {
		slog.Error("Failed to persist updated global config", "error", err)
		return UpdateStatusFailed, fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	status := UpdateStatusSuccess
	if IsRestartNeeded() {
		slog.Info("Restart is required after updating global config")
		status = UpdateStatusRestartRequired
	}
	return status, nil
}
