package config

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"reservoir/utils/bytesize"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type subConfig struct {
	Prop ConfigProp[string] `json:"prop"`
}

type testConfig struct {
	Sub  subConfig       `json:"sub"`
	Flat ConfigProp[int] `json:"flat"`
}

func newTestConfig(t *testing.T) *Config {
	t.Helper()

	cfg := NewDefault()
	cfg.path = filepath.ToSlash(filepath.Join(t.TempDir(), "config.json"))
	return cfg
}

func applyPartialUpdate(t *testing.T, cfg *Config, updates map[string]any) CommitResult {
	t.Helper()

	update, err := PreparePartialUpdate(cfg, updates)
	if err != nil {
		t.Fatalf("PreparePartialUpdate failed: %v", err)
	}
	if err := update.Persist(); err != nil {
		t.Fatalf("Persist failed: %v", err)
	}
	result, err := update.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	return result
}

func TestSetPropsFromMapRecursive(t *testing.T) {
	cfg := &testConfig{
		Sub: subConfig{
			Prop: NewConfigProp("old"),
		},
		Flat: NewConfigProp(1),
	}

	updates := map[string]any{
		"sub": map[string]any{
			"prop": "new",
		},
		"flat": 2,
	}

	staged, err := setPropsFromMapRecursive(reflect.ValueOf(cfg), updates)
	if err != nil {
		t.Fatalf("setPropsFromMapRecursive failed: %v", err)
	}

	if len(staged) != 2 {
		t.Errorf("Expected 2 staged props, got %d", len(staged))
	}

	for _, s := range staged {
		s.CommitStaged()
	}

	if cfg.Sub.Prop.Read() != "new" {
		t.Errorf("Expected Sub.Prop to be 'new', got %s", cfg.Sub.Prop.Read())
	}

	if cfg.Flat.Read() != 2 {
		t.Errorf("Expected Flat to be 2, got %d", cfg.Flat.Read())
	}
}

func TestSetPropsFromMapRecursive_UnknownKeyReturnsValidationError(t *testing.T) {
	cfg := &testConfig{
		Sub: subConfig{
			Prop: NewConfigProp("old"),
		},
		Flat: NewConfigProp(1),
	}

	_, err := setPropsFromMapRecursive(reflect.ValueOf(cfg), map[string]any{
		"sub": map[string]any{
			"unknown": "new",
		},
	})
	if err == nil {
		t.Fatal("expected unknown key to return an error")
	}
	if !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if !strings.Contains(err.Error(), "sub.unknown") {
		t.Fatalf("expected error to include unknown key path, got %v", err)
	}
	if got := cfg.Sub.Prop.Read(); got != "old" {
		t.Fatalf("unknown key update changed config: got %q", got)
	}
}

func TestSetPropsFromMapRecursive_Partial(t *testing.T) {
	cfg := &testConfig{
		Sub: subConfig{
			Prop: NewConfigProp("old"),
		},
		Flat: NewConfigProp(1),
	}

	// Only update one nested field
	updates := map[string]any{
		"sub": map[string]any{
			"prop": "new",
		},
	}

	staged, err := setPropsFromMapRecursive(reflect.ValueOf(cfg), updates)
	if err != nil {
		t.Fatalf("setPropsFromMapRecursive failed: %v", err)
	}

	if len(staged) != 1 {
		t.Errorf("Expected 1 staged prop, got %d", len(staged))
	}

	staged[0].CommitStaged()

	if cfg.Sub.Prop.Read() != "new" {
		t.Errorf("Expected Sub.Prop to be 'new', got %s", cfg.Sub.Prop.Read())
	}

	if cfg.Flat.Read() != 1 {
		t.Errorf("Expected Flat to remain 1, got %d", cfg.Flat.Read())
	}
}

func TestUpdatePartialFromJSON_RealConfig(t *testing.T) {
	cfg := newTestConfig(t)

	updates := map[string]any{
		"cache": map[string]any{
			"max_cache_size": "50G",
			"file": map[string]any{
				"dir": "/tmp/new-cache",
			},
		},
		"proxy": map[string]any{
			"cache_policy": map[string]any{
				"ignore_cache_control": false,
			},
		},
		"logging": map[string]any{
			"level": "DEBUG",
		},
	}

	applyPartialUpdate(t, cfg, updates)

	if cfg.Cache.MaxCacheSize.Read().String() != "50G" {
		t.Errorf("Expected MaxCacheSize to be 50G, got %s", cfg.Cache.MaxCacheSize.Read().String())
	}

	if cfg.Cache.File.Dir.Read() != "/tmp/new-cache" {
		t.Errorf("Expected Cache.File.Dir to be /tmp/new-cache, got %s", cfg.Cache.File.Dir.Read())
	}

	if cfg.Proxy.CachePolicy.IgnoreCacheControl.Read() != false {
		t.Errorf("Expected CachePolicy.IgnoreCacheControl to be false, got %v", cfg.Proxy.CachePolicy.IgnoreCacheControl.Read())
	}

	if cfg.Logging.Level.Read() != slog.LevelDebug {
		t.Errorf("Expected LogLevel to be DEBUG, got %v", cfg.Logging.Level.Read())
	}
}

