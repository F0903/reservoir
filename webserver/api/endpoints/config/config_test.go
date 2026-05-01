package config

import (
	"net/http"
	"net/http/httptest"
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
		apitypes.Context{Config: cfg},
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

func TestPatchReturnsBadRequestForNullBody(t *testing.T) {
	rec := httptest.NewRecorder()

	(&ConfigEndpoint{}).Patch(
		rec,
		newPatchRequest(`null`),
		apitypes.Context{Config: coreconfig.NewDefault()},
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), coreconfig.ErrValidationFailed.Error()) {
		t.Fatalf("expected validation error body, got %q", rec.Body.String())
	}
}
