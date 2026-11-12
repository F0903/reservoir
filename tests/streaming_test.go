package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reservoir/config"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Two halves, each big enough that net/http does not hold it back in a write buffer.
var (
	streamFirstHalf  = bytes.Repeat([]byte("a"), 64<<10)
	streamSecondHalf = bytes.Repeat([]byte("b"), 64<<10)
	streamWholeBody  = append(append([]byte{}, streamFirstHalf...), streamSecondHalf...)
)

// The proxy's own net/http server keeps the last few KB of a paused body in its
// write buffer, so the tests only expect part of the first half before the pause ends.
const streamProbe = 32 << 10

// pausedHandler is an upstream that sends the first half of the body, then waits for
// release before the second. If nothing releases it within 3 seconds it carries on by
// itself and timedOut reports true, so a proxy that holds the body back fails the test
// instead of hanging it.
func pausedHandler(t *testing.T) (handler http.Handler, release func(), timedOut func() bool, hits *atomic.Int64) {
	t.Helper()

	gate := make(chan struct{})
	release = sync.OnceFunc(func() { close(gate) })
	var late atomic.Bool
	timer := time.AfterFunc(3*time.Second, func() {
		late.Store(true)
		release()
	})
	t.Cleanup(func() {
		timer.Stop()
		release()
	})

	hits = &atomic.Int64{}
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Length", strconv.Itoa(len(streamWholeBody)))
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("ETag", `"stream-etag"`)
		w.Write(streamFirstHalf)
		w.(http.Flusher).Flush()
		<-gate
		w.Write(streamSecondHalf)
	})
	return handler, release, late.Load, hits
}

// startReading starts a request and reads the first bytes of the body from it.
func startReading(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })

	got := make([]byte, streamProbe)
	if _, err := io.ReadFull(resp.Body, got); err != nil || !bytes.Equal(got, streamWholeBody[:streamProbe]) {
		t.Fatalf("failed to read the first bytes of the body: %v", err)
	}
	return resp
}

func readRest(t *testing.T, resp *http.Response) {
	t.Helper()

	rest, err := io.ReadAll(resp.Body)
	if err != nil || !bytes.Equal(rest, streamWholeBody[streamProbe:]) {
		t.Fatalf("got %d bytes for the rest (err=%v), want %d", len(rest), err, len(streamWholeBody)-streamProbe)
	}
}

func TestClientReceivesBodyWhileDownloadIsStillRunning(t *testing.T) {
	for _, cacheType := range []config.CacheType{config.CacheTypeMemory, config.CacheTypeFile, config.CacheTypeHybrid} {
		t.Run(string(cacheType), func(t *testing.T) {
			handler, release, timedOut, hits := pausedHandler(t)
			upstream := httptest.NewServer(handler)
			t.Cleanup(upstream.Close)

			cfg := newFileCacheProxyConfig(t)
			cfg.Cache.Type.Overwrite(cacheType)
			client := newBareProxy(t, cfg, newTestUpstreamClient())
			target := upstream.URL + "/big.deb"

			resp := startReading(t, client, target)
			if timedOut() {
				t.Fatal("the client got its first bytes only after upstream had sent the whole file")
			}
			release()
			readRest(t, resp)

			cached, err := client.Get(target)
			if err != nil {
				t.Fatalf("second request failed: %v", err)
			}
			if body := readResponseBody(t, cached); body != string(streamWholeBody) || cached.Header.Get("X-Cache") != "HIT" {
				t.Fatalf("expected the streamed download to be cached whole, got X-Cache=%q and %d bytes", cached.Header.Get("X-Cache"), len(body))
			}
			if got := hits.Load(); got != 1 {
				t.Fatalf("expected 1 upstream request, got %d", got)
			}
		})
	}
}