func TestPreparePartialUpdateRejectsUnknownNestedKey(t *testing.T) {
	cfg := newTestConfig(t)
	originalDir := cfg.Cache.File.Dir.Read()

	update, err := PreparePartialUpdate(cfg, map[string]any{
		"cache": map[string]any{
			"file": map[string]any{
				"missing": "value",
			},
		},
	})
	if err == nil {
		t.Fatal("expected unknown nested key to return an error")
	}
	if update != nil {
		t.Fatal("expected failed update preparation to return nil update")
	}
	if !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if !strings.Contains(err.Error(), "cache.file.missing") {
		t.Fatalf("expected error to include full key path, got %v", err)
	}
	if got := cfg.Cache.File.Dir.Read(); got != originalDir {
		t.Fatalf("unknown update committed cache dir: got %q, want %q", got, originalDir)
	}
}

func TestPartialUpdateCommitReportsRestartRequired(t *testing.T) {
	cfg := newTestConfig(t)

	result := applyPartialUpdate(t, cfg, map[string]any{
		"proxy": map[string]any{
			"listen": ":10000",
		},
	})
	if !result.RestartRequired {
		t.Fatal("expected proxy.listen update to require restart")
	}
}

func TestPartialUpdateCommitDoesNotReportRestartForHotSwapProperty(t *testing.T) {
	cfg := newTestConfig(t)

	result := applyPartialUpdate(t, cfg, map[string]any{
		"logging": map[string]any{
			"level": "DEBUG",
		},
	})
	if result.RestartRequired {
		t.Fatal("expected logging.level update to avoid restart")
	}
}

func TestPreparePartialUpdate_InvalidUpdateDoesNotCommitOrNotify(t *testing.T) {
	cfg := NewDefault()
	originalSize := cfg.Cache.MaxCacheSize.Read()

	var notified atomic.Bool
	unsub := cfg.Cache.MaxCacheSize.OnChange(func(_ bytesize.ByteSize) {
		notified.Store(true)
	})
	defer unsub()

	update, err := PreparePartialUpdate(cfg, map[string]any{
		"cache": map[string]any{
			"max_cache_size": "0B",
		},
	})
	if err == nil {
		t.Fatal("expected invalid update to return an error")
	}
	if update != nil {
		t.Fatal("expected invalid update to return nil update")
	}
	if !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if got := cfg.Cache.MaxCacheSize.Read(); got != originalSize {
		t.Fatalf("invalid update committed max cache size: got %s, want %s", got, originalSize)
	}
	if notified.Load() {
		t.Fatal("invalid update fired runtime subscriber")
	}
}

