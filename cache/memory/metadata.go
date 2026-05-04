package memory

import (
	"reservoir/cache"
	"reservoir/cache/internal/tier"
	"time"
)

func (c *Cache[MetadataT]) updateMetadata(key cache.CacheKey, modifier func(*cache.EntryMetadata[MetadataT]), recordMetrics bool) error {
	lock := tier.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		if recordMetrics {
			c.recordCacheError()
		}
		return cache.ErrCacheEntryNotFound
	}

	modifier(entry.meta)
	entry.setLastAccess(time.Now())

	if recordMetrics {
		c.recordCacheHit()
	}
	return nil
}

func (c *Cache[MetadataT]) UpdateMetadata(key cache.CacheKey, modifier func(*cache.EntryMetadata[MetadataT])) error {
	return c.updateMetadata(key, modifier, true)
}

func (c *Cache[MetadataT]) UpdateMetadataQuiet(key cache.CacheKey, modifier func(*cache.EntryMetadata[MetadataT])) error {
	return c.updateMetadata(key, modifier, false)
}

func (c *Cache[MetadataT]) getMetadata(key cache.CacheKey, recordMetrics bool) (meta *cache.EntryMetadata[MetadataT], stale bool, err error) {
	lock := tier.GetLock(c.locks, key)
	lock.RLock()
	defer lock.RUnlock()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		if recordMetrics {
			c.recordCacheMiss()
		}
		return nil, false, cache.ErrCacheEntryNotFound
	}

	stale = false
	if entry.meta.Expires.Before(time.Now()) {
		stale = true
	}

	entry.touch(time.Now())
	if recordMetrics {
		c.recordCacheHit()
	}

	return entry.metadataSnapshot(), stale, nil
}

func (c *Cache[MetadataT]) GetMetadata(key cache.CacheKey) (meta *cache.EntryMetadata[MetadataT], stale bool, err error) {
	return c.getMetadata(key, true)
}

func (c *Cache[MetadataT]) GetMetadataQuiet(key cache.CacheKey) (meta *cache.EntryMetadata[MetadataT], stale bool, err error) {
	return c.getMetadata(key, false)
}
