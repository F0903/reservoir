package cache

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reservoir/config"
	"testing"
	"time"
)

func TestFileCache_Basic(t *testing.T) {
	ctx := t.Context()
	cfg := config.NewDefault()

	tmpDir, err := os.MkdirTemp("", "reservoir-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	c := NewFileCache[TestMeta](cfg, tmpDir, 1024*1024*1024, time.Minute, 16, ctx)
	defer c.Destroy()

	key := FromString("test-key")
	data := []byte("hello file world")
	expires := time.Now().Add(time.Hour)
	meta := TestMeta{ID: "meta-file"}

	// Cache
	entry, err := c.Cache(key, bytes.NewReader(data), expires, meta)
	if err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	entry.Data.Close()

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

	// Delete
	err = c.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = c.Get(key)
	if err != ErrCacheEntryNotFound {
		t.Errorf("Expected ErrCacheEntryNotFound, got %v", err)
	}
}

func TestFileCache_OverwriteUpdatesAccounting(t *testing.T) {
	ctx := t.Context()
	cfg := config.NewDefault()

	tmpDir := t.TempDir()
	c := NewFileCache[TestMeta](cfg, tmpDir, 1024*1024*1024, time.Minute, 16, ctx)
	defer c.Destroy()

	key := FromString("overwrite-key")
	firstData := []byte("first file body")
	secondData := []byte("replacement file body is larger")
	expires := time.Now().Add(time.Hour)

	first, err := c.Cache(key, bytes.NewReader(firstData), expires, TestMeta{ID: "first"})
	if err != nil {
		t.Fatalf("first cache failed: %v", err)
	}
	first.Data.Close()

	if got := c.byteSize.Get(); got != int64(len(firstData)) {
		t.Fatalf("expected first byte size %d, got %d", len(firstData), got)
	}
	if got := len(c.entriesMetadata); got != 1 {
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
	if got := len(c.entriesMetadata); got != 1 {
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

func TestFileCache_LoadsMetadataSidecarsOnRestart(t *testing.T) {
	ctx := t.Context()
	cfg := config.NewDefault()

	tmpDir := t.TempDir()
	key := FromString("restart-key")
	data := []byte("restart body")
	expires := time.Now().Add(time.Hour)

	firstCache := NewFileCache[TestMeta](cfg, tmpDir, 1024*1024*1024, time.Hour, 16, ctx)
	firstEntry, err := firstCache.Cache(key, bytes.NewReader(data), expires, TestMeta{ID: "restart-meta"})
	if err != nil {
		t.Fatalf("cache before restart failed: %v", err)
	}
	firstEntry.Data.Close()
	firstCache.Destroy()

	secondCache := NewFileCache[TestMeta](cfg, tmpDir, 1024*1024*1024, time.Hour, 16, ctx)
	defer secondCache.Destroy()

	retrieved, err := secondCache.Get(key)
	if err != nil {
		t.Fatalf("get after restart failed: %v", err)
	}
	defer retrieved.Data.Close()

	content, err := io.ReadAll(retrieved.Data)
	if err != nil {
		t.Fatalf("failed to read restored data: %v", err)
	}
	if !bytes.Equal(content, data) {
		t.Fatalf("expected restored data %q, got %q", data, content)
	}
	if retrieved.Metadata.Object.ID != "restart-meta" {
		t.Fatalf("expected restored metadata, got %q", retrieved.Metadata.Object.ID)
	}
	if got := secondCache.byteSize.Get(); got != int64(len(data)) {
		t.Fatalf("expected restored byte size %d, got %d", len(data), got)
	}
}

func TestFileCache_RemovesExpiredSidecarsOnStartup(t *testing.T) {
	ctx := t.Context()
	cfg := config.NewDefault()

	tmpDir := t.TempDir()
	key := FromString("expired-restart-key")
	data := []byte("expired body")

	firstCache := NewFileCache[TestMeta](cfg, tmpDir, 1024*1024*1024, time.Hour, 16, ctx)
	firstEntry, err := firstCache.Cache(key, bytes.NewReader(data), time.Now().Add(-time.Hour), TestMeta{ID: "expired"})
	if err != nil {
		t.Fatalf("cache expired entry failed: %v", err)
	}
	firstEntry.Data.Close()
	firstCache.Destroy()

	secondCache := NewFileCache[TestMeta](cfg, tmpDir, 1024*1024*1024, time.Hour, 16, ctx)
	defer secondCache.Destroy()

	if _, err := secondCache.Get(key); !errors.Is(err, ErrCacheEntryNotFound) {
		t.Fatalf("expected expired restored entry to be missing, got %v", err)
	}
	if _, err := os.Stat(secondCache.dataPath(key)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected expired data file to be removed, got %v", err)
	}
	if _, err := os.Stat(secondCache.metadataPath(key)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected expired metadata sidecar to be removed, got %v", err)
	}
}
