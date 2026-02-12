package tests

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheExpiryAndRevalidation(t *testing.T) {
	env := SetupTestEnv(t)

	var requestCount int32
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		// Check for conditional headers
		if r.Header.Get("If-None-Match") == "\"expiry-etag\"" {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		// Use a slightly longer max-age to be safe, but short enough to test
		w.Header().Set("Cache-Control", "max-age=2")
		w.Header().Set("ETag", "\"expiry-etag\"")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("expiry test body"))
	})
	env.Start()

	targetURL := env.Upstream.URL + "/expiry-test"

	// 1. First request - Cache Miss

	resp1, err := env.Client.Get(targetURL)
	if err == nil {
		resp1.Body.Close()
	}

	if count := atomic.LoadInt32(&requestCount); count != 1 {
		t.Fatalf("Expected 1 upstream request, got %d", count)
	}

	// 2. Second request - Cache Hit
	resp2, err := env.Client.Get(targetURL)
	if err == nil {
		if !strings.Contains(resp2.Header.Get("Cache-Status"), "hit") {
			t.Errorf("Expected Cache-Status to contain hit, got %s", resp2.Header.Get("Cache-Status"))
		}
		resp2.Body.Close()
	}

	if count := atomic.LoadInt32(&requestCount); count != 1 {
		t.Errorf("Expected 1 upstream request (cache hit), got %d", count)
	}

	// 3. Wait for expiry (max-age is 2s, so wait 2.5s)
	time.Sleep(2500 * time.Millisecond)

	// 4. Third request - Cache Stale, Revalidation (304)
	resp3, err := env.Client.Get(targetURL)
	if err == nil {
		cacheStatus := resp3.Header.Get("Cache-Status")
		if !strings.Contains(cacheStatus, "revalidated") {
			t.Errorf("Expected Cache-Status to contain revalidated, got %s", cacheStatus)
		}
		resp3.Body.Close()
	}

	if count := atomic.LoadInt32(&requestCount); count != 2 {
		t.Errorf("Expected 2 upstream requests (revalidation), got %d", count)
	}
}
