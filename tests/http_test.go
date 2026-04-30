package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestBasicHTTPRequest(t *testing.T) {
	env := SetupTestEnv(t)
	env.Start()

	targetURL := env.Upstream.URL + "/basic-test"
	resp, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if string(body) != "response body" {
		t.Errorf("Expected response body 'response body', got '%s'", string(body))
	}
}

func TestRangeRequests(t *testing.T) {
	env := SetupTestEnv(t)

	// Setup upstream to serve a larger body
	content := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("ETag", "\"range-etag\"")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})
	env.Start()

	targetURL := env.Upstream.URL + "/range-test"

	// 1. Warm up the cache
	resp, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("Warmup failed: %v", err)
	}
	resp.Body.Close()

	// 2. Make range request
	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header.Set("Range", "bytes=0-9")
	resp, err = env.Client.Do(req)
	if err != nil {
		t.Fatalf("Range request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		t.Errorf("Expected 206 Partial Content, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(body, content[0:10]) {
		t.Errorf("Expected body %q, got %q", content[0:10], body)
	}

	// 3. Make another range request
	req, _ = http.NewRequest("GET", targetURL, nil)
	req.Header.Set("Range", "bytes=10-19")
	resp, err = env.Client.Do(req)
	if err != nil {
		t.Fatalf("Second range request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	if !bytes.Equal(body, content[10:20]) {
		t.Errorf("Expected body %q, got %q", content[10:20], body)
	}
}

func TestInvalidCachedRangeReturns416WhenRetryDisabled(t *testing.T) {
	env := SetupTestEnv(t)

	content := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	var upstreamRequests atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRequests.Add(1)
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("ETag", "\"range-etag\"")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})
	env.Start()

	targetURL := env.Upstream.URL + "/invalid-range-no-retry"
	resp, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("warmup failed: %v", err)
	}
	resp.Body.Close()

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Range", "bytes=100-200")
	resp, err = env.Client.Do(req)
	if err != nil {
		t.Fatalf("range request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("expected 416 Range Not Satisfiable, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Range"); got != fmt.Sprintf("bytes */%d", len(content)) {
		t.Fatalf("expected Content-Range bytes */%d, got %q", len(content), got)
	}
	if got := upstreamRequests.Load(); got != 2 {
		t.Fatalf("expected warmup plus original range request, got %d upstream requests", got)
	}
}

func TestInvalidCachedRangeRetriesAsFullResponseWhenEnabled(t *testing.T) {
	env := SetupTestEnv(t)
	env.Cfg.Proxy.RetryOnInvalidRange.Overwrite(true)

	content := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	var upstreamRequests atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRequests.Add(1)
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("ETag", "\"range-etag\"")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})
	env.Start()

	targetURL := env.Upstream.URL + "/invalid-range-retry"
	resp, err := env.Client.Get(targetURL)
	if err != nil {
		t.Fatalf("warmup failed: %v", err)
	}
	resp.Body.Close()

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Range", "bytes=100-200")
	resp, err = env.Client.Do(req)
	if err != nil {
		t.Fatalf("range request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected retry to return 200 OK, got %d", resp.StatusCode)
	}
	if !bytes.Equal(body, content) {
		t.Fatalf("expected full cached body %q, got %q", content, body)
	}
	if got := upstreamRequests.Load(); got != 2 {
		t.Fatalf("expected warmup plus original range request, got %d upstream requests", got)
	}
}

func TestUpstream416IsReturnedWhenRetryDisabled(t *testing.T) {
	env := SetupTestEnv(t)
	env.Cfg.Proxy.RetryOnRange416.Overwrite(false)

	content := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	var upstreamRequests atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRequests.Add(1)
		if r.Header.Get("Range") != "" {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", len(content)))
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set("Cache-Control", "max-age=60")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})
	env.Start()

	req, err := http.NewRequest(http.MethodGet, env.Upstream.URL+"/upstream-416-no-retry", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Range", "bytes=0-9")
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("range request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("expected upstream 416 to pass through, got %d", resp.StatusCode)
	}
	if got := upstreamRequests.Load(); got != 1 {
		t.Fatalf("expected one upstream request with retry disabled, got %d", got)
	}
}

func TestUpstream416RetriesWithoutRangeWhenEnabled(t *testing.T) {
	env := SetupTestEnv(t)
	env.Cfg.Proxy.RetryOnRange416.Overwrite(true)

	content := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	var upstreamRequests atomic.Int64
	var upstreamRangeRequests atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRequests.Add(1)
		if r.Header.Get("Range") != "" {
			upstreamRangeRequests.Add(1)
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", len(content)))
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set("Cache-Control", "max-age=60")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})
	env.Start()

	req, err := http.NewRequest(http.MethodGet, env.Upstream.URL+"/upstream-416-retry", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Range", "bytes=0-9")
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("range request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected retry to return 200 OK, got %d", resp.StatusCode)
	}
	if !bytes.Equal(body, content) {
		t.Fatalf("expected full retried body %q, got %q", content, body)
	}
	if got := upstreamRequests.Load(); got != 2 {
		t.Fatalf("expected original range request plus retry, got %d upstream requests", got)
	}
	if got := upstreamRangeRequests.Load(); got != 1 {
		t.Fatalf("expected only original upstream request to carry Range, got %d", got)
	}
}

