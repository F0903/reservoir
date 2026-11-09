package cache

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	appservices "reservoir/application/services"
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

func TestStatusEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		cacheType     config.CacheType
		stats         cachecore.Stats
		wantMemoryCap bool
	}{
		{
			name:          "ReturnsCacheStats",
			cacheType:     config.CacheTypeMemory,
			stats:         cachecore.Stats{Entries: 3, Bytes: 128, MaxBytes: 1024, MemoryCapBytes: 2048},
			wantMemoryCap: true,
		},
		{
			name:          "ReturnsMemoryCapForHybridCache",
			cacheType:     config.CacheTypeHybrid,
			stats:         cachecore.Stats{Entries: 3, Bytes: 128, MaxBytes: 1024, MemoryCapBytes: 2048},
			wantMemoryCap: true,
		},
		{
			name:      "OmitsMemoryCapForFileCache",
			cacheType: config.CacheTypeFile,
			stats:     cachecore.Stats{Entries: 1, Bytes: 64, MaxBytes: 1024, MemoryCapBytes: 2048},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefault()
			cfg.Cache.Type.Overwrite(tt.cacheType)
			controller := &fakeCacheController{stats: tt.stats}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/cache/status", nil)

			(&StatusEndpoint{}).Get(rec, req, apitypes.Context{
				Services: appservices.NewContainer(cfg, nil, nil, controller),
			})

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var resp statusResponse
			if !decodeJSONResponse(t, rec, &resp) {
				return
			}

			if resp.Type != string(tt.cacheType) {
				t.Fatalf("expected cache type %q, got %q", tt.cacheType, resp.Type)
			}
			if resp.Entries != tt.stats.Entries {
				t.Fatalf("expected %d entries, got %d", tt.stats.Entries, resp.Entries)
			}
			if resp.Bytes != tt.stats.Bytes {
				t.Fatalf("expected %d bytes, got %d", tt.stats.Bytes, resp.Bytes)
			}
			if resp.MaxBytes != tt.stats.MaxBytes {
				t.Fatalf("expected max bytes %d, got %d", tt.stats.MaxBytes, resp.MaxBytes)
			}
			if !tt.wantMemoryCap {
				if resp.MemoryCapBytes != nil {
					t.Fatalf("expected memory cap to be omitted, got %v", *resp.MemoryCapBytes)
				}
				return
			}
			if resp.MemoryCapBytes == nil || *resp.MemoryCapBytes != tt.stats.MemoryCapBytes {
				t.Fatalf("expected memory cap %d, got %v", tt.stats.MemoryCapBytes, resp.MemoryCapBytes)
			}
		})
	}
}

func TestClearEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		clearErr   error
		wantStatus int
	}{
		{name: "ClearsCache", wantStatus: http.StatusNoContent},
		{name: "ReturnsErrorWhenClearFails", clearErr: errors.New("clear failed"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &fakeCacheController{clearErr: tt.clearErr}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/cache/clear", nil)

			(&ClearEndpoint{}).Post(rec, req, apitypes.Context{
				Services: appservices.NewContainer(config.NewDefault(), nil, nil, controller),
			})

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if !controller.clearCalled {
				t.Fatal("expected cache clear to be called")
			}
		})
	}
}
