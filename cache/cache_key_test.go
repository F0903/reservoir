package cache

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMakeFromRequestSeparatesHTTPAndHTTPS(t *testing.T) {
	httpReq := httptest.NewRequest(http.MethodGet, "http://example.test/package.deb", nil)
	httpsReq := httptest.NewRequest(http.MethodGet, "/package.deb", nil)
	httpsReq.Host = "example.test"
	httpsReq.TLS = &tls.ConnectionState{}

	if MakeFromRequest(httpReq) == MakeFromRequest(httpsReq) {
		t.Fatal("expected HTTP and tunneled HTTPS requests to use different cache keys")
	}
}

func TestMakeFromRequestUsesAbsoluteURLScheme(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.test/package.deb", nil)
	req.TLS = nil

	if got, want := MakeFromRequest(req), FromString("https|GET|example.test|/package.deb|"); got != want {
		t.Fatalf("expected cache key to use absolute URL scheme, got %s want %s", got.Hex, want.Hex)
	}
}
