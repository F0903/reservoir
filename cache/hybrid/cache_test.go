package hybrid

import (
	"bytes"
	"io"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/utils/duration"
	"testing"
	"time"
)

type TestMeta struct {
	ID string
}

func useFreshCacheMetrics(t *testing.T) {
	t.Helper()

	previous := metrics.Global
	metrics.Global = metrics.NewMetrics()
	t.Cleanup(func() {
		metrics.Global = previous
	})
}

func newTestHybridCache(t *testing.T, demoteAfter time.Duration) *Cache[TestMeta] {
	t.Helper()

	return newTestHybridCacheWithMax(t, demoteAfter, 1024*1024*1024)
}

func newTestHybridCacheWithMax(t *testing.T, demoteAfter time.Duration, maxCacheSize int64) *Cache[TestMeta] {
	t.Helper()

	cfg := config.NewDefault()
	cfg.Cache.Hybrid.DemoteAfter.Overwrite(duration.Duration(demoteAfter))

	c := New[TestMeta](cfg, t.TempDir(), 50, maxCacheSize, time.Hour, 16, t.Context())
	t.Cleanup(c.Destroy)
	return c
}

func setHybridMemoryCap(c *Cache[TestMeta], memoryCap int64) {
	c.memory.OverrideMemoryCapForTesting(memoryCap)
}

func setHybridMemoryLastAccess(t *testing.T, c *Cache[TestMeta], key cache.CacheKey, lastAccess time.Time) {
	t.Helper()

	if !c.memory.OverrideEntryLastAccessForTesting(key, lastAccess) {
		t.Fatalf("expected memory entry for key %s", key.Hex)
	}
}

func readEntryData(t *testing.T, entry *cache.Entry[TestMeta]) []byte {
	t.Helper()
	defer entry.Data.Close()

	content, err := io.ReadAll(entry.Data)
	if err != nil {
		t.Fatalf("failed to read cached data: %v", err)
	}
	return content
}

type noSizeReader struct {
	io.Reader
}

func unknownSizeReader(data []byte) io.Reader {
	return noSizeReader{Reader: bytes.NewReader(data)}
}

func TestHybridCache_WritesAndReadsMemoryFirst(t *testing.T) {
	c := newTestHybridCache(t, time.Hour)

	key := cache.FromString("hybrid-key")
	data := []byte("hello hybrid world")
	expires := time.Now().Add(time.Hour)

	entry, err := c.Cache(key, bytes.NewReader(data), expires, TestMeta{ID: "hybrid-meta"})
	if err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	entry.Data.Close()

	if stats := c.memory.Stats(); stats.Entries != 1 {
		t.Fatalf("expected 1 memory entry after cache write, got %d", stats.Entries)
	}
	if stats := c.file.Stats(); stats.Entries != 0 {
		t.Fatalf("expected 0 file entries after memory-first write, got %d", stats.Entries)
	}

	retrieved, err := c.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if content := readEntryData(t, retrieved); !bytes.Equal(content, data) {
		t.Fatalf("expected data %q, got %q", data, content)
	}
	if retrieved.Metadata.Object.ID != "hybrid-meta" {
		t.Fatalf("expected metadata ID hybrid-meta, got %q", retrieved.Metadata.Object.ID)
	}

	stats := c.Stats()
	if stats.Entries != 1 {
		t.Fatalf("expected 1 total entry, got %d", stats.Entries)
	}
	if stats.Bytes != int64(len(data)) {
		t.Fatalf("expected %d total bytes, got %d", len(data), stats.Bytes)
	}
	if stats.MemoryCapBytes <= 0 {
		t.Fatalf("expected positive memory cap, got %d", stats.MemoryCapBytes)
	}
}

