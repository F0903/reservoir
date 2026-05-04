package memory

import (
	"bytes"
	"errors"
	"io"
	"reservoir/cache"
	"time"
)

func (c *Cache[MetadataT]) get(key cache.CacheKey, recordMetrics bool) (*cache.Entry[MetadataT], error) {
	lock := cache.GetLock(c.locks, key)
	lock.RLock()
	defer lock.RUnlock()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		if recordMetrics {
			c.recordCacheMiss()
		}
		return nil, cache.ErrCacheEntryNotFound
	}

	stale := false
	if entry.meta.Expires.Before(time.Now()) {
		stale = true
	}

	entry.touch(time.Now())
	if recordMetrics {
		c.recordCacheHit()
	}

	return &cache.Entry[MetadataT]{
		Data:     &memoryReadSeekCloser{bytes.NewReader(entry.data)},
		Metadata: entry.metadataSnapshot(),
		Stale:    stale,
	}, nil
}

func (c *Cache[MetadataT]) Get(key cache.CacheKey) (*cache.Entry[MetadataT], error) {
	return c.get(key, true)
}

func (c *Cache[MetadataT]) GetQuiet(key cache.CacheKey) (*cache.Entry[MetadataT], error) {
	return c.get(key, false)
}

func (c *Cache[MetadataT]) cacheBytesInternal(key cache.CacheKey, dataBytes []byte, expires time.Time, metadata MetadataT, evictIfFull bool, strictLimit bool, recordErrors bool) (*cache.Entry[MetadataT], error) {
	maxCacheSize := c.maxCacheSize.Get()
	limit := min(maxCacheSize, c.memoryCap.Get())
	count := int64(len(dataBytes))

	c.mu.RLock()
	oldEntry, replacing := c.entries[key]
	oldSize := int64(0)
	if replacing {
		oldSize = oldEntry.meta.Size
	}
	c.mu.RUnlock()

	if c.byteSize.Get()-oldSize >= limit {
		if evictIfFull {
			c.janitor.Evict(limit)
		}

		if c.byteSize.Get()-oldSize >= limit {
			if recordErrors {
				c.recordCacheError()
			}
			return nil, cache.ErrCacheMemoryExceeded
		}
	}

	if c.byteSize.Get()-oldSize+count > limit {
		if evictIfFull {
			c.janitor.Evict(limit)
		}
		if strictLimit && c.byteSize.Get()-oldSize+count > limit {
			if recordErrors {
				c.recordCacheError()
			}
			return nil, cache.ErrCacheMemoryExceeded
		}
	}

	now := time.Now()
	meta := &cache.EntryMetadata[MetadataT]{
		TimeWritten: now,
		LastAccess:  now,
		Expires:     expires,
		Size:        count,
		Object:      metadata,
	}
	internalEntry := newMemoryInternalEntry(dataBytes, meta)

	c.mu.Lock()
	previousEntry, replaced := c.entries[key]
	c.entries[key] = internalEntry
	c.mu.Unlock()

	if replaced {
		cache.DecrementCacheSize(&c.byteSize, previousEntry.meta.Size)
	} else {
		cache.IncrementCacheEntries()
	}
	cache.AddCacheSize(&c.byteSize, int64(count))

	return &cache.Entry[MetadataT]{
		Data:     &memoryReadSeekCloser{bytes.NewReader(dataBytes)},
		Metadata: internalEntry.metadataSnapshot(),
	}, nil
}

func (c *Cache[MetadataT]) cacheInternal(key cache.CacheKey, data io.Reader, expires time.Time, metadata MetadataT, evictIfFull bool, strictLimit bool, recordErrors bool) (*cache.Entry[MetadataT], error) {
	limit := c.CapacityBytes()
	dataBytes, err := cache.ReadAllCacheBytes(data, limit)
	if err != nil {
		if recordErrors {
			c.recordCacheError()
		}
		return nil, err
	}
	return c.cacheBytesInternal(key, dataBytes, expires, metadata, evictIfFull, strictLimit, recordErrors)
}

func (c *Cache[MetadataT]) cache(key cache.CacheKey, data io.Reader, expires time.Time, metadata MetadataT, recordErrors bool) (*cache.Entry[MetadataT], error) {
	lock := cache.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	return c.cacheInternal(key, data, expires, metadata, true, false, recordErrors)
}

func (c *Cache[MetadataT]) cacheStrict(key cache.CacheKey, data io.Reader, expires time.Time, metadata MetadataT, recordErrors bool) (*cache.Entry[MetadataT], error) {
	lock := cache.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	return c.cacheInternal(key, data, expires, metadata, true, true, recordErrors)
}

func (c *Cache[MetadataT]) cacheBytesStrict(key cache.CacheKey, data []byte, expires time.Time, metadata MetadataT, recordErrors bool) (*cache.Entry[MetadataT], error) {
	lock := cache.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	return c.cacheBytesInternal(key, data, expires, metadata, true, true, recordErrors)
}

func (c *Cache[MetadataT]) CacheBytesStrictQuiet(key cache.CacheKey, data []byte, expires time.Time, metadata MetadataT) (*cache.Entry[MetadataT], error) {
	return c.cacheBytesStrict(key, data, expires, metadata, false)
}

func (c *Cache[MetadataT]) Cache(key cache.CacheKey, data io.Reader, expires time.Time, metadata MetadataT) (*cache.Entry[MetadataT], error) {
	return c.cache(key, data, expires, metadata, true)
}

func (c *Cache[MetadataT]) Delete(key cache.CacheKey) error {
	lock := cache.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	return c.deleteInternal(key)
}

func (c *Cache[MetadataT]) Clear() error {
	c.mu.RLock()
	keys := make([]cache.CacheKey, 0, len(c.entries))
	for key := range c.entries {
		keys = append(keys, key)
	}
	c.mu.RUnlock()

	var errs []error
	for _, key := range keys {
		lock := cache.GetLock(c.locks, key)
		lock.Lock()
		err := c.deleteInternal(key)
		lock.Unlock()

		if err != nil && !errors.Is(err, cache.ErrCacheEntryNotFound) {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (c *Cache[MetadataT]) deleteInternal(key cache.CacheKey) error {
	c.mu.Lock()
	entry, ok := c.entries[key]
	if !ok {
		c.mu.Unlock()
		return cache.ErrCacheEntryNotFound
	}
	delete(c.entries, key)
	c.mu.Unlock()

	cache.DecrementCacheEntries()
	cache.DecrementCacheSize(&c.byteSize, entry.meta.Size)

	return nil
}
