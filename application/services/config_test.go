package services

import (
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"reservoir/config"
)

func newConfigServiceTestConfig(t *testing.T) *config.Config {
	t.Helper()

	cfg, err := config.LoadOrDefault(filepath.ToSlash(filepath.Join(t.TempDir(), "config.json")))
	if err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}
	return cfg
}

func TestConfigServiceUpdatePartialRejectsUnknownKey(t *testing.T) {
	cfg := newConfigServiceTestConfig(t)
	svc := &ConfigService{cfg: cfg}
	originalDir := cfg.Cache.File.Dir.Read()

	status, err := svc.UpdatePartial(map[string]any{
		"cache": map[string]any{
			"file": map[string]any{
				"missing": "value",
			},
		},
	})
	if err == nil {
		t.Fatal("expected unknown config key to return an error")
	}
	if !errors.Is(err, ErrConfigValidationFailed) {
		t.Fatalf("expected service validation error, got %v", err)
	}
	if !strings.Contains(err.Error(), "cache.file.missing") {
		t.Fatalf("expected error to include full key path, got %v", err)
	}
	if status != ConfigUpdateSuccess {
		t.Fatalf("expected unsuccessful validation to return success status placeholder, got %v", status)
	}
	if got := cfg.Cache.File.Dir.Read(); got != originalDir {
		t.Fatalf("unknown update committed cache dir: got %q, want %q", got, originalDir)
	}
	if svc.RestartRequired() {
		t.Fatal("unknown update should not mark restart required")
	}
}

func TestConfigServiceTracksRestartRequired(t *testing.T) {
	cfg := newConfigServiceTestConfig(t)
	svc := &ConfigService{cfg: cfg}

	status, err := svc.UpdatePartial(map[string]any{
		"logging": map[string]any{
			"level": "DEBUG",
		},
	})
	if err != nil {
		t.Fatalf("hot-swap update failed: %v", err)
	}
	if status != ConfigUpdateSuccess {
		t.Fatalf("expected hot-swap update success, got %v", status)
	}
	if svc.RestartRequired() {
		t.Fatal("hot-swap update should not mark restart required")
	}
	if got := cfg.Logging.Level.Read(); got != slog.LevelDebug {
		t.Fatalf("expected logging level DEBUG, got %v", got)
	}

	status, err = svc.UpdatePartial(map[string]any{
		"proxy": map[string]any{
			"listen": ":10000",
		},
	})
	if err != nil {
		t.Fatalf("restart-required update failed: %v", err)
	}
	if status != ConfigUpdateRestartRequired {
		t.Fatalf("expected restart-required status, got %v", status)
	}
	if !svc.RestartRequired() {
		t.Fatal("restart-required update should mark restart required")
	}

	status, err = svc.UpdatePartial(map[string]any{
		"logging": map[string]any{
			"level": "INFO",
		},
	})
	if err != nil {
		t.Fatalf("second hot-swap update failed: %v", err)
	}
	if status != ConfigUpdateRestartRequired {
		t.Fatalf("expected restart-required status to stay sticky, got %v", status)
	}
}

func TestConfigServiceSnapshotReturnsDetachedConfig(t *testing.T) {
	cfg := newConfigServiceTestConfig(t)
	svc := &ConfigService{cfg: cfg}

	snapshot, err := svc.Snapshot()
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	snapshot.Proxy.Listen.Overwrite(":snapshot")
	if got := cfg.Proxy.Listen.Read(); got == ":snapshot" {
		t.Fatalf("snapshot mutation changed live config: got %q", got)
	}
}
