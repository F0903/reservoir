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

func TestPatchReturnsBadRequestForValidationError(t *testing.T) {
	cfg := coreconfig.NewDefault()
	originalSize := cfg.Cache.MaxCacheSize.Read()
	rec := httptest.NewRecorder()

	(&ConfigEndpoint{}).Patch(
		rec,
		newPatchRequest(`{"cache":{"max_cache_size":"0B"}}`),
		apitypes.Context{Services: appservices.NewContainer(cfg, nil, nil, nil)},
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), coreconfig.ErrValidationFailed.Error()) {
		t.Fatalf("expected validation error body, got %q", rec.Body.String())
	}
	if got := cfg.Cache.MaxCacheSize.Read(); got != originalSize {
		t.Fatalf("invalid update committed max cache size: got %s, want %s", got, originalSize)
	}
}

func TestPatchReturnsBadRequestForUnknownKey(t *testing.T) {
	cfg := coreconfig.NewDefault()
	originalDir := cfg.Cache.File.Dir.Read()
	rec := httptest.NewRecorder()

	(&ConfigEndpoint{}).Patch(
		rec,
		newPatchRequest(`{"cache":{"file":{"missing":"value"}}}`),
		apitypes.Context{Services: appservices.NewContainer(cfg, nil, nil, nil)},
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), coreconfig.ErrValidationFailed.Error()) {
		t.Fatalf("expected validation error body, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "cache.file.missing") {
		t.Fatalf("expected unknown key path in body, got %q", rec.Body.String())
	}
	if got := cfg.Cache.File.Dir.Read(); got != originalDir {
		t.Fatalf("unknown update committed cache dir: got %q, want %q", got, originalDir)
	}
}

func TestPatchReturnsBadRequestForNullBody(t *testing.T) {
	rec := httptest.NewRecorder()

	(&ConfigEndpoint{}).Patch(
		rec,
		newPatchRequest(`null`),
		apitypes.Context{Services: appservices.NewContainer(coreconfig.NewDefault(), nil, nil, nil)},
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), coreconfig.ErrValidationFailed.Error()) {
		t.Fatalf("expected validation error body, got %q", rec.Body.String())
	}
}
