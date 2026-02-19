package cache

import (
	"bytes"
	"io"
	"reservoir/config"
	"testing"
	"time"
)

func TestMemoryCache_Basic(t *testing.T) {
	ctx := t.Context()
	cfg := config.NewDefault()

	// 1% memory budget, large maxCacheSize, 1 min cleanup, 16 shards
	c := NewMemoryCache[TestMeta](cfg, 1, 1024*1024*1024, time.Minute, 16, ctx)
	defer c.Destroy()

	key := FromString("test-key")
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
	if err != ErrCacheEntryNotFound {
		t.Errorf("Expected ErrCacheEntryNotFound, got %v", err)
	}
}
