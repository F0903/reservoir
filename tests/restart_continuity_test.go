package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reservoir/config"
	"reservoir/logging"
	"reservoir/proxy"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFileCacheSidecarsRestoreProxyResponseAfterRestart(t *testing.T) {
	var upstreamRequests atomic.Int64
	body := "cached package payload"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRequests.Add(1)
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("ETag", `"restart-etag"`)
		w.Header().Set("X-Package-Revision", "42")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer upstream.Close()

	cfg := newFileCacheProxyConfig(t)
	firstProxy := startRestartableProxy(t, cfg)

	targetURL := upstream.URL + "/pool/main/package.deb"
	resp, err := firstProxy.client.Get(targetURL)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if got := readRestartTestBody(t, resp); got != body {
		t.Fatalf("unexpected first response body: %q", got)
	}
	if got := upstreamRequests.Load(); got != 1 {
		t.Fatalf("expected first request to hit upstream once, got %d", got)
	}

	firstProxy.close()

	secondProxy := startRestartableProxy(t, cfg)
	defer secondProxy.close()

	resp, err = secondProxy.client.Get(targetURL)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	if got := readRestartTestBody(t, resp); got != body {
		t.Fatalf("unexpected restored response body: %q", got)
	}
	if got := upstreamRequests.Load(); got != 1 {
		t.Fatalf("expected restored file-cache hit without upstream request, got %d upstream requests", got)
	}
	if got := resp.Header.Get("ETag"); got != `"restart-etag"` {
		t.Fatalf("expected restored ETag header, got %q", got)
	}
	if got := resp.Header.Get("X-Package-Revision"); got != "42" {
		t.Fatalf("expected restored custom header, got %q", got)
	}
	if cacheStatus := resp.Header.Get("Cache-Status"); !strings.Contains(cacheStatus, "hit") {
		t.Fatalf("expected restored response to be a cache hit, got Cache-Status %q", cacheStatus)
	}
}

func TestFileCacheSidecarsDoNotRestoreExpiredProxyResponse(t *testing.T) {
	var upstreamRequests atomic.Int64

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := upstreamRequests.Add(1)
		if count == 1 {
			w.Header().Set("Cache-Control", "max-age=1")
			w.Header().Set("ETag", `"expired-restart-etag"`)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("expired package payload"))
			return
		}

		w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("ETag", `"fresh-restart-etag"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fresh package payload"))
	}))
	defer upstream.Close()

	cfg := newFileCacheProxyConfig(t)
	firstProxy := startRestartableProxy(t, cfg)

	targetURL := upstream.URL + "/pool/main/expired-package.deb"
	resp, err := firstProxy.client.Get(targetURL)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if got := readRestartTestBody(t, resp); got != "expired package payload" {
		t.Fatalf("unexpected first response body: %q", got)
	}
	firstProxy.close()

	time.Sleep(1100 * time.Millisecond)

	secondProxy := startRestartableProxy(t, cfg)
	defer secondProxy.close()

	resp, err = secondProxy.client.Get(targetURL)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	if got := readRestartTestBody(t, resp); got != "fresh package payload" {
		t.Fatalf("expected expired sidecar to be ignored, got body %q", got)
	}
	if got := upstreamRequests.Load(); got != 2 {
		t.Fatalf("expected restart request to refetch expired entry, got %d upstream requests", got)
	}
	if got := resp.Header.Get("ETag"); got != `"fresh-restart-etag"` {
		t.Fatalf("expected fresh upstream ETag, got %q", got)
	}
}

type restartableProxy struct {
	proxy     *proxy.Proxy
	server    *httptest.Server
	client    *http.Client
	closeOnce sync.Once
}

func newFileCacheProxyConfig(t *testing.T) *config.Config {
	t.Helper()

	cfg := config.NewDefault()
	cfg.Proxy.UpstreamDefaultHttps.Overwrite(false)
	cfg.Proxy.RetryOnRange416.Overwrite(false)
	cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(false)
	cfg.Proxy.CachePolicy.ForceDefaultMaxAge.Overwrite(false)
	cfg.Cache.Type.Overwrite(config.CacheTypeFile)
	cfg.Cache.File.Dir.Overwrite(t.TempDir())
	cfg.Cache.LockShards.Overwrite(32)
	cfg.Logging.ToStdout.Overwrite(false)

	logging.Init(cfg)
	return cfg
}

func startRestartableProxy(t *testing.T, cfg *config.Config) *restartableProxy {
	t.Helper()

	p, err := proxy.NewProxy(cfg, &FakeCA{}, t.Context())
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	server := httptest.NewServer(p)
	proxyURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse proxy URL: %v", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	rp := &restartableProxy{
		proxy:  p,
		server: server,
		client: client,
	}
	t.Cleanup(rp.close)
	return rp
}

func (p *restartableProxy) close() {
	p.closeOnce.Do(func() {
		p.server.Close()
		p.proxy.Destroy()
		if transport, ok := p.client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	})
}

func readRestartTestBody(t *testing.T, resp *http.Response) string {
	t.Helper()

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	return string(body)
}
