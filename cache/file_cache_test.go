package cache

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func TestFileCache_Basic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "reservoir-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	c := NewFileCache[TestMeta](tmpDir, 1024*1024*1024, time.Minute, 16, ctx)

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
