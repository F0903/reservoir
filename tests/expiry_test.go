package tests

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheExpiryAndRevalidation(t *testing.T) {
	env := SetupTestEnv(t)

	var requestCount atomic.Int64
	var sawIfNoneMatch atomic.Bool
	var sawIfModifiedSince atomic.Bool
	lastModified := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)

		if count > 1 {
			if r.Header.Get("If-None-Match") == "\"expiry-etag\"" {
				sawIfNoneMatch.Store(true)
			}
			if r.Header.Get("If-Modified-Since") == lastModified {
				sawIfModifiedSince.Store(true)
			}
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Cache-Control", "max-age=1")
		w.Header().Set("ETag", "\"expiry-etag\"")
		w.Header().Set("Last-Modified", lastModified)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("expiry test body"))
	})
	env.Start()

	targetURL := env.Upstream.URL + "/expiry-test"

	resp1, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if body := readResponseBody(t, resp1); body != "expiry test body" {
		t.Fatalf("unexpected first response body: %q", body)
	}

	if count := requestCount.Load(); count != 1 {
		t.Fatalf("Expected 1 upstream request, got %d", count)
	}

	resp2, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	if !strings.Contains(resp2.Header.Get("Cache-Status"), "hit") {
		t.Errorf("Expected Cache-Status to contain hit, got %s", resp2.Header.Get("Cache-Status"))
	}
	readResponseBody(t, resp2)

	if count := requestCount.Load(); count != 1 {
		t.Errorf("Expected 1 upstream request (cache hit), got %d", count)
	}

	time.Sleep(1100 * time.Millisecond)

	resp3, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("third request failed: %v", err)
	}
	if body := readResponseBody(t, resp3); body != "expiry test body" {
		t.Fatalf("unexpected revalidated response body: %q", body)
	}
	cacheStatus := resp3.Header.Get("Cache-Status")
	if !strings.Contains(cacheStatus, "revalidated") {
		t.Errorf("Expected Cache-Status to contain revalidated, got %s", cacheStatus)
	}

	if count := requestCount.Load(); count != 2 {
		t.Errorf("Expected 2 upstream requests (revalidation), got %d", count)
	}
	if !sawIfNoneMatch.Load() {
		t.Fatal("expected stale revalidation to send If-None-Match")
	}
	if !sawIfModifiedSince.Load() {
		t.Fatal("expected stale revalidation to send If-Modified-Since")
	}

	resp4, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("fourth request failed: %v", err)
	}
	if body := readResponseBody(t, resp4); body != "expiry test body" {
		t.Fatalf("unexpected post-revalidation response body: %q", body)
	}
	if count := requestCount.Load(); count != 2 {
		t.Errorf("expected revalidated metadata to keep cache fresh, got %d upstream requests", count)
	}
}

func TestStaleCacheReplacedWhenUpstreamReturnsNewBody(t *testing.T) {
	env := SetupTestEnv(t)

	var requestCount atomic.Int64
	var sawIfNoneMatch atomic.Bool
	var sawIfModifiedSince atomic.Bool
	firstModified := time.Now().Add(-2 * time.Hour).UTC().Format(http.TimeFormat)
	secondModified := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count > 1 {
			if r.Header.Get("If-None-Match") == "\"body-v1\"" {
				sawIfNoneMatch.Store(true)
			}
			if r.Header.Get("If-Modified-Since") == firstModified {
				sawIfModifiedSince.Store(true)
			}

			w.Header().Set("Cache-Control", "max-age=60")
			w.Header().Set("ETag", "\"body-v2\"")
			w.Header().Set("Last-Modified", secondModified)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("body v2"))
			return
		}

		w.Header().Set("Cache-Control", "max-age=1")
		w.Header().Set("ETag", "\"body-v1\"")
		w.Header().Set("Last-Modified", firstModified)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("body v1"))
	})
	env.Start()

	targetURL := env.Upstream.URL + "/stale-replace"
	resp1, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if body := readResponseBody(t, resp1); body != "body v1" {
		t.Fatalf("unexpected first response body: %q", body)
	}

	time.Sleep(1100 * time.Millisecond)

	resp2, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	if body := readResponseBody(t, resp2); body != "body v2" {
		t.Fatalf("expected stale cache to be replaced with upstream body, got %q", body)
	}
	cacheStatus := resp2.Header.Get("Cache-Status")
	if !strings.Contains(cacheStatus, "revalidated") || !strings.Contains(cacheStatus, "fwd-status=200") {
		t.Fatalf("expected revalidated Cache-Status with upstream 200, got %q", cacheStatus)
	}
	if !sawIfNoneMatch.Load() {
		t.Fatal("expected stale replacement request to send If-None-Match")
	}
	if !sawIfModifiedSince.Load() {
		t.Fatal("expected stale replacement request to send If-Modified-Since")
	}

	resp3, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("third request failed: %v", err)
	}
	if body := readResponseBody(t, resp3); body != "body v2" {
		t.Fatalf("expected replacement body to be cached, got %q", body)
	}
	if count := requestCount.Load(); count != 2 {
		t.Fatalf("expected replacement body to be served from cache after revalidation, got %d upstream requests", count)
	}
}