func TestHybridCache_UnknownSizedEntryStaysInMemoryWhenItFits(t *testing.T) {
	c := newTestHybridCacheWithMax(t, time.Hour, 2000)
	setHybridMemoryCap(c, 1000)

	key := cache.FromString("hybrid-unknown-memory-key")
	data := bytes.Repeat([]byte("u"), 800)
	entry, err := c.Cache(key, unknownSizeReader(data), time.Now().Add(time.Hour), TestMeta{ID: "unknown-memory"})
	if err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	if content := readEntryData(t, entry); !bytes.Equal(content, data) {
		t.Fatalf("expected memory data %q, got %q", data, content)
	}

	if stats := c.memory.Stats(); stats.Entries != 1 || stats.Bytes != int64(len(data)) {
		t.Fatalf("expected unknown-sized entry in memory, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}
	if stats := c.file.Stats(); stats.Entries != 0 || stats.Bytes != 0 {
		t.Fatalf("expected no file entry, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}
}

func TestHybridCache_UnknownSizedEntrySpillsWhenAvailableMemoryIsExceeded(t *testing.T) {
	c := newTestHybridCacheWithMax(t, time.Hour, 2000)
	setHybridMemoryCap(c, 1000)

	memoryData := bytes.Repeat([]byte("m"), 700)
	memoryEntry, err := c.Cache(cache.FromString("hybrid-unknown-resident-key"), bytes.NewReader(memoryData), time.Now().Add(time.Hour), TestMeta{ID: "resident"})
	if err != nil {
		t.Fatalf("Cache resident entry failed: %v", err)
	}
	memoryEntry.Data.Close()

	fileKey := cache.FromString("hybrid-unknown-spill-key")
	fileData := bytes.Repeat([]byte("f"), 500)
	fileEntry, err := c.Cache(fileKey, unknownSizeReader(fileData), time.Now().Add(time.Hour), TestMeta{ID: "unknown-file"})
	if err != nil {
		t.Fatalf("Cache unknown spill entry failed: %v", err)
	}
	if content := readEntryData(t, fileEntry); !bytes.Equal(content, fileData) {
		t.Fatalf("expected streamed file data %q, got %q", fileData, content)
	}

	if stats := c.memory.Stats(); stats.Entries != 1 || stats.Bytes != int64(len(memoryData)) {
		t.Fatalf("expected resident memory entry to remain, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}
	if stats := c.file.Stats(); stats.Entries != 1 || stats.Bytes != int64(len(fileData)) {
		t.Fatalf("expected unknown-sized entry in file, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}
}

func TestHybridCache_StoresOversizedEntriesInFile(t *testing.T) {
	c := newTestHybridCacheWithMax(t, time.Hour, 2000)
	setHybridMemoryCap(c, 1000)

	key := cache.FromString("hybrid-oversized-key")
	data := bytes.Repeat([]byte("x"), 1200)
	entry, err := c.Cache(key, bytes.NewReader(data), time.Now().Add(time.Hour), TestMeta{ID: "oversized"})
	if err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	if content := readEntryData(t, entry); !bytes.Equal(content, data) {
		t.Fatalf("expected streamed file data %q, got %q", data, content)
	}

	if stats := c.memory.Stats(); stats.Entries != 0 || stats.Bytes != 0 {
		t.Fatalf("expected oversized entry to skip memory, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}
	if stats := c.file.Stats(); stats.Entries != 1 || stats.Bytes != int64(len(data)) {
		t.Fatalf("expected oversized entry in file, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}
}

func TestHybridCache_StoresInFileWhenRecentMemoryWouldExceedLimit(t *testing.T) {
	c := newTestHybridCacheWithMax(t, time.Hour, 2000)
	setHybridMemoryCap(c, 1000)

	memoryKey := cache.FromString("hybrid-recent-memory-key")
	memoryData := bytes.Repeat([]byte("m"), 700)
	entry, err := c.Cache(memoryKey, bytes.NewReader(memoryData), time.Now().Add(time.Hour), TestMeta{ID: "memory"})
	if err != nil {
		t.Fatalf("Cache memory entry failed: %v", err)
	}
	entry.Data.Close()

	fileKey := cache.FromString("hybrid-file-spill-key")
	fileData := bytes.Repeat([]byte("f"), 500)
	entry, err = c.Cache(fileKey, bytes.NewReader(fileData), time.Now().Add(time.Hour), TestMeta{ID: "file"})
	if err != nil {
		t.Fatalf("Cache spill entry failed: %v", err)
	}
	entry.Data.Close()

	if stats := c.memory.Stats(); stats.Entries != 1 || stats.Bytes != int64(len(memoryData)) {
		t.Fatalf("expected recent memory entry to stay within limit, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}
	if stats := c.file.Stats(); stats.Entries != 1 || stats.Bytes != int64(len(fileData)) {
		t.Fatalf("expected overflow entry to spill to file, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}
}

func TestHybridCache_DemotesIdleMemoryEntriesToMakeRoom(t *testing.T) {
	c := newTestHybridCacheWithMax(t, time.Hour, 2000)
	setHybridMemoryCap(c, 1000)

	idleKey := cache.FromString("hybrid-idle-pressure-key")
	idleData := bytes.Repeat([]byte("i"), 700)
	entry, err := c.Cache(idleKey, bytes.NewReader(idleData), time.Now().Add(time.Hour), TestMeta{ID: "idle"})
	if err != nil {
		t.Fatalf("Cache idle entry failed: %v", err)
	}
	entry.Data.Close()
	setHybridMemoryLastAccess(t, c, idleKey, time.Now().Add(-2*time.Hour))

	newKey := cache.FromString("hybrid-new-memory-key")
	newData := bytes.Repeat([]byte("n"), 500)
	entry, err = c.Cache(newKey, bytes.NewReader(newData), time.Now().Add(time.Hour), TestMeta{ID: "new"})
	if err != nil {
		t.Fatalf("Cache new entry failed: %v", err)
	}
	entry.Data.Close()

	if stats := c.memory.Stats(); stats.Entries != 1 || stats.Bytes != int64(len(newData)) {
		t.Fatalf("expected new entry to fit in memory after demotion, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}
	if stats := c.file.Stats(); stats.Entries != 1 || stats.Bytes != int64(len(idleData)) {
		t.Fatalf("expected idle entry to be demoted to file, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}
}

func TestHybridCache_DeleteSucceedsFromEitherTier(t *testing.T) {
	c := newTestHybridCacheWithMax(t, time.Hour, 2000)
	setHybridMemoryCap(c, 1000)

	memoryKey := cache.FromString("hybrid-delete-memory-key")
	entry, err := c.Cache(memoryKey, bytes.NewReader([]byte("memory")), time.Now().Add(time.Hour), TestMeta{ID: "memory"})
	if err != nil {
		t.Fatalf("Cache memory entry failed: %v", err)
	}
	entry.Data.Close()

	if err := c.Delete(memoryKey); err != nil {
		t.Fatalf("Delete memory-tier entry failed: %v", err)
	}
	if stats := c.Stats(); stats.Entries != 0 || stats.Bytes != 0 {
		t.Fatalf("expected memory-tier delete to clear cache, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}

	fileKey := cache.FromString("hybrid-delete-file-key")
	fileData := bytes.Repeat([]byte("f"), 1200)
	entry, err = c.Cache(fileKey, bytes.NewReader(fileData), time.Now().Add(time.Hour), TestMeta{ID: "file"})
	if err != nil {
		t.Fatalf("Cache file entry failed: %v", err)
	}
	entry.Data.Close()

	if err := c.Delete(fileKey); err != nil {
		t.Fatalf("Delete file-tier entry failed: %v", err)
	}
	if stats := c.Stats(); stats.Entries != 0 || stats.Bytes != 0 {
		t.Fatalf("expected file-tier delete to clear cache, got entries=%d bytes=%d", stats.Entries, stats.Bytes)
	}
}

func TestHybridCache_DemotesIdleMemoryEntriesToFile(t *testing.T) {
	c := newTestHybridCache(t, time.Hour)

	key := cache.FromString("hybrid-demote-key")
	data := []byte("hybrid demotion body")
	entry, err := c.Cache(key, bytes.NewReader(data), time.Now().Add(time.Hour), TestMeta{ID: "demote"})
	if err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	entry.Data.Close()

	setHybridMemoryLastAccess(t, c, key, time.Now().Add(-2*time.Hour))
	c.demoteIdleEntries()

	if stats := c.memory.Stats(); stats.Entries != 0 {
		t.Fatalf("expected memory entry to be demoted, got %d memory entries", stats.Entries)
	}
	if stats := c.file.Stats(); stats.Entries != 1 {
		t.Fatalf("expected demoted file entry, got %d file entries", stats.Entries)
	}

	fileEntry, err := c.file.Get(key)
	if err != nil {
		t.Fatalf("expected file cache hit after demotion: %v", err)
	}
	if content := readEntryData(t, fileEntry); !bytes.Equal(content, data) {
		t.Fatalf("expected demoted data %q, got %q", data, content)
	}
}

func TestHybridCache_RecentAccessPreventsDemotion(t *testing.T) {
	c := newTestHybridCache(t, time.Hour)

	key := cache.FromString("hybrid-recent-key")
	data := []byte("recent body")
	entry, err := c.Cache(key, bytes.NewReader(data), time.Now().Add(time.Hour), TestMeta{ID: "recent"})
	if err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	entry.Data.Close()

	setHybridMemoryLastAccess(t, c, key, time.Now().Add(-2*time.Hour))
	retrieved, err := c.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	retrieved.Data.Close()

	c.demoteIdleEntries()

	if stats := c.memory.Stats(); stats.Entries != 1 {
		t.Fatalf("expected recent access to keep entry in memory, got %d memory entries", stats.Entries)
	}
	if stats := c.file.Stats(); stats.Entries != 0 {
		t.Fatalf("expected no file entries after recent access, got %d", stats.Entries)
	}
}

func TestHybridCache_FileHitPromotesEntryBackToMemory(t *testing.T) {
	c := newTestHybridCache(t, time.Hour)

	key := cache.FromString("hybrid-promote-key")
	data := []byte("promote body")
	entry, err := c.Cache(key, bytes.NewReader(data), time.Now().Add(time.Hour), TestMeta{ID: "promote"})
	if err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	entry.Data.Close()

	setHybridMemoryLastAccess(t, c, key, time.Now().Add(-2*time.Hour))
	c.demoteIdleEntries()

	retrieved, err := c.Get(key)
	if err != nil {
		t.Fatalf("Get after demotion failed: %v", err)
	}
	if content := readEntryData(t, retrieved); !bytes.Equal(content, data) {
		t.Fatalf("expected promoted data %q, got %q", data, content)
	}

	if stats := c.memory.Stats(); stats.Entries != 1 {
		t.Fatalf("expected file hit to promote entry to memory, got %d memory entries", stats.Entries)
	}
	if stats := c.file.Stats(); stats.Entries != 1 {
		t.Fatalf("expected promoted file entry to remain shadowed, got %d file entries", stats.Entries)
	}
}

func TestHybridCache_DemotionOverwritesShadowedFileEntry(t *testing.T) {
	c := newTestHybridCache(t, time.Hour)

	key := cache.FromString("hybrid-shadowed-key")
	firstData := []byte("first file value")
	firstEntry, err := c.Cache(key, bytes.NewReader(firstData), time.Now().Add(time.Hour), TestMeta{ID: "first"})
	if err != nil {
		t.Fatalf("Cache first entry failed: %v", err)
	}
	firstEntry.Data.Close()
	setHybridMemoryLastAccess(t, c, key, time.Now().Add(-2*time.Hour))
	c.demoteIdleEntries()

	secondData := []byte("new memory value")
	secondEntry, err := c.Cache(key, bytes.NewReader(secondData), time.Now().Add(time.Hour), TestMeta{ID: "second"})
	if err != nil {
		t.Fatalf("Cache second entry failed: %v", err)
	}
	secondEntry.Data.Close()

	if stats := c.memory.Stats(); stats.Entries != 1 {
		t.Fatalf("expected newer entry in memory, got %d memory entries", stats.Entries)
	}
	if stats := c.file.Stats(); stats.Entries != 1 {
		t.Fatalf("expected older file entry to remain shadowed, got %d file entries", stats.Entries)
	}

	setHybridMemoryLastAccess(t, c, key, time.Now().Add(-2*time.Hour))
	c.demoteIdleEntries()

	if stats := c.memory.Stats(); stats.Entries != 0 {
		t.Fatalf("expected shadowing memory entry to demote, got %d memory entries", stats.Entries)
	}
	if stats := c.file.Stats(); stats.Entries != 1 {
		t.Fatalf("expected demotion to overwrite one file entry, got %d file entries", stats.Entries)
	}

	fileEntry, err := c.file.Get(key)
	if err != nil {
		t.Fatalf("expected file entry after demotion: %v", err)
	}
	if content := readEntryData(t, fileEntry); !bytes.Equal(content, secondData) {
		t.Fatalf("expected demoted file data %q, got %q", secondData, content)
	}
}

func TestHybridCache_BackgroundDemoterMovesIdleEntries(t *testing.T) {
	c := newTestHybridCache(t, 20*time.Millisecond)

	key := cache.FromString("hybrid-background-demote-key")
	entry, err := c.Cache(key, bytes.NewReader([]byte("background body")), time.Now().Add(time.Hour), TestMeta{ID: "background"})
	if err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	entry.Data.Close()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if c.file.Stats().Entries == 1 && c.memory.Stats().Entries == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected background demoter to move entry to file; memory=%d file=%d", c.memory.Stats().Entries, c.file.Stats().Entries)
}

func TestHybridCache_MetricsTrackShadowedFileAcrossDemotionAndPromotion(t *testing.T) {
	useFreshCacheMetrics(t)
	c := newTestHybridCache(t, time.Hour)

	key := cache.FromString("hybrid-metrics-key")
	data := []byte("hybrid metrics body")
	entry, err := c.Cache(key, bytes.NewReader(data), time.Now().Add(time.Hour), TestMeta{ID: "metrics"})
	if err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	entry.Data.Close()

	if got := metrics.Global.Cache.CacheEntries.Get(); got != 1 {
		t.Fatalf("expected global entry count 1 after hybrid cache write, got %d", got)
	}
	if got := metrics.Global.Cache.BytesCached.Get(); got != int64(len(data)) {
		t.Fatalf("expected global cached bytes %d, got %d", len(data), got)
	}

	setHybridMemoryLastAccess(t, c, key, time.Now().Add(-2*time.Hour))
	c.demoteIdleEntries()
	if got := metrics.Global.Cache.CacheEntries.Get(); got != 1 {
		t.Fatalf("expected global entry count 1 after demotion, got %d", got)
	}
	if got := metrics.Global.Cache.BytesCached.Get(); got != int64(len(data)) {
		t.Fatalf("expected global cached bytes %d after demotion, got %d", len(data), got)
	}

	retrieved, err := c.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	retrieved.Data.Close()
	if got := metrics.Global.Cache.CacheHits.Get(); got != 1 {
		t.Fatalf("expected one externally visible cache hit, got %d", got)
	}
	if got := metrics.Global.Cache.CacheEntries.Get(); got != 2 {
		t.Fatalf("expected memory promotion to track the shadowed file entry, got %d", got)
	}
	if got := metrics.Global.Cache.BytesCached.Get(); got != int64(len(data))*2 {
		t.Fatalf("expected global cached bytes %d after shadowed promotion, got %d", len(data)*2, got)
	}
}

func TestHybridCache_MemoryTierEvictionPreservesGlobalByteMetrics(t *testing.T) {
	useFreshCacheMetrics(t)
	c := newTestHybridCacheWithMax(t, time.Hour, 3000)
	setHybridMemoryCap(c, 1000)

	fileData := bytes.Repeat([]byte("f"), 1200)
	fileEntry, err := c.Cache(cache.FromString("hybrid-metric-file-key"), bytes.NewReader(fileData), time.Now().Add(time.Hour), TestMeta{ID: "file"})
	if err != nil {
		t.Fatalf("Cache file entry failed: %v", err)
	}
	fileEntry.Data.Close()

	memoryData := bytes.Repeat([]byte("m"), 1000)
	memoryEntry, err := c.Cache(cache.FromString("hybrid-metric-memory-key"), bytes.NewReader(memoryData), time.Now().Add(time.Hour), TestMeta{ID: "memory"})
	if err != nil {
		t.Fatalf("Cache memory entry failed: %v", err)
	}
	memoryEntry.Data.Close()

	newMemoryData := bytes.Repeat([]byte("n"), 100)
	newMemoryEntry, err := c.Cache(cache.FromString("hybrid-metric-new-memory-key"), bytes.NewReader(newMemoryData), time.Now().Add(time.Hour), TestMeta{ID: "new"})
	if err != nil {
		t.Fatalf("Cache new memory entry failed: %v", err)
	}
	newMemoryEntry.Data.Close()

	expectedBytes := int64(len(fileData) + len(newMemoryData))
	if got := metrics.Global.Cache.BytesCached.Get(); got != expectedBytes {
		t.Fatalf("expected global cached bytes %d after memory-tier eviction, got %d", expectedBytes, got)
	}
	if stats := c.Stats(); stats.Bytes != expectedBytes {
		t.Fatalf("expected hybrid stats bytes %d after memory-tier eviction, got %d", expectedBytes, stats.Bytes)
	}
}

func TestHybridCache_FileTierEvictionPreservesGlobalByteMetrics(t *testing.T) {
	useFreshCacheMetrics(t)
	c := newTestHybridCacheWithMax(t, time.Hour, 1500)
	setHybridMemoryCap(c, 1000)

	memoryData := bytes.Repeat([]byte("m"), 700)
	memoryEntry, err := c.Cache(cache.FromString("hybrid-file-eviction-memory-key"), bytes.NewReader(memoryData), time.Now().Add(time.Hour), TestMeta{ID: "memory"})
	if err != nil {
		t.Fatalf("Cache memory entry failed: %v", err)
	}
	memoryEntry.Data.Close()

	fileData := bytes.Repeat([]byte("f"), 1200)
	fileEntry, err := c.Cache(cache.FromString("hybrid-file-eviction-file-key"), bytes.NewReader(fileData), time.Now().Add(time.Hour), TestMeta{ID: "file"})
	if err != nil {
		t.Fatalf("Cache file entry failed: %v", err)
	}
	fileEntry.Data.Close()

	expectedBytes := int64(len(memoryData))
	if got := metrics.Global.Cache.BytesCached.Get(); got != expectedBytes {
		t.Fatalf("expected global cached bytes %d after file-tier eviction, got %d", expectedBytes, got)
	}
	if stats := c.Stats(); stats.Bytes != expectedBytes {
		t.Fatalf("expected hybrid stats bytes %d after file-tier eviction, got %d", expectedBytes, stats.Bytes)
	}
}
