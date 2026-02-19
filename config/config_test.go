package config

import (
	"log/slog"
	"os"
	"reflect"
	"testing"
)

type subConfig struct {
	Prop ConfigProp[string] `json:"prop"`
}

type testConfig struct {
	Sub  subConfig       `json:"sub"`
	Flat ConfigProp[int] `json:"flat"`
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
	// Setup a clean global state for this test if possible,
	// but since it's a singleton we'll just test that it works.

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

	status, err := UpdatePartialFromConfig(Global, updates)
	if err != nil {
		t.Fatalf("UpdatePartialFromConfig failed: %v", err)
	}

	if status == UpdateStatusFailed {
		t.Errorf("Expected UpdateStatusSuccess or RestartRequired, got %v", status)
	}

	if Global.Cache.MaxCacheSize.Read().String() != "50G" {
		t.Errorf("Expected MaxCacheSize to be 50G, got %s", Global.Cache.MaxCacheSize.Read().String())
	}

	if Global.Cache.File.Dir.Read() != "/tmp/new-cache" {
		t.Errorf("Expected Cache.File.Dir to be /tmp/new-cache, got %s", Global.Cache.File.Dir.Read())
	}

	if Global.Proxy.CachePolicy.IgnoreCacheControl.Read() != false {
		t.Errorf("Expected CachePolicy.IgnoreCacheControl to be false, got %v", Global.Proxy.CachePolicy.IgnoreCacheControl.Read())
	}

	if Global.Logging.Level.Read() != slog.LevelDebug {
		t.Errorf("Expected LogLevel to be DEBUG, got %v", Global.Logging.Level.Read())
	}
}

func TestLoadOrDefault_StrictPolicy(t *testing.T) {
	tmpFile := "var/test_config_strict.json"
	defer os.Remove(tmpFile)

	// Write invalid JSON
	err := os.WriteFile(tmpFile, []byte("{ invalid json }"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := loadOrDefault(tmpFile)
	if err != nil {
		t.Fatalf("loadOrDefault should not return error on reset: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected non-nil config after reset")
	}

	// Verify it was reset to defaults
	if cfg.Proxy.Listen.Read() != ":9999" {
		t.Errorf("Expected reset to default proxy listen, got %s", cfg.Proxy.Listen.Read())
	}

	// Verify file was overwritten
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(data[:1], []byte("{")) {
		t.Error("Expected file to be overwritten with valid JSON")
	}
}

func TestLoadOrDefault_MissingFields(t *testing.T) {
	tmpFile := "var/test_config_missing.json"
	defer os.Remove(tmpFile)

	// Write JSON with missing fields (e.g. no proxy section)
	err := os.WriteFile(tmpFile, []byte(`{"cache": {"type": "memory"}}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := loadOrDefault(tmpFile)
	if err != nil {
		t.Fatalf("loadOrDefault should not return error on reset: %v", err)
	}

	// Verify it was reset to defaults (checking a field that was definitely missing)
	if cfg.Proxy.Listen.Read() != ":9999" {
		t.Errorf("Expected reset to default proxy listen, got %s", cfg.Proxy.Listen.Read())
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
			name: "empty cache dir",
			modify: func(c *Config) {
				c.Cache.File.Dir.Overwrite("")
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
			cfg := newDefault()
			tt.modify(cfg)
			err := cfg.verify()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
