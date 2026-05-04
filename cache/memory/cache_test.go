package memory

import (
	"bytes"
	"io"
	"reservoir/cache"
	"reservoir/config"
	"testing"
	"time"
)

type TestMeta struct {
	ID string
}

func TestMemoryCache_Basic(t *testing.T) {
	ctx := t.Context()
	cfg := config.NewDefault()

	// 1% memory budget, large maxCacheSize, 1 min cleanup, 16 shards
	c := New[TestMeta](cfg, 1, 1024*1024*1024, time.Minute, 16, ctx)
	defer c.Destroy()

	key := cache.FromString("test-key")
	data := []byte("hello world")
	expires := time.Now().Add(time.Hour)
	meta := TestMeta{ID: "meta-1"}

	// Cache
	entry, err := c.Cache(key, bytes.NewReader(data), expires, meta)
	if err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	entry.Data.Close()

	if entry.Metadata.Size != int64(len(data)) {
		t.Errorf("Expected size %d, got %d", len(data), entry.Metadata.Size)
	}

	// Get
	retrieved, err := c.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	content, _ := io.ReadAll(retrieved.Data)
	retrieved.Data.Close()
	if !bytes.Equal(content, data) {
		t.Errorf("Expected data %s, got %s", data, content)
	}

	if retrieved.Metadata.Object.ID != "meta-1" {
		t.Errorf("Expected meta ID meta-1, got %s", retrieved.Metadata.Object.ID)
	}

	// Delete
	err = c.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = c.Get(key)
	if err != cache.ErrCacheEntryNotFound {
		t.Errorf("Expected cache.ErrCacheEntryNotFound, got %v", err)
	}
}

func TestMemoryCache_OverwriteUpdatesAccounting(t *testing.T) {
	ctx := t.Context()
	cfg := config.NewDefault()

	c := New[TestMeta](cfg, 1, 1024*1024*1024, time.Minute, 16, ctx)
	defer c.Destroy()

	key := cache.FromString("overwrite-key")
	firstData := []byte("first body")
	secondData := []byte("replacement body is larger")
	expires := time.Now().Add(time.Hour)

	first, err := c.Cache(key, bytes.NewReader(firstData), expires, TestMeta{ID: "first"})
	if err != nil {
		t.Fatalf("first cache failed: %v", err)
	}
	first.Data.Close()

	if got := c.byteSize.Get(); got != int64(len(firstData)) {
		t.Fatalf("expected first byte size %d, got %d", len(firstData), got)
	}
	if got := len(c.entries); got != 1 {
		t.Fatalf("expected 1 entry after first cache, got %d", got)
	}

	second, err := c.Cache(key, bytes.NewReader(secondData), expires, TestMeta{ID: "second"})
	if err != nil {
		t.Fatalf("second cache failed: %v", err)
	}
	second.Data.Close()

	if got := c.byteSize.Get(); got != int64(len(secondData)) {
		t.Fatalf("expected replacement byte size %d, got %d", len(secondData), got)
	}
	if got := len(c.entries); got != 1 {
		t.Fatalf("expected overwrite to keep 1 entry, got %d", got)
	}

	retrieved, err := c.Get(key)
	if err != nil {
		t.Fatalf("get after overwrite failed: %v", err)
	}
	defer retrieved.Data.Close()

	content, err := io.ReadAll(retrieved.Data)
	if err != nil {
		t.Fatalf("failed to read overwritten data: %v", err)
	}
	if !bytes.Equal(content, secondData) {
		t.Fatalf("expected overwritten data %q, got %q", secondData, content)
	}
	if retrieved.Metadata.Object.ID != "second" {
		t.Fatalf("expected replacement metadata, got %q", retrieved.Metadata.Object.ID)
	}
}

func TestMemoryCache_ReturnsMetadataSnapshots(t *testing.T) {
	ctx := t.Context()
	cfg := config.NewDefault()

	c := New[TestMeta](cfg, 1, 1024*1024*1024, time.Minute, 16, ctx)
	defer c.Destroy()

	key := cache.FromString("metadata-snapshot-key")
	entry, err := c.Cache(key, bytes.NewReader([]byte("snapshot body")), time.Now().Add(time.Hour), TestMeta{ID: "original"})
	if err != nil {
		t.Fatalf("cache failed: %v", err)
	}
	entry.Data.Close()
	entry.Metadata.Object.ID = "mutated"

	retrieved, err := c.Get(key)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	retrieved.Data.Close()
	if retrieved.Metadata.Object.ID != "original" {
		t.Fatalf("expected cached metadata to remain original, got %q", retrieved.Metadata.Object.ID)
	}
}

func TestMemoryCache_ClearRemovesEntriesAndUpdatesStats(t *testing.T) {
	ctx := t.Context()
	cfg := config.NewDefault()
	maxCacheSize := int64(1024 * 1024 * 1024)

	c := New[TestMeta](cfg, 1, maxCacheSize, time.Minute, 16, ctx)
	defer c.Destroy()

	firstData := []byte("first")
	secondData := []byte("second body")
	expires := time.Now().Add(time.Hour)

	first, err := c.Cache(cache.FromString("clear-memory-first"), bytes.NewReader(firstData), expires, TestMeta{ID: "first"})
	if err != nil {
		t.Fatalf("first cache failed: %v", err)
	}
	first.Data.Close()

	second, err := c.Cache(cache.FromString("clear-memory-second"), bytes.NewReader(secondData), expires, TestMeta{ID: "second"})
	if err != nil {
		t.Fatalf("second cache failed: %v", err)
	}
	second.Data.Close()

	stats := c.Stats()
	if stats.Entries != 2 {
		t.Fatalf("expected 2 entries before clear, got %d", stats.Entries)
	}
	if stats.Bytes != int64(len(firstData)+len(secondData)) {
		t.Fatalf("expected %d cached bytes before clear, got %d", len(firstData)+len(secondData), stats.Bytes)
	}
	if stats.MaxBytes != maxCacheSize {
		t.Fatalf("expected max cache size %d, got %d", maxCacheSize, stats.MaxBytes)
	}
	if stats.MemoryCapBytes <= 0 {
		t.Fatalf("expected positive memory cap, got %d", stats.MemoryCapBytes)
	}

	if err := c.Clear(); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	stats = c.Stats()
	if stats.Entries != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", stats.Entries)
	}
	if stats.Bytes != 0 {
		t.Fatalf("expected 0 cached bytes after clear, got %d", stats.Bytes)
	}
	if _, err := c.Get(cache.FromString("clear-memory-first")); err != cache.ErrCacheEntryNotFound {
		t.Fatalf("expected cleared first entry to be missing, got %v", err)
	}
	if _, err := c.Get(cache.FromString("clear-memory-second")); err != cache.ErrCacheEntryNotFound {
		t.Fatalf("expected cleared second entry to be missing, got %v", err)
	}
	if err := c.Clear(); err != nil {
		t.Fatalf("clear on empty cache failed: %v", err)
	}
}
