package proxy

import (
	"net/http"
	"reservoir/config"
	"reservoir/proxy/headers"
	"testing"
)

func decideForTest(req *http.Request, resp *http.Response, cfg *config.Config) cacheDecision {
	upstreamHd := headers.ParseHeaderDirective(resp.Header)
	return newCachePolicy(cfg).Decide(req, resp, upstreamHd)
}

func TestCachePolicyAllowsAggressivePackageCacheNoStore(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)
	cfg.Proxy.CachePolicy.ForceDefaultMaxAge.Overwrite(true)

	req := httptestRequest(t)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Cache-Control": []string{"no-store"},
		},
	}

	decision := decideForTest(req, resp, cfg)
	if !decision.Cacheable {
		t.Fatalf("expected aggressive package-cache policy to allow no-store response, reason: %s", decision.Reason)
	}
}

func TestCachePolicyRejectsCredentialedRequests(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)

	req := httptestRequest(t)
	req.Header.Set("Authorization", "Bearer token")
	resp := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header)}

	decision := decideForTest(req, resp, cfg)
	if decision.Cacheable {
		t.Fatal("expected credentialed request to be uncacheable")
	}
}

func TestCachePolicyRejectsSetCookieResponses(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)

	req := httptestRequest(t)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Set-Cookie": []string{"sid=abc"},
		},
	}

	decision := decideForTest(req, resp, cfg)
	if decision.Cacheable {
		t.Fatal("expected Set-Cookie response to be uncacheable")
	}
}

func TestCachePolicyAllowsAcceptEncodingVary(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)

	req := httptestRequest(t)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Vary": []string{"Accept-Encoding"},
		},
	}

	decision := decideForTest(req, resp, cfg)
	if !decision.Cacheable {
		t.Fatalf("expected Accept-Encoding Vary response to be cacheable, reason: %s", decision.Reason)
	}
}

func TestCachePolicyRejectsUnsupportedVary(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)

	req := httptestRequest(t)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Vary": []string{"User-Agent"},
		},
	}

	decision := decideForTest(req, resp, cfg)
	if decision.Cacheable {
		t.Fatal("expected unsupported Vary response to be uncacheable")
	}
}

func httptestRequest(t *testing.T) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, "http://example.test/package.deb", nil)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	return req
}
