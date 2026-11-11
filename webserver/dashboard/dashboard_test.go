package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDirectoryRequestServesIndex(t *testing.T) {
	rec := httptest.NewRecorder()
	// _app is a directory in the embedded build: it opens, but it is not a seekable file.
	req := httptest.NewRequest(http.MethodGet, "/_app", nil)

	New(nil).ServeDashboard(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("expected index.html, got status %d", rec.Code)
	}
}
