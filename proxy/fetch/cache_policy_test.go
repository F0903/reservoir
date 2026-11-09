package fetch

import (
	"maps"
	"net/http"
	"reservoir/config"
	"reservoir/proxy/headers"
	"testing"
)

func decideForTest(req *http.Request, resp *http.Response, cfg *config.Config) cacheDecision {
	upstreamHd := headers.ParseHeaderDirective(resp.Header)
	return newCachePolicy(cfg).Decide(req, resp, upstreamHd)
}

func TestCachePolicyDecide(t *testing.T) {
	tests := []struct {
		name           string
		honorCC        bool // Every other row runs with IgnoreCacheControl on.
		method         string
		requestHeader  http.Header
		responseHeader http.Header
		wantCacheable  bool
	}{
		{name: "authorization request", requestHeader: http.Header{"Authorization": {"Bearer token"}}},
		{name: "cookie request", requestHeader: http.Header{"Cookie": {"sid=abc"}}},
		{name: "private-token request", requestHeader: http.Header{"Private-Token": {"secret"}}},
		{name: "x-api-key request", requestHeader: http.Header{"X-Api-Key": {"secret"}}},
		{name: "set-cookie response", responseHeader: http.Header{"Set-Cookie": {"sid=abc"}}},
		{name: "post request", method: http.MethodPost},
		{name: "no-store when honoring cache-control", honorCC: true, responseHeader: http.Header{"Cache-Control": {"no-store"}}},
		{name: "vary star", responseHeader: http.Header{"Vary": {"*"}}},
		{name: "unsupported vary", responseHeader: http.Header{"Vary": {"User-Agent"}}},
		{name: "encoded without accept-encoding vary", responseHeader: http.Header{"Content-Encoding": {"gzip"}}},
		{name: "encoded with accept-encoding vary", responseHeader: http.Header{"Content-Encoding": {"gzip"}, "Vary": {"Accept-Encoding"}}, wantCacheable: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefault()
			cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(!tt.honorCC)

			req, err := http.NewRequest(tt.method, "http://example.test/package.deb", nil)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			maps.Copy(req.Header, tt.requestHeader)
			resp := &http.Response{StatusCode: http.StatusOK, Header: tt.responseHeader}

			decision := decideForTest(req, resp, cfg)
			if decision.Cacheable != tt.wantCacheable {
				t.Fatalf("Cacheable = %v, want %v (reason: %s)", decision.Cacheable, tt.wantCacheable, decision.Reason)
			}
		})
	}
}
