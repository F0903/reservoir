package file

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reservoir/cache"
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
			c.metrics.Cache.CacheMisses.Increment()
		}
		slog.Debug("Cache miss", "key", key.Hex)
		return nil, cache.ErrCacheEntryNotFound
	}

	fileName := c.dataPath(key)
	dataFile, err := os.Open(fileName)
	if err != nil {
		if recordMetrics {
			c.metrics.Cache.CacheErrors.Increment()
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
		c.metrics.Cache.CacheHits.Increment()
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

	maxCacheSize := c.maxCacheSize.Get()
	if maxCacheSize <= 0 {
		c.metrics.Cache.CacheErrors.Increment()
		return nil, cache.ErrCacheMemoryExceeded
	}
	if sizeHint, ok := cache.ReaderSizeHint(data); ok && sizeHint > maxCacheSize {
		c.metrics.Cache.CacheErrors.Increment()
		slog.Debug("Skipping file cache write because response exceeds cache max size", "key", key.Hex, "size_hint", sizeHint, "max_cache_size", maxCacheSize)
		return nil, fmt.Errorf("%w: response size %d exceeds cache max size %d", cache.ErrCacheMemoryExceeded, sizeHint, maxCacheSize)
	}

	c.mu.RLock()
	oldMeta, replacing := c.entriesMetadata[key]
	oldSize := int64(0)
	if replacing {
		oldSize = oldMeta.Size
	}
	c.mu.RUnlock()

	fileName := c.dataPath(key)
	tmpFile, err := os.CreateTemp(c.rootDir.Path, key.Hex+".*.tmp")
	if err != nil {
		c.metrics.Cache.CacheErrors.Increment()
		slog.Error("Failed to create temporary cache file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to create temporary cache file for '%s'", ErrCreate, fileName)
	}
	tmpFileName := tmpFile.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpFileName)
		}
	}()

	fileSize, err := copyBoundedCacheFile(tmpFile, data, maxCacheSize)
	closeErr := tmpFile.Close()
	if err != nil {
		c.metrics.Cache.CacheErrors.Increment()
		if errors.Is(err, cache.ErrCacheMemoryExceeded) {
			slog.Debug("Skipping file cache write because response exceeds cache max size", "key", key.Hex, "file_size", fileSize, "max_cache_size", maxCacheSize)
			return nil, fmt.Errorf("%w: response exceeds cache max size %d", cache.ErrCacheMemoryExceeded, maxCacheSize)
		}
		slog.Error("Failed to write cache file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to write cache file '%s'", ErrWrite, fileName)
	}
	if closeErr != nil {
		c.metrics.Cache.CacheErrors.Increment()
		slog.Error("Failed to close cache file after write", "key", key.Hex, "error", closeErr)
		return nil, fmt.Errorf("%w: failed to close cache file '%s'", ErrWrite, fileName)
	}

	if fileSize == 0 {
		c.metrics.Cache.CacheErrors.Increment()
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

	if c.byteSize.Get()-oldSize+fileSize > maxCacheSize {
		c.janitor.Evict(maxCacheSize)
	}
	if c.byteSize.Get()-oldSize+fileSize > maxCacheSize {
		c.metrics.Cache.CacheErrors.Increment()
		slog.Debug("Skipping file cache write because eviction did not free enough space", "key", key.Hex, "file_size", fileSize, "max_cache_size", maxCacheSize)
		return nil, fmt.Errorf("%w: insufficient file cache capacity for '%s'", cache.ErrCacheMemoryExceeded, key.Hex)
	}

	if err := replaceCacheDataFile(tmpFileName, fileName); err != nil {
		c.metrics.Cache.CacheErrors.Increment()
		slog.Error("Failed to replace cache file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to replace cache file '%s'", ErrWrite, fileName)
	}
	committed = true

	c.mu.Lock()
	previousMeta, replaced := c.entriesMetadata[key]
	c.entriesMetadata[key] = meta
	c.mu.Unlock()

	if replaced {
		cache.DecrementCacheSize(c.metrics, &c.byteSize, previousMeta.Size)
	} else {
		cache.IncrementCacheEntries(c.metrics)
	}
	cache.AddCacheSize(c.metrics, &c.byteSize, fileSize)
	c.writeMetadataSidecar(key, meta)

	slog.Debug("Successfully cached data", "key", key.Hex, "size", fileSize)

	file, err := os.Open(fileName)
	if err != nil {
		c.metrics.Cache.CacheErrors.Increment()
		slog.Error("Failed to open written cache file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to open written cache file '%s'", ErrRead, fileName)
	}

	return &cache.Entry[MetadataT]{
		Data:     file,
		Metadata: metadataSnapshot(meta),
	}, nil
}

func copyBoundedCacheFile(file *os.File, data io.Reader, maxCacheSize int64) (int64, error) {
	limited := &io.LimitedReader{R: data, N: fileCacheProbeLimit(maxCacheSize)}
	fileSize, err := io.Copy(file, limited)
	if err != nil {
		return fileSize, err
	}
	if fileSize > maxCacheSize {
		return fileSize, cache.ErrCacheMemoryExceeded
	}
	return fileSize, nil
}

func fileCacheProbeLimit(maxCacheSize int64) int64 {
	const maxInt64 = int64(^uint64(0) >> 1)
	if maxCacheSize == maxInt64 {
		return maxCacheSize
	}
	return maxCacheSize + 1
}

func replaceCacheDataFile(source string, destination string) error {
	if err := os.Rename(source, destination); err == nil {
		return nil
	} else if _, statErr := os.Stat(destination); statErr != nil {
		return err
	}

	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(source, destination)
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

	cache.DecrementCacheEntries(c.metrics)
	cache.DecrementCacheSize(c.metrics, &c.byteSize, meta.Size)

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
