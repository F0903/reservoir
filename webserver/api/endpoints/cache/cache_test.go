package cache

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	cachecore "reservoir/cache"
	"reservoir/config"
	"reservoir/webserver/api/apitypes"
	"testing"
)

type fakeCacheController struct {
	stats       cachecore.Stats
	clearErr    error
	clearCalled bool
}

func TestEndpointAdminRequirements(t *testing.T) {
	tests := []struct {
		name              string
		method            apitypes.EndpointMethod
		wantRequiresAdmin bool
	}{
		{
			name:              "status read",
			method:            (&StatusEndpoint{}).EndpointMethods()[0],
			wantRequiresAdmin: false,
		},
		{
			name:              "clear mutation",
			method:            (&ClearEndpoint{}).EndpointMethods()[0],
			wantRequiresAdmin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.method.RequiresAuth {
				t.Fatal("expected cache endpoint to require authentication")
			}
			if tt.method.RequiresAdmin != tt.wantRequiresAdmin {
				t.Fatalf("expected RequiresAdmin=%t, got %t", tt.wantRequiresAdmin, tt.method.RequiresAdmin)
			}
		})
	}
}

func (f *fakeCacheController) CacheStats() cachecore.Stats {
	return f.stats
}

func (f *fakeCacheController) ClearCache() error {
	f.clearCalled = true
	return f.clearErr
}

func decodeJSONResponse(t *testing.T, rec *httptest.ResponseRecorder, value any) bool {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), value); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rec.Body.String(), err)
	}
	return true
}

func TestStatusEndpointReturnsCacheStats(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Cache.Type.Overwrite(config.CacheTypeMemory)

	controller := &fakeCacheController{
		stats: cachecore.Stats{
			Entries:        3,
			Bytes:          128,
			MaxBytes:       1024,
			MemoryCapBytes: 2048,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/cache/status", nil)

	(&StatusEndpoint{}).Get(rec, req, apitypes.Context{
		Config: cfg,
		Cache:  controller,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp statusResponse
	if !decodeJSONResponse(t, rec, &resp) {
		return
	}

	if resp.Type != config.CacheTypeMemory {
		t.Fatalf("expected cache type %q, got %q", config.CacheTypeMemory, resp.Type)
	}
	if resp.Entries != 3 {
		t.Fatalf("expected 3 entries, got %d", resp.Entries)
	}
	if resp.Bytes != 128 {
		t.Fatalf("expected 128 bytes, got %d", resp.Bytes)
	}
	if resp.MaxBytes != 1024 {
		t.Fatalf("expected max bytes 1024, got %d", resp.MaxBytes)
	}
	if resp.MemoryCapBytes == nil || *resp.MemoryCapBytes != 2048 {
		t.Fatalf("expected memory cap 2048, got %v", resp.MemoryCapBytes)
	}
}

func TestStatusEndpointReturnsMemoryCapForHybridCache(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Cache.Type.Overwrite(config.CacheTypeHybrid)

	controller := &fakeCacheController{
		stats: cachecore.Stats{
			Entries:        3,
			Bytes:          128,
			MaxBytes:       1024,
			MemoryCapBytes: 2048,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/cache/status", nil)

	(&StatusEndpoint{}).Get(rec, req, apitypes.Context{
		Config: cfg,
		Cache:  controller,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp statusResponse
	if !decodeJSONResponse(t, rec, &resp) {
		return
	}

	if resp.Type != config.CacheTypeHybrid {
		t.Fatalf("expected cache type %q, got %q", config.CacheTypeHybrid, resp.Type)
	}
	if resp.MemoryCapBytes == nil || *resp.MemoryCapBytes != 2048 {
		t.Fatalf("expected memory cap 2048, got %v", resp.MemoryCapBytes)
	}
}

func TestStatusEndpointOmitsMemoryCapForFileCache(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Cache.Type.Overwrite(config.CacheTypeFile)

	controller := &fakeCacheController{
		stats: cachecore.Stats{
			Entries:        1,
			Bytes:          64,
			MaxBytes:       1024,
			MemoryCapBytes: 2048,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/cache/status", nil)

	(&StatusEndpoint{}).Get(rec, req, apitypes.Context{
		Config: cfg,
		Cache:  controller,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp statusResponse
	if !decodeJSONResponse(t, rec, &resp) {
		return
	}

	if resp.MemoryCapBytes != nil {
		t.Fatalf("expected memory cap to be omitted for file cache, got %v", *resp.MemoryCapBytes)
	}
}

func TestClearEndpointClearsCache(t *testing.T) {
	controller := &fakeCacheController{}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/cache/clear", nil)

	(&ClearEndpoint{}).Post(rec, req, apitypes.Context{Cache: controller})

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if !controller.clearCalled {
		t.Fatal("expected cache clear to be called")
	}
}

func TestClearEndpointReturnsErrorWhenClearFails(t *testing.T) {
	controller := &fakeCacheController{clearErr: errors.New("clear failed")}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/cache/clear", nil)

	(&ClearEndpoint{}).Post(rec, req, apitypes.Context{Cache: controller})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if !controller.clearCalled {
		t.Fatal("expected cache clear to be called")
	}
}
