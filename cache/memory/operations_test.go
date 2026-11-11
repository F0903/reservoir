package memory

import (
	"bytes"
	"errors"
	"fmt"
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

// gatedReader blocks until release is closed, then yields one byte.
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

func newOperationsTestCache(t *testing.T, shards int) *Cache[TestMeta] {
	t.Helper()
	c := New[TestMeta](config.NewDefault(), metrics.NewMetrics(), 100, 1<<40, time.Hour, shards, t.Context())
	t.Cleanup(c.Destroy)
	return c
}

func TestMemoryCache_EvictsWhenFullWithoutDeadlock(t *testing.T) {
	const entrySize, capacity = 100, 100 * 1000

	c := newOperationsTestCache(t, 1024)
	c.OverrideMemoryCapForTesting(capacity)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 5000 {
			key := cache.FromString(fmt.Sprint("key-", i))
			// No size hint, so eviction runs while the write holds its shard lock.
			data := struct{ io.Reader }{bytes.NewReader(make([]byte, entrySize))}
			entry, err := c.Cache(key, data, time.Now().Add(time.Hour), TestMeta{})
			if err == nil {
				entry.Data.Close()
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("writer deadlocked while evicting a full cache")
	}
	if got := c.StoredBytes(); got > capacity {
		t.Fatalf("stored %d bytes, want at most %d", got, capacity)
	}
}

func TestMemoryCache_SlowWriteDoesNotBlockOtherKeyOnShard(t *testing.T) {
	c := newOperationsTestCache(t, 1) // One shard, so every key shares a lock.

	other := cache.FromString("other")
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

func TestMemoryCache_CountsInFlightWritesAgainstBudget(t *testing.T) {
	c := newOperationsTestCache(t, 1024)
	c.OverrideMemoryCapForTesting(100)

	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseOnce)

	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		data := cache.WithSizeHint(&gatedReader{release: release}, 60)
		if e, err := c.Cache(cache.FromString("first"), data, time.Now().Add(time.Hour), TestMeta{}); err == nil {
			e.Data.Close()
		}
	}()
	time.Sleep(100 * time.Millisecond)

	ready := make(chan struct{})
	close(ready)
	_, err := c.Cache(cache.FromString("second"), cache.WithSizeHint(&gatedReader{release: ready}, 60), time.Now().Add(time.Hour), TestMeta{})
	if !errors.Is(err, cache.ErrCacheMemoryExceeded) {
		t.Fatalf("expected in-flight download to count against the budget, got err=%v", err)
	}

	releaseOnce()
	<-firstDone
}

func TestMemoryCache_EvictsToFitEntryLargerThanFreeSpace(t *testing.T) {
	c := newOperationsTestCache(t, 1024)
	c.OverrideMemoryCapForTesting(1000)

	// 750 of 1000 bytes used: below the 80% mark that eviction trims down to.
	for i := range 3 {
		entry, err := c.Cache(cache.FromString(fmt.Sprint("old-", i)), bytes.NewReader(make([]byte, 250)), time.Now().Add(time.Hour), TestMeta{})
		if err != nil {
			t.Fatalf("failed to seed entry %d: %v", i, err)
		}
		entry.Data.Close()
	}

	entry, err := c.Cache(cache.FromString("new"), bytes.NewReader(make([]byte, 300)), time.Now().Add(time.Hour), TestMeta{})
	if err != nil {
		t.Fatalf("expected old entries to be evicted to make room, got %v", err)
	}
	entry.Data.Close()

	if got := c.StoredBytes(); got > 1000 {
		t.Fatalf("stored %d bytes, want at most 1000", got)
	}
}

func TestMemoryTotalUsesLowestLimitOnOwnCgroupPath(t *testing.T) {
	const host = int64(8 << 30)
	const small, smaller = "1073741824\n", "536870912\n"

	tests := []struct {
		name   string
		proc   string            // Contents of /proc/self/cgroup. Empty means the file is missing.
		limits map[string]string // memory.max contents by cgroup directory.
		want   int64
	}{
		{"private namespace", "0::/\n", map[string]string{"/": small}, 1 << 30},
		{"host namespace", "0::/system.slice/docker-abc.scope\n", map[string]string{"/system.slice/docker-abc.scope": small}, 1 << 30},
		{"parent is lower", "0::/a/b\n", map[string]string{"/a": smaller, "/a/b": small}, 1 << 29},
		{"own limit is lower", "0::/a/b\n", map[string]string{"/a": small, "/a/b": smaller}, 1 << 29},
		{"unlimited with limited parent", "0::/a/b\n", map[string]string{"/a": small, "/a/b": "max\n"}, 1 << 30},
		{"other cgroups are ignored", "0::/a\n", map[string]string{"/other": smaller}, host},
		{"hybrid layout uses the v2 line", "12:memory:/old\n0::/a\n", map[string]string{"/a": small, "/old": smaller}, 1 << 30},
		{"limit above host", "0::/\n", map[string]string{"/": "17179869184\n"}, host},
		{"no limit", "0::/\n", map[string]string{"/": "max\n"}, host},
		{"garbage", "0::/\n", map[string]string{"/": "not-a-number\n"}, host},
		{"no limit files", "0::/a\n", nil, host},
		{"proc file missing falls back to the root", "", map[string]string{"/": small}, 1 << 30},
		{"proc path cannot escape the root", "0::/..\n", map[string]string{"/..": smaller}, host},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "cgroup")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			for dir, content := range tt.limits {
				full := filepath.Join(root, filepath.FromSlash(dir))
				if err := os.MkdirAll(full, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(full, "memory.max"), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			procFile := filepath.Join(base, "self-cgroup")
			if tt.proc != "" {
				if err := os.WriteFile(procFile, []byte(tt.proc), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			if got := memoryTotal(uint64(host), root, procFile); got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