func TestStaleCacheServedWhenUpstreamReturnsServerError(t *testing.T) {
	env := SetupTestEnv(t)

	var requestCount atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count > 1 {
			http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Cache-Control", "max-age=1")
		w.Header().Set("ETag", "\"stale-on-error\"")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("cached package body"))
	})
	env.Start()

	targetURL := env.Upstream.URL + "/stale-on-error"
	resp1, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if body := readResponseBody(t, resp1); body != "cached package body" {
		t.Fatalf("unexpected first response body: %q", body)
	}

	time.Sleep(1100 * time.Millisecond)

	resp2, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected stale cache to be served with 200 OK, got %d", resp2.StatusCode)
	}
	if body := readResponseBody(t, resp2); body != "cached package body" {
		t.Fatalf("expected stale cached body, got %q", body)
	}
	cacheStatus := resp2.Header.Get("Cache-Status")
	if !strings.Contains(cacheStatus, "detail=\"stale\"") || !strings.Contains(cacheStatus, "fwd-status=503") {
		t.Fatalf("expected stale Cache-Status with upstream 503, got %q", cacheStatus)
	}
	if got := resp2.Header.Get("X-Cache"); got != "STALE" {
		t.Fatalf("expected X-Cache STALE, got %q", got)
	}
	if count := requestCount.Load(); count != 2 {
		t.Fatalf("expected one failed revalidation attempt, got %d upstream requests", count)
	}
}

func TestStaleCacheServedWhenUpstreamIsUnavailable(t *testing.T) {
	env := SetupTestEnv(t)

	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=1")
		w.Header().Set("ETag", "\"stale-when-offline\"")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("offline cached package"))
	})
	env.Start()

	targetURL := env.Upstream.URL + "/stale-when-offline"
	resp1, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if body := readResponseBody(t, resp1); body != "offline cached package" {
		t.Fatalf("unexpected first response body: %q", body)
	}

	time.Sleep(1100 * time.Millisecond)
	env.Upstream.Close()

	resp2, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected stale cache to be served with 200 OK, got %d", resp2.StatusCode)
	}
	if body := readResponseBody(t, resp2); body != "offline cached package" {
		t.Fatalf("expected stale cached body, got %q", body)
	}
	cacheStatus := resp2.Header.Get("Cache-Status")
	if !strings.Contains(cacheStatus, "detail=\"stale\"") {
		t.Fatalf("expected stale Cache-Status for unavailable upstream, got %q", cacheStatus)
	}
	if strings.Contains(cacheStatus, "fwd-status=") {
		t.Fatalf("did not expect fwd-status when upstream was unreachable, got %q", cacheStatus)
	}
	if got := resp2.Header.Get("X-Cache"); got != "STALE" {
		t.Fatalf("expected X-Cache STALE, got %q", got)
	}
}

func readResponseBody(t *testing.T, resp *http.Response) string {
	t.Helper()

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	return string(body)
}
