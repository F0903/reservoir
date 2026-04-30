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

func TestCachePolicyRejectsCookieRequests(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)

	req := httptestRequest(t)
	req.Header.Set("Cookie", "sid=abc")
	resp := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header)}

	decision := decideForTest(req, resp, cfg)
	if decision.Cacheable {
		t.Fatal("expected request with Cookie header to be uncacheable")
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

func TestCachePolicyRejectsNonGetMethods(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)

	req, err := http.NewRequest(http.MethodPost, "http://example.test/package.deb", nil)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	resp := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header)}

	decision := decideForTest(req, resp, cfg)
	if decision.Cacheable {
		t.Fatal("expected POST response to be uncacheable")
	}
}

func TestCachePolicyRejectsNonOKStatuses(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)

	tests := []struct {
		name   string
		status int
	}{
		{name: "partial-content", status: http.StatusPartialContent},
		{name: "not-found", status: http.StatusNotFound},
		{name: "not-modified", status: http.StatusNotModified},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptestRequest(t)
			resp := &http.Response{StatusCode: tt.status, Header: make(http.Header)}

			decision := decideForTest(req, resp, cfg)
			if decision.Cacheable {
				t.Fatalf("expected status %d response to be uncacheable", tt.status)
			}
		})
	}
}

func TestCachePolicyRejectsNoStoreWhenNotIgnoringCacheControl(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(false)

	req := httptestRequest(t)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Cache-Control": []string{"no-store"},
		},
	}

	decision := decideForTest(req, resp, cfg)
	if decision.Cacheable {
		t.Fatal("expected no-store response to be uncacheable when cache-control is honored")
	}
}

func TestCachePolicyRejectsPrivateWhenNotIgnoringCacheControl(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(false)

	req := httptestRequest(t)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Cache-Control": []string{"private, max-age=60"},
		},
	}

	decision := decideForTest(req, resp, cfg)
	if decision.Cacheable {
		t.Fatal("expected private response to be uncacheable when cache-control is honored")
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

func TestCachePolicyRejectsVaryStar(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)

	req := httptestRequest(t)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Vary": []string{"*"},
		},
	}

	decision := decideForTest(req, resp, cfg)
	if decision.Cacheable {
		t.Fatal("expected Vary: * response to be uncacheable")
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

func TestCachePolicyRejectsEncodedResponseWithoutAcceptEncodingVary(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)

	req := httptestRequest(t)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Encoding": []string{"gzip"},
		},
	}

	decision := decideForTest(req, resp, cfg)
	if decision.Cacheable {
		t.Fatal("expected encoded response without Vary: Accept-Encoding to be uncacheable")
	}
}

func TestCachePolicyAllowsEncodedResponseWithAcceptEncodingVary(t *testing.T) {
	cfg := config.NewDefault()
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)

	req := httptestRequest(t)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Encoding": []string{"gzip"},
			"Vary":             []string{"Accept-Encoding"},
		},
	}

	decision := decideForTest(req, resp, cfg)
	if !decision.Cacheable {
		t.Fatalf("expected encoded response with Vary: Accept-Encoding to be cacheable, reason: %s", decision.Reason)
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
