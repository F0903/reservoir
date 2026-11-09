package tests

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"

	"reservoir/utils/bytesize"
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

func TestInvalidCachedRange(t *testing.T) {
	const content = "0123456789abcdefghijklmnopqrstuvwxyz"

	tests := []struct {
		name             string
		retry            bool
		wantStatus       int
		wantBody         string // Only checked when set.
		wantContentRange string
		wantClientErrors int64
	}{
		{
			name:             "returns 416 when retry disabled",
			wantStatus:       http.StatusRequestedRangeNotSatisfiable,
			wantContentRange: fmt.Sprintf("bytes */%d", len(content)),
			wantClientErrors: 1,
		},
		{
			name:       "retries as full response when enabled",
			retry:      true,
			wantStatus: http.StatusOK,
			wantBody:   content,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, hits, target := cachedRangeEnv(t, content)
			env.Cfg.Proxy.RetryOnInvalidRange.Overwrite(tt.retry)

			resp, body := getRange(t, env, target, "bytes=100-200")

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantBody != "" && body != tt.wantBody {
				t.Fatalf("expected body %q, got %q", tt.wantBody, body)
			}
			if got := resp.Header.Get("Content-Range"); got != tt.wantContentRange {
				t.Fatalf("expected Content-Range %q, got %q", tt.wantContentRange, got)
			}
			if got := hits.Load(); got != 1 {
				t.Fatalf("expected only the warmup to reach upstream, got %d upstream requests", got)
			}
			if got := env.Metrics.Requests.StatusClientErrorResponses.Get(); got != tt.wantClientErrors {
				t.Fatalf("expected %d client errors, got %d", tt.wantClientErrors, got)
			}
		})
	}
}

func TestUpstream416(t *testing.T) {
	const content = "0123456789abcdefghijklmnopqrstuvwxyz"

	tests := []struct {
		name         string
		retry        bool
		wantStatus   int
		wantBody     string
		wantRequests int64
	}{
		{name: "returned when retry disabled", wantStatus: http.StatusRequestedRangeNotSatisfiable, wantRequests: 1},
		{name: "retried without range when enabled", retry: true, wantStatus: http.StatusOK, wantBody: content, wantRequests: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := SetupTestEnv(t)
			env.Cfg.Proxy.RetryOnRange416.Overwrite(tt.retry)

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
				w.Write([]byte(content))
			})
			env.Start()

			req, err := http.NewRequest(http.MethodGet, env.Upstream.URL+"/upstream-416", nil)
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
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if string(body) != tt.wantBody {
				t.Fatalf("expected body %q, got %q", tt.wantBody, body)
			}
			if got := upstreamRequests.Load(); got != tt.wantRequests {
				t.Fatalf("expected %d upstream requests, got %d", tt.wantRequests, got)
			}
			if got := upstreamRangeRequests.Load(); got != 1 {
				t.Fatalf("expected only the original upstream request to carry Range, got %d", got)
			}
		})
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

func TestOversizedMemoryCacheResponseBypassesCache(t *testing.T) {
	env := SetupTestEnv(t)
	env.Cfg.Cache.MaxCacheSize.Overwrite(bytesize.ByteSize(4))

	assertRepeatedRequestsBypassCache(t, env, "/oversized-memory-cache", "oversized memory-cache response")
}

func TestZeroMemoryBudgetBypassesCache(t *testing.T) {
	env := SetupTestEnv(t)
	env.Cfg.Cache.Memory.MemoryBudgetPercent.Overwrite(0)

	assertRepeatedRequestsBypassCache(t, env, "/zero-memory-budget", "zero memory budget response")
}

func assertRepeatedRequestsBypassCache(t testing.TB, env *TestEnv, path string, reason string) {
	t.Helper()

	var upstreamRequests atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := upstreamRequests.Add(1)
		body := fmt.Sprintf("response %d", count)
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})
	env.Start()

	targetURL := env.Upstream.URL + path
	bodies := make([]string, 0, 2)
	for i := range 2 {
		resp, err := env.Client.Get(targetURL)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("failed to read response %d body: %v", i, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200 for request %d, got %d", i, resp.StatusCode)
		}
		bodies = append(bodies, string(body))
	}
	if bodies[0] == bodies[1] {
		t.Fatalf("expected %s to bypass cache, got repeated body %q", reason, bodies[0])
	}
	if got := upstreamRequests.Load(); got < 2 {
		t.Fatalf("expected %s to hit upstream more than once, got %d upstream requests", reason, got)
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
	for range 2 {
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
