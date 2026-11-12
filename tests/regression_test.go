package tests

import (
	"context"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reservoir/application"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/proxy"
	"reservoir/proxy/certs"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFollowerSurvivesLeaderDisconnect(t *testing.T) {
	env := SetupTestEnv(t)

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	var hits atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Length", "10")
		w.Header().Set("Cache-Control", "max-age=60")
		w.Write([]byte("hello"))
		w.(http.Flusher).Flush()
		once.Do(func() { close(started) })
		<-release
		w.Write([]byte("world"))
	})
	env.Start()
	target := env.Upstream.URL + "/shared.deb"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	leaderDone := make(chan struct{})
	go func() {
		defer close(leaderDone)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if resp, err := env.Client.Do(req); err == nil {
			resp.Body.Close()
		}
	}()
	<-started

	type result struct {
		status int
		body   string
		err    error
	}
	follower := make(chan result, 1)
	go func() {
		resp, err := env.Client.Get(target)
		if err != nil {
			follower <- result{err: err}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		follower <- result{status: resp.StatusCode, body: string(body), err: err}
	}()
	time.Sleep(200 * time.Millisecond)

	cancel()
	<-leaderDone
	time.Sleep(100 * time.Millisecond)
	close(release)

	select {
	case got := <-follower:
		if got.err != nil || got.status != http.StatusOK || got.body != "helloworld" {
			t.Fatalf("follower got status=%d body=%q err=%v", got.status, got.body, got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("follower never received a response")
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("expected the shared download to survive, got %d upstream requests", got)
	}
}

func TestHTTPSTunnelUsesTLSUpstreamWithUpgradeOff(t *testing.T) {
	env := SetupHttpsTestEnv(t)
	env.Cfg.Proxy.UpgradeHttpToHttps.Overwrite(false)

	var sawTLS atomic.Bool
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawTLS.Store(r.TLS != nil)
		w.Write([]byte("secure"))
	})
	env.Start()

	resp, err := env.Client.Get(env.Upstream.URL + "/secret")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK || !sawTLS.Load() {
		t.Fatalf("status=%d, upstream saw TLS=%v; HTTPS must stay encrypted", resp.StatusCode, sawTLS.Load())
	}
}

func TestUpgradeSettingSendsPlainHTTPOverTLS(t *testing.T) {
	env := SetupHttpsTestEnv(t)
	env.Cfg.Proxy.UpgradeHttpToHttps.Overwrite(true)

	var sawTLS atomic.Bool
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawTLS.Store(r.TLS != nil)
		w.Write([]byte("upgraded"))
	})
	env.Start()

	plainURL := strings.Replace(env.Upstream.URL, "https://", "http://", 1)
	resp, err := env.Client.Get(plainURL + "/upgrade-me")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK || !sawTLS.Load() {
		t.Fatalf("status=%d, upstream saw TLS=%v; expected the request to be upgraded", resp.StatusCode, sawTLS.Load())
	}
}

func newBareProxy(t *testing.T, cfg *config.Config, upstreamClient *http.Client) *http.Client {
	t.Helper()

	client, _ := newMeteredProxy(t, cfg, upstreamClient)
	return client
}

func newMeteredProxy(t *testing.T, cfg *config.Config, upstreamClient *http.Client) (*http.Client, *metrics.Metrics) {
	t.Helper()

	m := metrics.NewMetrics()
	store, err := application.NewProxyCache(cfg, m, t.Context())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	p, err := proxy.NewProxy(cfg, &FakeCA{}, proxy.Dependencies{Cache: store, Metrics: m, UpstreamClient: upstreamClient})
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	t.Cleanup(p.Destroy)

	server := httptest.NewServer(p)
	t.Cleanup(server.Close)

	proxyURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{Transport: transport}, m
}

func TestDefaultUpstreamClientRefusesLoopback(t *testing.T) {
	var hits atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		hits.Add(1)
	}))
	t.Cleanup(upstream.Close)

	cfg := config.NewDefault()
	cfg.Proxy.UpgradeHttpToHttps.Overwrite(false)
	cfg.Cache.Type.Overwrite(config.CacheTypeMemory)
	cfg.Logging.File.Overwrite("")

	resp, err := newBareProxy(t, cfg, nil).Get(upstream.URL + "/internal")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway || hits.Load() != 0 {
		t.Fatalf("status=%d upstream hits=%d; loopback must be unreachable", resp.StatusCode, hits.Load())
	}
}

