package cache

import (
	"bytes"
	"reservoir/config"
	"reservoir/utils/bytesize"
	"testing"
	"time"
)

func TestCache_JanitorCleanup(t *testing.T) {
	ctx := t.Context()

	// Use very short cleanup interval for testing
	cleanupInterval := 100 * time.Millisecond
	c := NewMemoryCache[TestMeta](1, config.Global.MaxCacheSize.Read().Bytes(), cleanupInterval, 16, ctx)
	defer c.Destroy()

	key := FromString("expired-key")
	data := []byte("expired data")
	// Expired 1 second ago
	expires := time.Now().Add(-time.Second)

	_, err := c.Cache(key, bytes.NewReader(data), expires, TestMeta{})
	if err != nil {
		t.Fatalf("Cache failed: %v", err)
	}

	// Verify it's there but stale
	entry, err := c.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	entry.Data.Close()
	if !entry.Stale {
		t.Error("Expected entry to be stale")
	}

	// Wait for janitor to run
	time.Sleep(300 * time.Millisecond)

	// Verify it's gone
	_, err = c.Get(key)
	if err != ErrCacheEntryNotFound {
		t.Errorf("Expected ErrCacheEntryNotFound after janitor cleanup, got %v", err)
	}
}

func TestCache_JanitorEviction(t *testing.T) {
	ctx := t.Context()

	// Mock global config for max cache size
	oldMaxCacheSize := config.Global.MaxCacheSize.Read()
	defer config.Global.MaxCacheSize.Overwrite(oldMaxCacheSize)

	// Set max cache size to 1KB
	config.Global.MaxCacheSize.Overwrite(bytesize.ParseUnchecked("1K"))

	cleanupInterval := 100 * time.Millisecond
	c := NewMemoryCache[TestMeta](1, config.Global.MaxCacheSize.Read().Bytes(), cleanupInterval, 16, ctx)
	defer c.Destroy()

	// Add 2 entries of 600 bytes each (Total 1200 > 1024)
	data := make([]byte, 600)

	entry1, err := c.Cache(FromString("key-1"), bytes.NewReader(data), time.Now().Add(time.Hour), TestMeta{})
	if err != nil {
		t.Fatalf("Cache 1 failed: %v", err)
	}
	entry1.Data.Close()

	// Small sleep to ensure different LastAccess times
	time.Sleep(50 * time.Millisecond)

	entry2, err := c.Cache(FromString("key-2"), bytes.NewReader(data), time.Now().Add(time.Hour), TestMeta{})
	if err != nil {
		t.Fatalf("Cache 2 failed: %v", err)
	}
	entry2.Data.Close()

	// Wait for janitor to run
	time.Sleep(300 * time.Millisecond)

	// One of them should have been evicted to get under 80% (819 bytes)
	r1, err1 := c.Get(FromString("key-1"))
	if err1 == nil {
		r1.Data.Close()
	}
	r2, err2 := c.Get(FromString("key-2"))
	if err2 == nil {
		r2.Data.Close()
	}

	if err1 != nil && err2 != nil {
		t.Error("Both entries were evicted, expected one to remain")
	}
	if err1 == nil && err2 == nil {
		t.Error("Neither entry was evicted, expected one to be removed")
	}
}
