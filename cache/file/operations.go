package file

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reservoir/cache"
	"reservoir/metrics"
	"time"
)

func (c *Cache[MetadataT]) get(key cache.CacheKey, recordMetrics bool) (*cache.Entry[MetadataT], error) {
	lock := cache.GetLock(c.locks, key)
	lock.Lock() // Upgraded from RLock to Lock to prevent data race on LastAccess
	defer lock.Unlock()

	c.mu.RLock()
	entryMeta, exists := c.entriesMetadata[key]
	c.mu.RUnlock()

	if !exists {
		if recordMetrics {
			metrics.Global.Cache.CacheMisses.Increment()
		}
		slog.Debug("Cache miss", "key", key.Hex)
		return nil, cache.ErrCacheEntryNotFound
	}

	fileName := c.dataPath(key)
	dataFile, err := os.Open(fileName)
	if err != nil {
		if recordMetrics {
			metrics.Global.Cache.CacheErrors.Increment()
		}
		slog.Error("Failed to open cached data file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to open cached data file '%s'", ErrRead, fileName)
	}
	// We don't close dataFile here since we are returning it in the Entry.

	stale := false
	if entryMeta.Expires.Before(time.Now()) {
		stale = true // The entry is stale if the expiration time is in the past
	}

	entryMeta.LastAccess = time.Now()
	metaSnapshot := metadataSnapshot(entryMeta)

	if recordMetrics {
		metrics.Global.Cache.CacheHits.Increment()
	}
	slog.Debug("Successful cache hit", "key", key.Hex)
	return &cache.Entry[MetadataT]{
		Data:     dataFile,
		Metadata: metaSnapshot,
		Stale:    stale,
	}, nil
}

func (c *Cache[MetadataT]) Get(key cache.CacheKey) (*cache.Entry[MetadataT], error) {
	return c.get(key, true)
}

func (c *Cache[MetadataT]) GetQuiet(key cache.CacheKey) (*cache.Entry[MetadataT], error) {
	return c.get(key, false)
}

func (c *Cache[MetadataT]) Cache(key cache.CacheKey, data io.Reader, expires time.Time, metadata MetadataT) (*cache.Entry[MetadataT], error) {
	lock := cache.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	c.mu.RLock()
	oldMeta, replacing := c.entriesMetadata[key]
	oldSize := int64(0)
	if replacing {
		oldSize = oldMeta.Size
	}
	c.mu.RUnlock()

	fileName := c.dataPath(key)
	file, err := os.Create(fileName)
	if err != nil {
		metrics.Global.Cache.CacheErrors.Increment()
		slog.Error("Failed to create cache file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to create cache file '%s'", ErrCreate, fileName)
	}

	fileSize, err := io.Copy(file, data)
	if err != nil {
		file.Close()
		os.Remove(fileName)
		metrics.Global.Cache.CacheErrors.Increment()
		slog.Error("Failed to write cache file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to write cache file '%s'", ErrWrite, fileName)
	}

	if fileSize == 0 {
		file.Close()
		os.Remove(fileName)
		metrics.Global.Cache.CacheErrors.Increment()
		slog.Error("Cache file is empty", "key", key.Hex, "file_size", fileSize)
		return nil, fmt.Errorf("%w: wrote 0 bytes to cache file '%s'", ErrEmpty, fileName)
	}

	now := time.Now()
	meta := &cache.EntryMetadata[MetadataT]{
		TimeWritten: now,
		LastAccess:  now,
		Expires:     expires,
		Size:        fileSize,
		Object:      metadata,
	}

	maxCacheSize := c.maxCacheSize.Get()
	if c.byteSize.Get()-oldSize+fileSize >= maxCacheSize {
		c.janitor.Evict(maxCacheSize)
	}

	c.mu.Lock()
	previousMeta, replaced := c.entriesMetadata[key]
	c.entriesMetadata[key] = meta
	c.mu.Unlock()

	if replaced {
		cache.DecrementCacheSize(&c.byteSize, previousMeta.Size)
	} else {
		cache.IncrementCacheEntries()
	}
	cache.AddCacheSize(&c.byteSize, fileSize)
	c.writeMetadataSidecar(key, meta)

	slog.Debug("Successfully cached data", "key", key.Hex, "size", fileSize)

	slog.Debug("Seeking to the beginning of written cache file...", "key", key.Hex)
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		metrics.Global.Cache.CacheErrors.Increment()
		slog.Error("Failed to seek to start of cache file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to seek to start of cache file '%s'", ErrRead, fileName)
	}

	return &cache.Entry[MetadataT]{
		Data:     file,
		Metadata: metadataSnapshot(meta),
	}, nil
}

func (c *Cache[MetadataT]) Delete(key cache.CacheKey) error {
	lock := cache.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	return c.ensureRemove(key)
}

func (c *Cache[MetadataT]) removeDataFile(key cache.CacheKey) error {
	path := c.dataPath(key)
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		slog.Error("Failed to remove cache file", "path", path, "error", err)
		return fmt.Errorf("%w: failed to remove file '%s'", ErrRemove, path)
	}

	slog.Debug("Removed cache file", "path", path)
	return nil
}

// Ensures that both the cached file, its metadata and lock are removed without acquiring a lock.
func (c *Cache[MetadataT]) ensureRemove(key cache.CacheKey) error {
	c.mu.RLock()
	meta, exists := c.entriesMetadata[key]
	c.mu.RUnlock()

	if fileErr := c.removeDataFile(key); fileErr != nil {
		slog.Error("Failed to remove cached file", "key", key.Hex, "error", fileErr)
		return fmt.Errorf("%w: failed to remove cached file '%s'", fileErr, c.dataPath(key))
	}
	if metaErr := c.removeMetadataSidecar(key); metaErr != nil {
		slog.Error("Failed to remove cache metadata sidecar", "key", key.Hex, "error", metaErr)
		return fmt.Errorf("%w: failed to remove cache metadata sidecar for key '%s'", ErrRemove, key.Hex)
	}
	if !exists {
		return cache.ErrCacheEntryNotFound
	}

	c.mu.Lock()
	delete(c.entriesMetadata, key)
	c.mu.Unlock()

	cache.DecrementCacheEntries()
	cache.DecrementCacheSize(&c.byteSize, meta.Size)

	return nil
}

func (c *Cache[MetadataT]) Stats() cache.Stats {
	c.mu.RLock()
	entries := len(c.entriesMetadata)
	c.mu.RUnlock()

	return cache.Stats{
		Entries:  entries,
		Bytes:    c.byteSize.Get(),
		MaxBytes: c.maxCacheSize.Get(),
	}
}

func (c *Cache[MetadataT]) EvictTo(maxCacheBytes int64) {
	c.janitor.Evict(maxCacheBytes)
}

func (c *Cache[MetadataT]) Clear() error {
	c.mu.RLock()
	keys := make([]cache.CacheKey, 0, len(c.entriesMetadata))
	for key := range c.entriesMetadata {
		keys = append(keys, key)
	}
	c.mu.RUnlock()

	var errs []error
	for _, key := range keys {
		lock := cache.GetLock(c.locks, key)
		lock.Lock()
		err := c.ensureRemove(key)
		lock.Unlock()

		if err != nil && !errors.Is(err, cache.ErrCacheEntryNotFound) {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