func TestPreparePartialUpdate_NilUpdateReturnsValidationError(t *testing.T) {
	update, err := PreparePartialUpdate(NewDefault(), nil)
	if err == nil {
		t.Fatal("expected nil update to return an error")
	}
	if update != nil {
		t.Fatal("expected nil update to return nil update")
	}
	if !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestLoadOrDefault_InvalidJSONDoesNotResetExistingConfig(t *testing.T) {
	tmpFile := filepath.ToSlash(filepath.Join(t.TempDir(), "test_config_strict.json"))

	invalidJSON := []byte("{ invalid json }")
	if err := os.WriteFile(tmpFile, invalidJSON, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadOrDefault(tmpFile)
	if err == nil {
		t.Fatal("expected invalid existing config to return an error")
	}
	if cfg != nil {
		t.Fatal("expected invalid existing config to return nil config")
	}
	if !errors.Is(err, ErrConfigInvalidFormat) {
		t.Fatalf("expected invalid format error, got %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(data, invalidJSON) {
		t.Fatal("invalid existing config was overwritten")
	}
}

func TestLoadOrDefaultPersistsDefaultsToMissingProvidedPath(t *testing.T) {
	tmpFile := filepath.ToSlash(filepath.Join(t.TempDir(), "nested", "config.json"))

	cfg, err := LoadOrDefault(tmpFile)
	if err != nil {
		t.Fatalf("LoadOrDefault should create missing config: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config after default creation")
	}

	reloaded, err := load(tmpFile)
	if err != nil {
		t.Fatalf("expected created config at provided path to be loadable: %v", err)
	}
	if got := reloaded.Database.File.Read(); got != "var/database.db" {
		t.Fatalf("expected default database file in created config, got %q", got)
	}
}

func TestLoadOrDefault_MigratesMissingDatabaseFile(t *testing.T) {
	tmpFile := filepath.ToSlash(filepath.Join(t.TempDir(), "config.json"))

	defaultData, err := json.Marshal(NewDefault())
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]any
	if err := json.Unmarshal(defaultData, &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw, "database")

	oldConfigData, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpFile, oldConfigData, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadOrDefault(tmpFile)
	if err != nil {
		t.Fatalf("expected old config to be migrated: %v", err)
	}
	if got := cfg.Database.File.Read(); got != "var/database.db" {
		t.Fatalf("expected migrated database file, got %q", got)
	}

	reloaded, err := load(tmpFile)
	if err != nil {
		t.Fatalf("expected migrated config to be persisted: %v", err)
	}
	if got := reloaded.Database.File.Read(); got != "var/database.db" {
		t.Fatalf("expected persisted migrated database file, got %q", got)
	}
}

func TestLoadOrDefault_MissingFields(t *testing.T) {
	tmpFile := filepath.ToSlash(filepath.Join(t.TempDir(), "test_config_missing.json"))

	// Write JSON with missing fields (e.g. no proxy section)
	err := os.WriteFile(tmpFile, []byte(`{"cache": {"type": "memory"}}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadOrDefault(tmpFile)
	if err == nil {
		t.Fatal("expected existing config with missing fields to return an error")
	}
	if cfg != nil {
		t.Fatal("expected existing config with missing fields to return nil config")
	}
	if !strings.Contains(err.Error(), "missing or uninitialized required configuration property") {
		t.Fatalf("expected missing field error, got %v", err)
	}
}

func TestDefaultConfig_HybridPackageCacheTuning(t *testing.T) {
	cfg := NewDefault()

	if got := cfg.Cache.Type.Read(); got != CacheTypeHybrid {
		t.Fatalf("expected default cache type %q, got %q", CacheTypeHybrid, got)
	}
	if got := cfg.Proxy.CachePolicy.DefaultMaxAge.Read().Cast(); got != 15*time.Minute {
		t.Fatalf("expected default cache max age 15m, got %s", got)
	}
	if got := cfg.Proxy.CachePolicy.ForceDefaultMaxAge.Read(); !got {
		t.Fatal("expected default cache policy to force the package freshness window")
	}
	if got := cfg.Proxy.CachePolicy.IgnoreCacheControl.Read(); !got {
		t.Fatal("expected default cache policy to ignore upstream cache-control")
	}
	if got := cfg.Cache.CleanupInterval.Read().Cast(); got != 5*time.Minute {
		t.Fatalf("expected default cleanup interval 5m, got %s", got)
	}
	if got := cfg.Cache.Memory.MemoryBudgetPercent.Read(); got != 25 {
		t.Fatalf("expected default memory tier limit 25%%, got %d%%", got)
	}
	if got := cfg.Cache.Hybrid.DemoteAfter.Read().Cast(); got != 5*time.Minute {
		t.Fatalf("expected default hybrid demotion after 5m, got %s", got)
	}
}

func TestConfig_Verify(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid default",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "valid hybrid cache type",
			modify: func(c *Config) {
				c.Cache.Type.Overwrite(CacheTypeHybrid)
			},
			wantErr: false,
		},
		{
			name: "empty proxy listen",
			modify: func(c *Config) {
				c.Proxy.Listen.Overwrite("")
			},
			wantErr: true,
		},
		{
			name: "empty webserver listen",
			modify: func(c *Config) {
				c.Webserver.Listen.Overwrite("")
			},
			wantErr: true,
		},
		{
			name: "negative max cache size",
			modify: func(c *Config) {
				c.Cache.MaxCacheSize.Overwrite(0)
			},
			wantErr: true,
		},
		{
			name: "invalid memory budget (too high)",
			modify: func(c *Config) {
				c.Cache.Memory.MemoryBudgetPercent.Overwrite(101)
			},
			wantErr: true,
		},
		{
			name: "invalid memory budget (negative)",
			modify: func(c *Config) {
				c.Cache.Memory.MemoryBudgetPercent.Overwrite(-1)
			},
			wantErr: true,
		},
		{
			name: "zero cleanup interval",
			modify: func(c *Config) {
				c.Cache.CleanupInterval.Overwrite(0)
			},
			wantErr: true,
		},
		{
			name: "zero cache lock shards",
			modify: func(c *Config) {
				c.Cache.LockShards.Overwrite(0)
			},
			wantErr: true,
		},
		{
			name: "zero hybrid demote age",
			modify: func(c *Config) {
				c.Cache.Hybrid.DemoteAfter.Overwrite(0)
			},
			wantErr: true,
		},
		{
			name: "empty cache dir",
			modify: func(c *Config) {
				c.Cache.File.Dir.Overwrite("")
			},
			wantErr: true,
		},
		{
			name: "empty database file",
			modify: func(c *Config) {
				c.Database.File.Overwrite("")
			},
			wantErr: true,
		},
		{
			name: "invalid cache type",
			modify: func(c *Config) {
				c.Cache.Type.Overwrite("invalid")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefault()
			tt.modify(cfg)
			err := cfg.verify()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadOrDefault_MigratesUpstreamDefaultHttps(t *testing.T) {
	tmpFile := filepath.ToSlash(filepath.Join(t.TempDir(), "config.json"))

	defaultData, err := json.Marshal(NewDefault())
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(defaultData, &raw); err != nil {
		t.Fatal(err)
	}
	proxy := raw["proxy"].(map[string]any)
	delete(proxy, "upgrade_http_to_https")
	proxy["upstream_default_https"] = true // The old default, saved in every existing config.

	oldConfigData, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpFile, oldConfigData, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadOrDefault(tmpFile)
	if err != nil {
		t.Fatalf("expected old config to be migrated: %v", err)
	}
	if cfg.Proxy.UpgradeHttpToHttps.Read() {
		t.Fatal("expected the old value not to turn on upgrading")
	}

	persisted, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(persisted), "upstream_default_https") {
		t.Fatal("expected the old key to be removed from the saved config")
	}
	if !strings.Contains(string(persisted), "upgrade_http_to_https") {
		t.Fatal("expected the new key in the saved config")
	}
}

func TestTrustedProxiesValidation(t *testing.T) {
	tests := []struct {
		value   string
		wantErr bool
	}{
		{"", false},
		{"10.0.0.1", false},
		{"10.0.0.0/8, 192.168.1.1", false},
		{"::1,fd00::/8", false},
		{"not-an-ip", true},
		{"10.0.0.0/99", true},
		{"10.0.0.1, nope", true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			cfg := NewDefault()
			cfg.Webserver.TrustedProxies.Overwrite(tt.value)
			if err := cfg.verify(); (err != nil) != tt.wantErr {
				t.Fatalf("verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTrustedProxyPrefixesParsesAddressesAndRanges(t *testing.T) {
	cfg := NewDefault()
	if got := cfg.Webserver.TrustedProxyPrefixes(); len(got) != 0 {
		t.Fatalf("expected nothing trusted by default, got %v", got)
	}

	cfg.Webserver.TrustedProxies.Overwrite("10.0.0.0/8, 192.168.1.1")
	prefixes := cfg.Webserver.TrustedProxyPrefixes()
	if len(prefixes) != 2 {
		t.Fatalf("expected 2 prefixes, got %v", prefixes)
	}
	if !prefixes[0].Contains(netip.MustParseAddr("10.200.0.1")) {
		t.Fatalf("expected %v to contain 10.200.0.1", prefixes[0])
	}
	if !prefixes[1].Contains(netip.MustParseAddr("192.168.1.1")) || prefixes[1].Contains(netip.MustParseAddr("192.168.1.2")) {
		t.Fatalf("expected %v to contain only 192.168.1.1", prefixes[1])
	}
}

func TestLoadOrDefault_AddsMissingTrustedProxies(t *testing.T) {
	tmpFile := filepath.ToSlash(filepath.Join(t.TempDir(), "config.json"))

	defaultData, err := json.Marshal(NewDefault())
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(defaultData, &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw["webserver"].(map[string]any), "trusted_proxies")

	oldConfigData, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpFile, oldConfigData, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadOrDefault(tmpFile)
	if err != nil {
		t.Fatalf("expected old config to be migrated: %v", err)
	}
	if got := cfg.Webserver.TrustedProxies.Read(); got != "" {
		t.Fatalf("expected nothing trusted after migration, got %q", got)
	}
}

func TestPersistFailureLeavesExistingConfigIntact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := NewDefault()
	cfg.path = path
	if err := cfg.persist(); err != nil {
		t.Fatalf("initial persist failed: %v", err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// A directory at the temp path makes the next write fail before it can touch the config.
	if err := os.Mkdir(path+".tmp", 0o755); err != nil {
		t.Fatal(err)
	}
	cfg.Proxy.Listen.Overwrite(":1234")
	if err := cfg.persist(); err == nil {
		t.Fatal("expected persist to fail")
	}

	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(current) != string(original) {
		t.Fatal("failed persist changed the existing config")
	}
}

func TestPersistLeavesNoTempFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := NewDefault()
	cfg.path = path
	if err := cfg.persist(); err != nil {
		t.Fatalf("persist failed: %v", err)
	}

	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no temp file, stat err=%v", err)
	}
	if _, err := load(path); err != nil {
		t.Fatalf("persisted config does not load: %v", err)
	}
}
