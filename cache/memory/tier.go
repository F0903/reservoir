package memory

import (
	"bytes"
	"io"
	"reservoir/cache"
	"reservoir/cache/internal/tier"
	"time"
)

func (c *Cache[MetadataT]) Stats() cache.Stats {
	c.mu.RLock()
	entries := len(c.entries)
	c.mu.RUnlock()

	return cache.Stats{
		Entries:        entries,
		Bytes:          c.byteSize.Get(),
		MaxBytes:       c.maxCacheSize.Get(),
		MemoryCapBytes: c.memoryCap.Get(),
	}
}

func (c *Cache[MetadataT]) CapacityBytes() int64 {
	return min(c.maxCacheSize.Get(), c.memoryCap.Get())
}

func (c *Cache[MetadataT]) StoredBytes() int64 {
	return c.byteSize.Get()
}

func (c *Cache[MetadataT]) EntrySize(key cache.CacheKey) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return 0
	}
	return entry.meta.Size
}

func (c *Cache[MetadataT]) CanStore(key cache.CacheKey, size int64) bool {
	limit := c.CapacityBytes()
	if limit <= 0 || size > limit {
		return false
	}

	oldSize := c.EntrySize(key)
	return c.byteSize.Get()-oldSize+size <= limit
}

func (c *Cache[MetadataT]) EvictTo(maxCacheBytes int64) {
	c.janitor.Evict(maxCacheBytes)
}

func (c *Cache[MetadataT]) OverrideMemoryCapForTesting(memoryCap int64) {
	c.memoryCap.Set(memoryCap)
}

func (c *Cache[MetadataT]) OverrideEntryLastAccessForTesting(key cache.CacheKey, lastAccess time.Time) bool {
	lock := tier.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return false
	}

	entry.setLastAccess(lastAccess)
	return true
}

func (c *Cache[MetadataT]) DeleteAfter(key cache.CacheKey, fn func() error) error {
	lock := tier.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	if err := fn(); err != nil {
		return err
	}
	return c.deleteInternal(key)
}

func (c *Cache[MetadataT]) DemotionCandidates(cutoff time.Time) []cache.CacheKey {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]cache.CacheKey, 0)
	for key, entry := range c.entries {
		if entry.lastAccess.Get().Before(cutoff) {
			keys = append(keys, key)
		}
	}
	return keys
}

func (c *Cache[MetadataT]) DemoteEntry(key cache.CacheKey, cutoff time.Time, write func(data io.Reader, expires time.Time, metadata MetadataT) error) error {
	lock := tier.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil
	}
	lastAccess := entry.lastAccess.Get()
	if lastAccess.After(cutoff) || lastAccess.Equal(cutoff) {
		return nil
	}
	if entry.meta.Expires.Before(time.Now()) {
		return c.deleteInternal(key)
	}

	if err := write(bytes.NewReader(entry.data), entry.meta.Expires, entry.meta.Object); err != nil {
		return err
	}
	return c.deleteInternal(key)
}
