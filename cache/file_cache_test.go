package cache

import (
	"bytes"
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
