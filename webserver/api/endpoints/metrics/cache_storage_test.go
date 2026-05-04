package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	cachecore "reservoir/cache"
	"reservoir/config"
	runtimeMetrics "reservoir/metrics"
	"reservoir/webserver/api/apitypes"
	"testing"
)

type fakeCacheController struct {
	stats cachecore.Stats
}

func (f *fakeCacheController) CacheStats() cachecore.Stats {
	return f.stats
}

func (f *fakeCacheController) ClearCache() error {
	return nil
}

func useFreshMetrics(t *testing.T) {
	t.Helper()

	previous := runtimeMetrics.Global
	runtimeMetrics.Global = runtimeMetrics.NewMetrics()
	t.Cleanup(func() {
		runtimeMetrics.Global = previous
	})
}

func TestCacheMetricsEndpointIncludesCacheStorage(t *testing.T) {
	useFreshMetrics(t)

	cfg := config.NewDefault()
	cfg.Cache.Type.Overwrite(config.CacheTypeMemory)
	controller := &fakeCacheController{
		stats: cachecore.Stats{
			Entries:        4,
			Bytes:          2048,
			MaxBytes:       4096,
			MemoryCapBytes: 8192,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/metrics/cache", nil)

	(&CacheMetricsEndpoint{}).Get(rec, req, apitypes.Context{
		Config: cfg,
		Cache:  controller,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp struct {
		Storage runtimeMetrics.CacheStorageMetrics `json:"storage"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rec.Body.String(), err)
	}

	if resp.Storage.Type != string(config.CacheTypeMemory) {
		t.Fatalf("expected cache type %q, got %q", config.CacheTypeMemory, resp.Storage.Type)
	}
	if resp.Storage.Entries != 4 {
		t.Fatalf("expected 4 entries, got %d", resp.Storage.Entries)
	}
	if resp.Storage.Bytes != 2048 {
		t.Fatalf("expected 2048 bytes, got %d", resp.Storage.Bytes)
	}
	if resp.Storage.MaxBytes != 4096 {
		t.Fatalf("expected max bytes 4096, got %d", resp.Storage.MaxBytes)
	}
	if resp.Storage.MemoryCapBytes == nil || *resp.Storage.MemoryCapBytes != 8192 {
		t.Fatalf("expected memory cap 8192, got %v", resp.Storage.MemoryCapBytes)
	}
}

func TestCacheMetricsEndpointIncludesHybridCacheStorage(t *testing.T) {
	useFreshMetrics(t)

	cfg := config.NewDefault()
	cfg.Cache.Type.Overwrite(config.CacheTypeHybrid)
	controller := &fakeCacheController{
		stats: cachecore.Stats{
			Entries:        4,
			Bytes:          2048,
			MaxBytes:       4096,
			MemoryCapBytes: 8192,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/metrics/cache", nil)

	(&CacheMetricsEndpoint{}).Get(rec, req, apitypes.Context{
		Config: cfg,
		Cache:  controller,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp struct {
		Storage runtimeMetrics.CacheStorageMetrics `json:"storage"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rec.Body.String(), err)
	}

	if resp.Storage.Type != string(config.CacheTypeHybrid) {
		t.Fatalf("expected cache type %q, got %q", config.CacheTypeHybrid, resp.Storage.Type)
	}
	if resp.Storage.MemoryCapBytes == nil || *resp.Storage.MemoryCapBytes != 8192 {
		t.Fatalf("expected memory cap 8192, got %v", resp.Storage.MemoryCapBytes)
	}
}

func TestAllMetricsEndpointIncludesFileCacheStorage(t *testing.T) {
	useFreshMetrics(t)

	cfg := config.NewDefault()
	cfg.Cache.Type.Overwrite(config.CacheTypeFile)
	controller := &fakeCacheController{
		stats: cachecore.Stats{
			Entries:        2,
			Bytes:          512,
			MaxBytes:       1024,
			MemoryCapBytes: 2048,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)

	(&AllMetricsEndpoint{}).Get(rec, req, apitypes.Context{
		Config: cfg,
		Cache:  controller,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp struct {
		Cache struct {
			Storage runtimeMetrics.CacheStorageMetrics `json:"storage"`
		} `json:"cache"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rec.Body.String(), err)
	}

	if resp.Cache.Storage.Type != string(config.CacheTypeFile) {
		t.Fatalf("expected cache type %q, got %q", config.CacheTypeFile, resp.Cache.Storage.Type)
	}
	if resp.Cache.Storage.Entries != 2 {
		t.Fatalf("expected 2 entries, got %d", resp.Cache.Storage.Entries)
	}
	if resp.Cache.Storage.MemoryCapBytes != nil {
		t.Fatalf("expected memory cap to be omitted for file cache, got %v", *resp.Cache.Storage.MemoryCapBytes)
	}
}