func TestRequestSentToProxyItselfIsRefused(t *testing.T) {
	env := SetupTestEnv(t)

	var hops atomic.Int64
	env.ProxyServer.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cap the recursion so a regression fails the test instead of hanging it.
		if hops.Add(1) > 20 {
			w.WriteHeader(http.StatusLoopDetected)
			return
		}
		env.Proxy.ServeHTTP(w, r)
	})
	env.Start()

	resp, err := http.Head(env.ProxyServer.URL + "/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest || hops.Load() != 1 {
		t.Fatalf("status=%d proxy hops=%d; a request sent to the proxy itself must be refused at once", resp.StatusCode, hops.Load())
	}
}

func TestEncodedPathReachesUpstreamIntact(t *testing.T) {
	env := SetupTestEnv(t)

	seen := make(chan string, 1)
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.RequestURI
		w.Write([]byte("ok"))
	})
	env.Start()

	resp, err := env.Client.Get(env.Upstream.URL + "/api/group%2Fproject/x")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if got := <-seen; got != "/api/group%2Fproject/x" {
		t.Fatalf("upstream saw %q", got)
	}
}

func TestTrailingSlashHasItsOwnCacheEntry(t *testing.T) {
	env := SetupTestEnv(t)
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		w.Write([]byte("body for " + r.RequestURI))
	})
	env.Start()

	for _, path := range []string{"/dir/", "/dir"} {
		resp, err := env.Client.Get(env.Upstream.URL + path)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if got := readResponseBody(t, resp); got != "body for "+path {
			t.Fatalf("asked for %s, got %q", path, got)
		}
	}
}

func TestEmptyBodyWithFileCacheIsServedDirectly(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	cfg := config.NewDefault()
	cfg.Proxy.UpgradeHttpToHttps.Overwrite(false)
	cfg.Cache.Type.Overwrite(config.CacheTypeFile)
	cfg.Cache.File.Dir.Overwrite(t.TempDir())
	cfg.Logging.File.Overwrite("")

	resp, err := newBareProxy(t, cfg, newTestUpstreamClient()).Get(upstream.URL + "/empty-Packages")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected the empty response to be served, got status %d", resp.StatusCode)
	}
}

func TestTunnelClosesAfterFailedResponse(t *testing.T) {
	env := SetupHttpsTestEnv(t)
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.Write([]byte("0123456789"))
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	})
	env.Start()

	req, _ := http.NewRequest(http.MethodGet, env.Upstream.URL+"/truncated", nil)
	req.Header.Set("Authorization", "x") // Skips the cache, so the failure happens mid-response.

	start := time.Now()
	resp, err := env.Client.Do(req)
	if err == nil {
		_, err = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	if err == nil {
		t.Fatal("expected the truncated response to fail")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("client hung for %v after the upstream failed", elapsed)
	}
}

func TestTunnelStaysOpenAfterHeadResponse(t *testing.T) {
	env := SetupHttpsTestEnv(t)

	var connections atomic.Int64
	env.ProxyServer.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "2")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodHead {
			w.Write([]byte("ok"))
		}
	})
	env.Start()

	head, err := env.Client.Head(env.Upstream.URL + "/file")
	if err != nil {
		t.Fatalf("HEAD failed: %v", err)
	}
	head.Body.Close()
	if head.ContentLength != 2 {
		t.Fatalf("HEAD reported Content-Length %d", head.ContentLength)
	}

	get, err := env.Client.Get(env.Upstream.URL + "/file")
	if err != nil {
		t.Fatalf("GET after HEAD failed: %v", err)
	}
	if got := readResponseBody(t, get); got != "ok" {
		t.Fatalf("GET body %q", got)
	}

	if got := connections.Load(); got != 1 {
		t.Fatalf("expected one tunnel for both requests, got %d", got)
	}
}

func TestTunnelReusesUpstreamConnections(t *testing.T) {
	env := SetupHttpsTestEnv(t)

	var mu sync.Mutex
	addrs := map[string]bool{}
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		addrs[r.RemoteAddr] = true
		mu.Unlock()
		w.Header().Set("Cache-Control", "no-store")
		w.Write([]byte("ok"))
	})
	env.Start()

	for _, path := range []string{"/a", "/b", "/c", "/d"} {
		req, _ := http.NewRequest(http.MethodGet, env.Upstream.URL+path, nil)
		req.Header.Set("Authorization", "x")
		resp, err := env.Client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	mu.Lock()
	defer mu.Unlock()
	if len(addrs) != 1 {
		t.Fatalf("expected one upstream connection, got %d", len(addrs))
	}
}

