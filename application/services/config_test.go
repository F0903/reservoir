package services

import (
	"log/slog"
	"path/filepath"
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
