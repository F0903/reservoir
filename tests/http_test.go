package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
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
