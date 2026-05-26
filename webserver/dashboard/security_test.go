package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"reservoir/webserver/dashboard/csp"
)

func TestSecureSetsDashboardSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)

	headers := rec.Header()
	if got := headers.Get("Content-Security-Policy"); got != csp.Header {
		t.Fatalf("expected dashboard CSP header, got %q", got)
	}
	if got := headers.Get("Cross-Origin-Opener-Policy"); got != "same-origin" {
		t.Fatalf("expected Cross-Origin-Opener-Policy same-origin, got %q", got)
	}
	if got := headers.Get("Cross-Origin-Embedder-Policy"); got != "require-corp" {
		t.Fatalf("expected Cross-Origin-Embedder-Policy require-corp, got %q", got)
	}
	if got := headers.Get("Cross-Origin-Resource-Policy"); got != "same-origin" {
		t.Fatalf("expected Cross-Origin-Resource-Policy same-origin, got %q", got)
	}
}
