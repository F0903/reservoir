package file

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
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

func newOperationsTestCache(t *testing.T, dir string, shards int) *Cache[TestMeta] {
	t.Helper()
	c := New[TestMeta](config.NewDefault(), metrics.NewMetrics(), dir, 1<<30, time.Hour, shards, t.Context())
	t.Cleanup(c.Destroy)
	return c
}

func seedFileEntry(t *testing.T, c *Cache[TestMeta], key cache.CacheKey, data string) {
	t.Helper()
	entry, err := c.Cache(key, bytes.NewReader([]byte(data)), time.Now().Add(time.Hour), TestMeta{})
	if err != nil {
		t.Fatalf("failed to seed entry: %v", err)
	}
	entry.Data.Close()
}

func TestFileCache_SlowWriteDoesNotBlockOtherKeyOnShard(t *testing.T) {
	c := newOperationsTestCache(t, t.TempDir(), 1) // One shard, so every key shares a lock.

	other := cache.FromString("other")
	seedFileEntry(t, c, other, "cached")

	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseOnce)

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		if e, err := c.Cache(cache.FromString("slow"), &gatedReader{release: release}, time.Now().Add(time.Hour), TestMeta{}); err == nil {
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
		t.Fatal("unrelated key was blocked behind a slow write")
	}

	releaseOnce()
	<-writeDone
}

func TestFileCache_CorruptSidecarCountsError(t *testing.T) {
	dir := t.TempDir()
	key := cache.FromString("corrupt")
	if err := os.WriteFile(filepath.Join(dir, key.Hex+".meta.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newOperationsTestCache(t, dir, 8)

	if got := c.metrics.Cache.CacheErrors.Get(); got != 1 {
		t.Fatalf("expected one cache error, got %d", got)
	}
}

func TestFileCache_FailedSidecarWriteCountsError(t *testing.T) {
	c := newOperationsTestCache(t, t.TempDir(), 8)
	key := cache.FromString("no-sidecar")
	if err := os.Mkdir(c.metadataPath(key), 0o755); err != nil { // A directory can't be created as a file.
		t.Fatal(err)
	}

	seedFileEntry(t, c, key, "body")

	if got := c.metrics.Cache.CacheErrors.Get(); got != 1 {
		t.Fatalf("expected one cache error, got %d", got)
	}
}

func TestFileCache_FailedDeleteCountsOneError(t *testing.T) {
	tests := []struct {
		name string
		path func(*Cache[TestMeta], cache.CacheKey) string
	}{
		{"data file", (*Cache[TestMeta]).dataPath},
		{"sidecar", (*Cache[TestMeta]).metadataPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newOperationsTestCache(t, t.TempDir(), 8)
			key := cache.FromString("undeletable")
			seedFileEntry(t, c, key, "body")

			// A non-empty directory can't be removed with os.Remove on any platform.
			path := tt.path(c, key)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(path, "child"), 0o755); err != nil {
				t.Fatal(err)
			}

			if err := c.Delete(key); err == nil {
				t.Fatal("expected the delete to fail")
			}
			if got := c.metrics.Cache.CacheErrors.Get(); got != 1 {
				t.Fatalf("expected one cache error, got %d", got)
			}
		})
	}
}

func TestFileCache_StartupRemovesTempFiles(t *testing.T) {
	dir := t.TempDir()
	leftover := filepath.Join(dir, "abc.123.tmp")
	if err := os.WriteFile(leftover, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}

	newOperationsTestCache(t, dir, 8)

	if _, err := os.Stat(leftover); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected leftover temp file to be removed, stat err=%v", err)
	}
}

func TestFileCache_StartupDiscardsEntryWhoseSizeDoesNotMatch(t *testing.T) {
	dir := t.TempDir()
	key := cache.FromString("entry")

	first := newOperationsTestCache(t, dir, 8)
	seedFileEntry(t, first, key, "original")
	first.Destroy()

	// A crash between replacing the body and rewriting the sidecar leaves them disagreeing.
	if err := os.WriteFile(first.dataPath(key), []byte("a longer replacement body"), 0o644); err != nil {
		t.Fatal(err)
	}

	second := newOperationsTestCache(t, dir, 8)
	if _, err := second.Get(key); !errors.Is(err, cache.ErrCacheEntryNotFound) {
		t.Fatalf("expected mismatched entry to be discarded, got err=%v", err)
	}
	if _, err := os.Stat(second.dataPath(key)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected mismatched body to be removed, stat err=%v", err)
	}
}
