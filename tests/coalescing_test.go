package tests

import (
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRequestCoalescing(t *testing.T) {
	env := SetupTestEnv(t)

	// 1. Setup Mock Upstream to track hits
	var requestCount int32
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		// Delay to ensure concurrent requests pile up
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("ETag", "\"test-etag\"")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})
	env.Start()

	// 2. Fire Concurrent Requests
	concurrentRequests := 20
	var wg sync.WaitGroup
	wg.Add(concurrentRequests)

	targetURL := env.Upstream.URL + "/coalesce-test"
	start := make(chan struct{})

	for range concurrentRequests {
		go func() {
			defer wg.Done()
			<-start // Wait for signal

			resp, err := env.Client.Get(targetURL)
			if err != nil {
				t.Errorf("Request failed: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
			}
		}()
	}

	// Release the hounds
	close(start)
	wg.Wait()

	// 3. Verify
	count := atomic.LoadInt32(&requestCount)
	if count != 1 {
		t.Errorf("Coalescing failed: Expected 1 upstream request, got %d", count)
	} else {
		t.Logf("Success: Coalescing worked. Upstream received %d request for %d client requests.", count, concurrentRequests)
	}
}

func BenchmarkRequestCoalescing(b *testing.B) {
	env := SetupTestEnv(b)

	// For benchmarking, we simulate some latency to allow coalescing to happen.
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("ETag", "\"bench-etag\"")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})
	env.Start()

	// Use a client with enough connections to support concurrency
	if transport, ok := env.Client.Transport.(*http.Transport); ok {
		transport.MaxIdleConns = 1000
		transport.MaxIdleConnsPerHost = 1000
	}

	targetURL := env.Upstream.URL + "/bench-test"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := env.Client.Get(targetURL)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if resp.StatusCode != 200 {
				b.Errorf("Expected 200 OK, got %d", resp.StatusCode)
			}
		}
	})
}