func cachedRangeEnv(t *testing.T, content string) (*TestEnv, *atomic.Int64, string) {
	t.Helper()

	env := SetupTestEnv(t)
	hits := &atomic.Int64{}
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("ETag", "\"range-etag\"")
		w.Write([]byte(content))
	})
	env.Start()

	target := env.Upstream.URL + "/range"
	resp, err := env.Client.Get(target)
	if err != nil {
		t.Fatalf("warmup failed: %v", err)
	}
	resp.Body.Close()
	return env, hits, target
}

func getRange(t *testing.T, env *TestEnv, target, rangeHeader string) (*http.Response, string) {
	t.Helper()

	req, _ := http.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("Range", rangeHeader)
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("range request failed: %v", err)
	}
	return resp, readResponseBody(t, resp)
}

func TestCachedRangeDoesNotGoUpstream(t *testing.T) {
	env, hits, target := cachedRangeEnv(t, "0123456789abcdefghijklmnopqrstuvwxyz")

	resp, body := getRange(t, env, target, "bytes=2-5")

	if resp.StatusCode != http.StatusPartialContent || body != "2345" {
		t.Fatalf("status=%d body=%q", resp.StatusCode, body)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("expected only the warmup upstream, got %d requests", got)
	}
}

func TestRangePastEndOfFileIsTrimmed(t *testing.T) {
	env, _, target := cachedRangeEnv(t, "0123456789abcdefghijklmnopqrstuvwxyz")

	resp, body := getRange(t, env, target, "bytes=30-100")

	if resp.StatusCode != http.StatusPartialContent || body != "uvwxyz" {
		t.Fatalf("status=%d body=%q", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Range"); got != "bytes 30-35/36" {
		t.Fatalf("Content-Range = %q", got)
	}
}

func TestCachedResponseWithoutETagOmitsHeader(t *testing.T) {
	env := SetupTestEnv(t)
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		w.Write([]byte("no etag"))
	})
	env.Start()

	for range 2 {
		resp, err := env.Client.Get(env.Upstream.URL + "/no-etag")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()
		if values, present := resp.Header["Etag"]; present {
			t.Fatalf("expected no ETag header, got %q", values)
		}
	}
}

func TestTokenHeadersBypassSharedCache(t *testing.T) {
	for _, header := range []string{"Authorization", "Private-Token", "X-Api-Key"} {
		t.Run(header, func(t *testing.T) {
			env := SetupTestEnv(t)
			var hits atomic.Int64
			env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				w.Header().Set("Cache-Control", "max-age=60")
				w.Write([]byte("secret for " + r.Header.Get(header)))
			})
			env.Start()

			bytesFetchedBefore := env.Metrics.Requests.BytesFetched.Get()
			for _, token := range []string{"alice", "bob"} {
				req, _ := http.NewRequest(http.MethodGet, env.Upstream.URL+"/repo.tar.gz", nil)
				req.Header.Set(header, token)
				resp, err := env.Client.Do(req)
				if err != nil {
					t.Fatalf("request failed: %v", err)
				}
				if got := readResponseBody(t, resp); got != "secret for "+token {
					t.Fatalf("%s received %q", token, got)
				}
			}
			if got := hits.Load(); got != 2 {
				t.Fatalf("expected both requests upstream, got %d", got)
			}
			expectedFetched := int64(len("secret for alice") + len("secret for bob"))
			if got := env.Metrics.Requests.BytesFetched.Get() - bytesFetchedBefore; got < expectedFetched {
				t.Fatalf("expected at least %d direct upstream bytes fetched, got %d", expectedFetched, got)
			}
		})
	}
}

func TestPrivateResponseIsNeverSharedEvenWhenIgnoringCacheControl(t *testing.T) {
	env := SetupTestEnv(t)
	env.Cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)

	var hits atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Cache-Control", "private, max-age=60")
		w.Write([]byte("mine"))
	})
	env.Start()

	for range 2 {
		resp, err := env.Client.Get(env.Upstream.URL + "/profile")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("expected a private response to stay uncached, got %d upstream requests", got)
	}
}

