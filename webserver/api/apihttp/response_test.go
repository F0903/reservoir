package apihttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsJSONContentType(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "plain json", raw: "application/json", want: true},
		{name: "json with charset", raw: "application/json; charset=utf-8", want: true},
		{name: "case insensitive", raw: "Application/JSON", want: true},
		{name: "structured suffix", raw: "application/vnd.api+json", want: true},
		{name: "empty", raw: "", want: false},
		{name: "text json", raw: "text/json", want: false},
		{name: "non json application", raw: "application/not-json", want: false},
		{name: "invalid parameter", raw: "application/json; charset", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsJSONContentType(tt.raw); got != tt.want {
				t.Fatalf("IsJSONContentType(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRequireJSONContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/api/config", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()

	if !RequireJSONContentType(rec, req) {
		t.Fatal("expected JSON content type to be accepted")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected response recorder to remain unwritten, got status %d", rec.Code)
	}
}

func TestRequireJSONContentTypeRejectsInvalidMediaType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/api/config", nil)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	if RequireJSONContentType(rec, req) {
		t.Fatal("expected non-JSON content type to be rejected")
	}
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d", http.StatusUnsupportedMediaType, rec.Code)
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	if !WriteJSON(rec, http.StatusCreated, map[string]string{"status": "ok"}) {
		t.Fatal("expected JSON write to succeed")
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != JSONContentType {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	if got := rec.Body.String(); got != `{"status":"ok"}` {
		t.Fatalf("unexpected body %q", got)
	}
}

func TestDecodeJSONRejectsInvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	var payload map[string]any
	if DecodeJSON(rec, req, &payload) {
		t.Fatal("expected invalid JSON to be rejected")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
