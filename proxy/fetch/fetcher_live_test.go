package fetch

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"reservoir/cache"
	"reservoir/cache/memory"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/proxy/headers"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// A request that reads far slower than the download falls out of the window of
// recent bytes. It must get the rest of the body from the finished cache entry.
func TestRequestLeftBehindByDownloadContinuesFromCache(t *testing.T) {
	body := make([]byte, 1<<20)
	for i := range body {
		body[i] = byte(i ^ i>>8)
	}
	const firstPart = 8 << 10

	gate := make(chan struct{})
	release := sync.OnceFunc(func() { close(gate) })
	t.Cleanup(release)
	var hits atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("Cache-Control", "max-age=60")
		w.Write(body[:firstPart])
		w.(http.Flusher).Flush()
		<-gate
		w.Write(body[firstPart:])
	}))
	t.Cleanup(upstream.Close)

	cfg := config.NewDefault()
	cfg.Proxy.UpgradeHttpToHttps.Overwrite(false)
	m := metrics.NewMetrics()
	store := memory.New[CacheMetadata](cfg, m, 100, 1<<30, time.Hour, 16, t.Context())
	t.Cleanup(store.Destroy)

	f := New(store, cfg, &http.Client{}, m)
	// Holds the first part, so the request can join before the pause, but far less
	// than the body, so the download runs ahead of the request once it continues.
	f.liveWindow = 2 * firstPart

	req := httptest.NewRequest(http.MethodGet, upstream.URL+"/slow-reader.deb", nil)
	baseKey := cache.MakeFromRequest(req)
	fetched, err := f.Fetch(req, baseKey, headers.ParseHeaderDirective(req.Header))
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	data := fetched.Cached.Entry.Data
	defer data.Close()

	start := make([]byte, 512)
	if _, err := io.ReadFull(data, start); err != nil {
		t.Fatalf("failed to read the start of the body while it was downloading: %v", err)
	}

	// Let the download finish without reading along.
	release()
	deadline := time.Now().Add(5 * time.Second)
	for {
		entry, err := store.Get(f.lookupCacheKey(req, baseKey))
		if err == nil {
			entry.Data.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the download never reached the cache: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	rest, err := io.ReadAll(data)
	if err != nil || !bytes.Equal(append(start, rest...), body) {
		t.Fatalf("got %d bytes (err=%v) that differ from the %d byte body", len(start)+len(rest), err, len(body))
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("expected the rest to come from the cache, got %d upstream requests", got)
	}
}
