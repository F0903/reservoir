package log

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	appservices "reservoir/application/services"
	"reservoir/config"
	"reservoir/webserver/api/apitypes"
	"testing"
)

func TestLogEndpointServesTailWithoutCaching(t *testing.T) {
	logFile := filepath.ToSlash(filepath.Join(t.TempDir(), "proxy.log"))
	if err := os.WriteFile(logFile, []byte("first line\nlast line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.NewDefault()
	cfg.Logging.File.Overwrite(logFile)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/log", nil)
	req.Header.Set("Range", "bytes=-10") // The dashboard asks for the end of the file only.

	(&LogEndpoint{}).Get(rec, req, apitypes.Context{Services: appservices.NewContainer(cfg, nil, nil, nil)})

	if rec.Code != http.StatusPartialContent || rec.Body.String() != "last line\n" {
		t.Fatalf("expected the last 10 bytes, got status %d: %q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected Cache-Control no-store on the response, got %q", got)
	}
}
