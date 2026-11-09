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
		cache.DecrementCacheSize(c.metrics, &c.byteSize, previousEntry.meta.Size)
	} else {
		cache.IncrementCacheEntries(c.metrics)
	}
	cache.AddCacheSize(c.metrics, &c.byteSize, int64(count))

	return &cache.Entry[MetadataT]{
		Data:     &memoryReadSeekCloser{bytes.NewReader(dataBytes)},
		Metadata: internalEntry.metadataSnapshot(),
	}, nil
}

// reserve claims budget for a download about to be read into memory, evicting if needed.
// The caller must release it with reservedBytes.Sub once the download is stored or abandoned.
func (c *Cache[MetadataT]) reserve(size int64) bool {
	limit := c.CapacityBytes()
	if c.byteSize.Get()+c.reservedBytes.Get()+size > limit {
		c.janitor.Evict(limit)
	}

	c.reservedBytes.Add(size)
	if c.byteSize.Get()+c.reservedBytes.Get() > limit {
		c.reservedBytes.Sub(size)
		return false
	}
	return true
}

func (c *Cache[MetadataT]) cache(key cache.CacheKey, data io.Reader, expires time.Time, metadata MetadataT, recordErrors bool) (*cache.Entry[MetadataT], error) {
	if size, ok := cache.ReaderSizeHint(data); ok {
		if !c.reserve(size) {
			if recordErrors {
				c.recordCacheError()
			}
			return nil, cache.ErrCacheMemoryExceeded
		}
		defer c.reservedBytes.Sub(size)
	}

	// Read before locking so a slow download can't hold up other keys on this shard.
	dataBytes, err := cache.ReadAllCacheBytes(data, c.CapacityBytes())
	if err != nil {
		if recordErrors {
			c.recordCacheError()
		}
		return nil, err
	}

	lock := cache.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	return c.cacheBytesInternal(key, dataBytes, expires, metadata, true, false, recordErrors)
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

	cache.DecrementCacheEntries(c.metrics)
	cache.DecrementCacheSize(c.metrics, &c.byteSize, entry.meta.Size)

	return nil
}