func TestBadGatewayIsCounted(t *testing.T) {
	env := SetupTestEnv(t)
	env.Upstream.Config.Handler = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	})
	env.Start()

	resp, err := env.Client.Get(env.Upstream.URL + "/fails")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	requests := &env.Metrics.Requests
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}
	if got := requests.StatusServerErrorResponses.Get(); got != 1 {
		t.Fatalf("expected one server error, got %d", got)
	}
	if got := requests.ClientResponses.Get(); got != 1 {
		t.Fatalf("expected one client response, got %d", got)
	}
	if got := requests.NonCoalescedRequests.Get(); got != 1 {
		t.Fatalf("expected the failed request to be counted, got %d", got)
	}
}

func TestBytesServedCountsBrokenDownloads(t *testing.T) {
	env := SetupTestEnv(t)
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.Write(make([]byte, 100))
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	})
	env.Start()

	req, _ := http.NewRequest(http.MethodGet, env.Upstream.URL+"/broken", nil)
	req.Header.Set("Authorization", "x") // Skips the cache, so the body is streamed as it arrives.
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if got := env.Metrics.Requests.BytesServed.Get(); got != 100 {
		t.Fatalf("expected 100 bytes served, got %d", got)
	}
}

func TestBytesFetchedCountsFailedCacheWrites(t *testing.T) {
	env := SetupTestEnv(t)
	var hits atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Length", "1000")
		w.Header().Set("Cache-Control", "max-age=60")
		w.Write(make([]byte, 100))
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	})
	env.Start()

	resp, err := env.Client.Get(env.Upstream.URL + "/truncated")
	if err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// The client reads along with the cache attempt, so the fallback is a request for the
	// rest. This upstream answers it with a file that cannot be verified, which is not read.
	if hits.Load() != 2 {
		t.Fatalf("expected the cache attempt and a request for the rest upstream, got %d requests", hits.Load())
	}
	if got := env.Metrics.Requests.BytesFetched.Get(); got != 100 {
		t.Fatalf("expected the 100 bytes of the failed cache attempt to be counted, got %d", got)
	}
}

func TestFailedCacheReadCountsOneError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("body"))
	}))
	t.Cleanup(upstream.Close)

	dir := t.TempDir()
	cfg := config.NewDefault()
	cfg.Cache.Type.Overwrite(config.CacheTypeFile)
	cfg.Cache.File.Dir.Overwrite(dir)
	cfg.Logging.File.Overwrite("")
	client, m := newMeteredProxy(t, cfg, newTestUpstreamClient())

	resp, err := client.Get(upstream.URL + "/file")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Ext(file.Name()) == "" { // The body; its metadata stays loaded.
			if err := os.Remove(filepath.Join(dir, file.Name())); err != nil {
				t.Fatal(err)
			}
		}
	}

	resp, err = client.Get(upstream.URL + "/file")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected the direct fallback to succeed, got %d", resp.StatusCode)
	}
	if got := m.Cache.CacheErrors.Get(); got != 1 {
		t.Fatalf("expected one cache error, got %d", got)
	}
}

func TestCoalescedStaleResponsesAreCounted(t *testing.T) {
	env := SetupTestEnv(t)

	var hits atomic.Int64
	release := make(chan struct{})
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.Header().Set("Cache-Control", "max-age=1")
			w.Write([]byte("cached"))
			return
		}
		<-release
		w.WriteHeader(http.StatusInternalServerError)
	})
	env.Start()
	target := env.Upstream.URL + "/stale"

	resp, err := env.Client.Get(target)
	if err != nil {
		t.Fatalf("warmup failed: %v", err)
	}
	resp.Body.Close()
	time.Sleep(1100 * time.Millisecond)

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			if resp, err := env.Client.Get(target); err == nil {
				resp.Body.Close()
			}
		})
	}
	time.Sleep(200 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := env.Metrics.Requests.CoalescedCacheStales.Get(); got != 2 {
		t.Fatalf("expected both joined requests to be counted as stale, got %d", got)
	}
}

func TestGeneratedCertificatesTolerateSlowClientClocks(t *testing.T) {
	certFile, keyFile := GenerateTestCA(t)
	ca, err := certs.NewPrivateCA(certFile, keyFile)
	if err != nil {
		t.Fatalf("failed to create CA: %v", err)
	}

	tlsCert, err := ca.GetCertForHost("example.test:443")
	if err != nil {
		t.Fatalf("failed to issue certificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	if earliest := time.Now().Add(-30 * time.Minute); leaf.NotBefore.After(earliest) {
		t.Fatalf("NotBefore %v is too close to now", leaf.NotBefore)
	}
	if !strings.Contains(leaf.Issuer.String(), "reservoir-test") {
		t.Fatalf("unexpected issuer %s", leaf.Issuer)
	}
}