func TestVaryAcceptEncodingUsesSeparateCacheVariants(t *testing.T) {
	env := SetupTestEnv(t)

	var upstreamRequests atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRequests.Add(1)
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("Vary", "Accept-Encoding")
		w.WriteHeader(http.StatusOK)

		if r.Header.Get("Accept-Encoding") == "gzip" {
			w.Write([]byte("gzip variant"))
			return
		}
		w.Write([]byte("identity variant"))
	})
	env.Start()

	targetURL := env.Upstream.URL + "/vary-accept-encoding"
	doRequest := func(acceptEncoding string) string {
		req, err := http.NewRequest(http.MethodGet, targetURL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Accept-Encoding", acceptEncoding)

		resp, err := env.Client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}
		return string(body)
	}

	if got := doRequest("gzip"); got != "gzip variant" {
		t.Fatalf("unexpected gzip variant body: %q", got)
	}
	if got := doRequest("identity"); got != "identity variant" {
		t.Fatalf("unexpected identity variant body: %q", got)
	}
	if got := doRequest("gzip"); got != "gzip variant" {
		t.Fatalf("unexpected cached gzip variant body: %q", got)
	}
	if got := upstreamRequests.Load(); got != 2 {
		t.Fatalf("expected 2 upstream requests for 2 variants, got %d", got)
	}
}

func TestCredentialedRequestsBypassCache(t *testing.T) {
	env := SetupTestEnv(t)

	var upstreamRequests atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := upstreamRequests.Add(1)
		w.Header().Set("Cache-Control", "max-age=60")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "response %d", count)
	})
	env.Start()

	targetURL := env.Upstream.URL + "/private"
	doRequest := func() string {
		req, err := http.NewRequest(http.MethodGet, targetURL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := env.Client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}
		return string(body)
	}

	if got := doRequest(); got != "response 1" {
		t.Fatalf("unexpected first response body: %q", got)
	}
	if got := doRequest(); got != "response 2" {
		t.Fatalf("credentialed request was served from cache: %q", got)
	}
	if got := upstreamRequests.Load(); got != 2 {
		t.Fatalf("expected 2 upstream requests, got %d", got)
	}
}

func TestNoStoreCachesWhenAggressivePackageCacheEnabled(t *testing.T) {
	env := SetupTestEnv(t)
	env.Cfg.Proxy.CachePolicy.IgnoreCacheControl.Overwrite(true)
	env.Cfg.Proxy.CachePolicy.ForceDefaultMaxAge.Overwrite(true)

	var upstreamRequests atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := upstreamRequests.Add(1)
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "package %d", count)
	})
	env.Start()

	targetURL := env.Upstream.URL + "/package.deb"
	for i := 0; i < 2; i++ {
		resp, err := env.Client.Get(targetURL)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}
		if string(body) != "package 1" {
			t.Fatalf("unexpected response body: %q", body)
		}
	}
	if got := upstreamRequests.Load(); got != 1 {
		t.Fatalf("expected aggressive package-cache mode to cache no-store response, got %d upstream requests", got)
	}
}

func BenchmarkProxyLatencyCold(b *testing.B) {
	env := SetupTestEnv(b)
	env.Start()

	i := 0
	b.ResetTimer()
	for b.Loop() {
		targetURL := fmt.Sprintf("%s/uncached-%d", env.Upstream.URL, i)
		resp, err := env.Client.Get(targetURL)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		i++
	}
}

func BenchmarkProxyLatencyHot(b *testing.B) {
	env := SetupTestEnv(b)
	env.Start()
	targetURL := env.Upstream.URL + "/cached-resource"

	resp, _ := env.Client.Get(targetURL)
	resp.Body.Close()

	b.ResetTimer()
	for b.Loop() {
		resp, err := env.Client.Get(targetURL)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkThroughput(b *testing.B) {
	env := SetupTestEnv(b)

	// 10MB payload
	payloadSize := 10 * 1024 * 1024
	payload := make([]byte, payloadSize)

	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
	})
	env.Start()

	targetURL := env.Upstream.URL + "/large-file"

	b.SetBytes(int64(payloadSize))
	b.ResetTimer()
	for b.Loop() {
		resp, err := env.Client.Get(targetURL)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
