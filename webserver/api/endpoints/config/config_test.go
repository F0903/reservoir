package config

import (
	"net/http"
	"net/http/httptest"
	appservices "reservoir/application/services"
	coreconfig "reservoir/config"
	"reservoir/webserver/api/apitypes"
	"strings"
	"testing"
)

func newPatchRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPatch, "/api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestPatchReturnsBadRequest(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantBody []string
	}{
		{
			name:     "ValidationError",
			body:     `{"cache":{"max_cache_size":"0B"}}`,
			wantBody: []string{coreconfig.ErrValidationFailed.Error()},
		},
		{
			name:     "UnknownKey",
			body:     `{"cache":{"file":{"missing":"value"}}}`,
			wantBody: []string{coreconfig.ErrValidationFailed.Error(), "cache.file.missing"},
		},
		{
			name:     "NullBody",
			body:     `null`,
			wantBody: []string{coreconfig.ErrValidationFailed.Error()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := coreconfig.NewDefault()
			originalSize := cfg.Cache.MaxCacheSize.Read()
			originalDir := cfg.Cache.File.Dir.Read()
			rec := httptest.NewRecorder()

			(&ConfigEndpoint{}).Patch(
				rec,
				newPatchRequest(tt.body),
				apitypes.Context{Services: appservices.NewContainer(cfg, nil, nil, nil)},
			)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
			for _, want := range tt.wantBody {
				if !strings.Contains(rec.Body.String(), want) {
					t.Fatalf("expected %q in body, got %q", want, rec.Body.String())
				}
			}
			if got := cfg.Cache.MaxCacheSize.Read(); got != originalSize {
				t.Fatalf("invalid update committed max cache size: got %s, want %s", got, originalSize)
			}
			if got := cfg.Cache.File.Dir.Read(); got != originalDir {
				t.Fatalf("invalid update committed cache dir: got %q, want %q", got, originalDir)
			}
		})
	}
}
