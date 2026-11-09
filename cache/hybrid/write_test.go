package hybrid

import (
	"bytes"
	"fmt"
	"io"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/utils"
	"sync"
	"testing"
	"time"
)

type gatedReader struct {
	release chan struct{}
	sent    bool
}

func (r *gatedReader) Read(p []byte) (int, error) {
	if r.sent {
		return 0, io.EOF
	}
	<-r.release
	r.sent = true
	p[0] = 'x'
	return 1, nil
}

func keyOnSameShard(key cache.CacheKey, shards uint32) cache.CacheKey {
	want := utils.Hex8ToIndex(key.Hex) % shards
	for i := 0; ; i++ {
		candidate := cache.FromString(fmt.Sprint("other-", i))
		if utils.Hex8ToIndex(candidate.Hex)%shards == want {
			return candidate
		}
	}
}

func TestHybridCache_StreamingDownloadDoesNotBlockMemoryShard(t *testing.T) {
	c := newTestHybridCache(t, time.Hour)
	setHybridMemoryCap(c, 64)

	slow := cache.FromString("slow")
	other := keyOnSameShard(slow, 16)

	entry, err := c.Cache(other, bytes.NewReader([]byte("cached")), time.Now().Add(time.Hour), TestMeta{})
	if err != nil {
		t.Fatalf("failed to seed entry: %v", err)
	}
	entry.Data.Close()

	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseOnce)

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		// Larger than the memory budget, so it streams straight to the file tier.
		data := cache.WithSizeHint(&gatedReader{release: release}, 128)
		if e, err := c.Cache(slow, data, time.Now().Add(time.Hour), TestMeta{}); err == nil {
			e.Data.Close()
		}
	}()
	time.Sleep(100 * time.Millisecond)

	got := make(chan error, 1)
	go func() {
		e, err := c.Get(other)
		if err == nil {
			e.Data.Close()
		}
		got <- err
	}()

	select {
	case err := <-got:
		if err != nil {
			t.Fatalf("failed to read unrelated key: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("unrelated key was blocked behind a streaming download")
	}

	releaseOnce()
	<-writeDone
}

func TestHybridCache_ReservationsCountBufferedDownloads(t *testing.T) {
	c := newTestHybridCache(t, time.Hour)
	setHybridMemoryCap(c, 100)

	first := cache.FromString("first")
	second := cache.FromString("second")

	if !c.reserveMemory(first, 60) {
		t.Fatal("expected the first download to fit")
	}
	if c.reserveMemory(second, 60) {
		t.Fatal("expected the second download to go to disk while the first is buffering")
	}

	c.bufferedBytes.Sub(60)
	if !c.reserveMemory(second, 60) {
		t.Fatal("expected room again after the first download finished")
	}
}

func TestHybridCache_PromotionSkippedWhenBudgetIsReserved(t *testing.T) {
	c := newTestHybridCache(t, time.Hour)
	setHybridMemoryCap(c, 100)

	key := cache.FromString("on-disk")
	data := []byte("file tier content")
	stored, err := c.file.Cache(key, bytes.NewReader(data), time.Now().Add(time.Hour), TestMeta{})
	if err != nil {
		t.Fatalf("failed to seed file entry: %v", err)
	}
	stored.Data.Close()

	if !c.reserveMemory(cache.FromString("buffering"), 100) {
		t.Fatal("expected reservation to succeed")
	}

	entry, err := c.Get(key)
	if err != nil {
		t.Fatalf("failed to read entry: %v", err)
	}
	if got := readEntryData(t, entry); !bytes.Equal(got, data) {
		t.Fatalf("got %q, want %q", got, data)
	}
	if got := c.memory.StoredBytes(); got != 0 {
		t.Fatalf("expected no promotion into a fully reserved budget, memory holds %d bytes", got)
	}
}

func TestHybridCache_FailedFileWriteCountsOneError(t *testing.T) {
	c := newTestHybridCache(t, time.Hour)

	// The file tier refuses empty bodies.
	if _, err := c.Cache(cache.FromString("empty"), unknownSizeReader(nil), time.Now().Add(time.Hour), TestMeta{}); err == nil {
		t.Fatal("expected the write to fail")
	}

	if got := c.metrics.Cache.CacheErrors.Get(); got != 1 {
		t.Fatalf("expected one cache error, got %d", got)
	}
}

func TestHybridCache_RecordsCleanupOncePerCycle(t *testing.T) {
	const interval = 20 * time.Millisecond

	m := metrics.NewMetrics()
	c := New[TestMeta](config.NewDefault(), m, t.TempDir(), 50, 1<<30, interval, 16, t.Context())
	t.Cleanup(c.Destroy)

	data := []byte("entry that expires")
	entry, err := c.Cache(cache.FromString("expires"), unknownSizeReader(data), time.Now().Add(10*time.Millisecond), TestMeta{})
	if err != nil {
		t.Fatalf("failed to cache entry: %v", err)
	}
	entry.Data.Close()

	start := time.Now()
	time.Sleep(300 * time.Millisecond)
	elapsed := time.Since(start)
	runs := m.Cache.CleanupRuns.Get()

	if got := m.Cache.BytesCleaned.Get(); got != int64(len(data)) {
		t.Fatalf("expected %d bytes cleaned, got %d", len(data), got)
	}
	if runs == 0 {
		t.Fatal("expected cleanup runs to be recorded")
	}
	if most := int64(elapsed/interval) + 1; runs > most {
		t.Fatalf("expected at most %d cleanup runs in %v, got %d; both tiers are counting", most, elapsed, runs)
	}
}

func TestHybridCache_UnbufferableEntriesGoToFile(t *testing.T) {
	oversized := bytes.Repeat([]byte("x"), 1200)
	unknown := []byte("unknown size content")

	tests := []struct {
		name         string
		maxCacheSize int64
		memoryCap    int64 // Zero keeps the default budget.
		data         []byte
		reader       io.Reader
	}{
		{"oversized", 2000, 1000, oversized, bytes.NewReader(oversized)},
		{"unknown size", 1024 * 1024 * 1024, 0, unknown, unknownSizeReader(unknown)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestHybridCacheWithMax(t, time.Hour, tt.maxCacheSize)
			if tt.memoryCap > 0 {
				setHybridMemoryCap(c, tt.memoryCap)
			}

			entry, err := c.Cache(cache.FromString(tt.name), tt.reader, time.Now().Add(time.Hour), TestMeta{})
			if err != nil {
				t.Fatalf("failed to cache entry: %v", err)
			}
			if got := readEntryData(t, entry); !bytes.Equal(got, tt.data) {
				t.Fatalf("got %q, want %q", got, tt.data)
			}

			if stats := c.memory.Stats(); stats.Entries != 0 || stats.Bytes != 0 {
				t.Fatalf("expected entry to skip memory, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
			}
			if stats := c.file.Stats(); stats.Entries != 1 || stats.Bytes != int64(len(tt.data)) {
				t.Fatalf("expected entry in file, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
			}
		})
	}
}