func TestSecondClientJoinsRunningDownload(t *testing.T) {
	env := SetupTestEnv(t)
	handler, release, timedOut, hits := pausedHandler(t)
	env.Upstream.Config.Handler = handler
	env.Start()
	target := env.Upstream.URL + "/shared.deb"

	first := startReading(t, env.Client, target)
	second := startReading(t, env.Client, target) // Arrives while the download is paused halfway.
	if timedOut() {
		t.Fatal("a client got its first bytes only after upstream had sent the whole file")
	}
	release()
	readRest(t, first)
	readRest(t, second)

	if got := hits.Load(); got != 1 {
		t.Fatalf("expected both clients to share 1 upstream request, got %d", got)
	}
}

// The download never waits for a client. One that stops reading for a while still gets
// the exact body: from the socket buffers if they are large, otherwise from the finished
// cache entry once it has fallen out of the window of recent bytes.
func TestSlowClientStillGetsTheWholeBody(t *testing.T) {
	env := SetupTestEnv(t)

	// Much bigger than the window, and never repeating, so a wrong offset shows up.
	body := make([]byte, 24<<20)
	for i := range body {
		body[i] = byte(i ^ i>>8 ^ i>>16)
	}
	downloaded := make(chan struct{})
	var hits atomic.Int64
	env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("Cache-Control", "max-age=60")
		w.Write(body)
		close(downloaded)
	})
	env.Start()

	resp, err := env.Client.Get(env.Upstream.URL + "/huge.deb")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	start := make([]byte, 1<<20)
	if _, err := io.ReadFull(resp.Body, start); err != nil {
		t.Fatalf("failed to read the start of the body: %v", err)
	}

	// Stop reading until the proxy has the whole file.
	select {
	case <-downloaded:
	case <-time.After(10 * time.Second):
		t.Fatal("the download waited for the slow client")
	}
	time.Sleep(200 * time.Millisecond)

	rest, err := io.ReadAll(resp.Body)
	if err != nil || !bytes.Equal(append(start, rest...), body) {
		t.Fatalf("got %d bytes (err=%v) that differ from the %d byte body", len(start)+len(rest), err, len(body))
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("expected 1 upstream request, got %d", got)
	}
}

// The first download dies halfway through the body. The client already has the
// first half by then, so the proxy has to get it the rest from upstream.
func TestClientIsContinuedFromUpstreamWhenDownloadBreaks(t *testing.T) {
	tests := []struct {
		name           string
		upstreamRanges bool
	}{
		{name: "UpstreamSupportsRanges", upstreamRanges: true},
		{name: "UpstreamIgnoresRanges", upstreamRanges: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := SetupTestEnv(t)

			var requests atomic.Int64
			env.Upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "max-age=60")
				w.Header().Set("ETag", `"stream-etag"`)

				if requests.Add(1) == 1 {
					w.Header().Set("Content-Length", strconv.Itoa(len(streamWholeBody)))
					w.Write(streamFirstHalf)
					w.(http.Flusher).Flush()
					if conn, _, err := w.(http.Hijacker).Hijack(); err == nil {
						conn.Close()
					}
					return
				}

				var from int
				if _, err := fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-", &from); err != nil || r.Header.Get("If-Range") != `"stream-etag"` {
					t.Errorf("unexpected request for the rest: Range=%q If-Range=%q", r.Header.Get("Range"), r.Header.Get("If-Range"))
					return
				}
				if !tt.upstreamRanges {
					w.Write(streamWholeBody) // The same file from its first byte again.
					return
				}
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", from, len(streamWholeBody)-1, len(streamWholeBody)))
				w.WriteHeader(http.StatusPartialContent)
				w.Write(streamWholeBody[from:])
			})
			env.Start()

			resp, err := env.Client.Get(env.Upstream.URL + "/flaky.deb")
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil || !bytes.Equal(body, streamWholeBody) {
				t.Fatalf("got %d bytes (err=%v), want the whole body of %d bytes", len(body), err, len(streamWholeBody))
			}
			if got := requests.Load(); got != 2 {
				t.Fatalf("expected the broken download and one request for the rest, got %d upstream requests", got)
			}
		})
	}
}
