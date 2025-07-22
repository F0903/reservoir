package cache

import (
	"apt_cacher_go/config"
	"apt_cacher_go/metrics"
	"apt_cacher_go/utils/assertedpath"
	"apt_cacher_go/utils/bytesize"
	"apt_cacher_go/utils/syncmap"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

type FileCache[ObjectData any] struct {
	rootDir  assertedpath.AssertedPath
	locks    *syncmap.SyncMap[string, *sync.RWMutex]
	entries  map[CacheKey]*EntryMetadata[ObjectData]
	byteSize int64
}

// NewFileCache creates a new FileCache instance with the specified root directory.
func NewFileCache[ObjectData any](rootDir string, cleanupInterval time.Duration, ctx context.Context) *FileCache[ObjectData] {
	c := &FileCache[ObjectData]{
		rootDir:  assertedpath.Assert(rootDir).EnsureCleared(),
		locks:    syncmap.New[string, *sync.RWMutex](),
		entries:  make(map[CacheKey]*EntryMetadata[ObjectData]),
		byteSize: 0,
	}
	c.startCleanupTask(cleanupInterval, ctx)
	return c
}

// Gets or creates a lock for the given key.
func (c *FileCache[ObjectData]) getLock(key CacheKey) *sync.RWMutex {
	return c.locks.GetOrSet(key.Hex, &sync.RWMutex{})
}

func (c *FileCache[ObjectData]) ensureCacheSize() {
	if c.byteSize < config.Global.MaxCacheSize.Bytes() {
		return
	}

	slog.Info("Cache size exceeds limit, starting eviction", "byte_size", c.byteSize, "max_cache_size", config.Global.MaxCacheSize.Bytes())

	type entryForEviction struct {
		key      CacheKey
		meta     *EntryMetadata[ObjectData]
		priority int64 // Lower = evict first
	}

	candidates := make([]entryForEviction, 0, len(c.entries))
	now := time.Now()
	startCacheSize := c.byteSize

	for key, meta := range c.entries {
		timeSinceAccess := now.Sub(meta.LastAccess).Milliseconds()
		sizeWeight := meta.FileSize.Convert(bytesize.UnitM)

		// Calculate eviction priority (highest = evict first)
		// Factors: age since last access + file size weight
		priority := timeSinceAccess + (sizeWeight * 100) // Give size significant weight

		candidates = append(candidates, entryForEviction{
			key:      key,
			meta:     meta,
			priority: priority,
		})
	}

	// Sort by priority (highest = evict first)
	slices.SortFunc(candidates, func(x, y entryForEviction) int {
		return cmp.Compare(y.priority, x.priority) // Swapped x and y for descending order
	})

	// Evict entries until we're under the limit
	targetSize := int64(float64(config.Global.MaxCacheSize.Bytes()) * 0.8) // Evict to 80% to avoid thrashing

	slog.Info("Target size for eviction", "target_size", bytesize.ByteSize(targetSize))
	evictions := 0
	for _, candidate := range candidates {
		if c.byteSize <= targetSize {
			break
		}

		lock := c.getLock(candidate.key)
		if lock.TryLock() {
			slog.Info("Evicting cache entry", "key", candidate.key.Hex, "size", candidate.meta.FileSize, "last_access", candidate.meta.LastAccess)

			if err := c.ensureRemove(candidate.key); err != nil {
				slog.Info("Failed to evict cache entry", "key", candidate.key.Hex, "error", err)
			}
			evictions++
			lock.Unlock()
		} else {
			slog.Info("Failed to acquire lock for cache entry", "key", candidate.key.Hex)
			continue
		}
	}

	endCacheSize := c.byteSize
	metrics.Global.Cache.BytesCached.Set(endCacheSize)
	metrics.Global.Cache.BytesCleaned.Add(startCacheSize - endCacheSize)

	slog.Info("Cache eviction complete", "evicted_entries", evictions, "new_size", endCacheSize)
}

func (c *FileCache[ObjectData]) cleanExpiredEntries() {
	slog.Info("Cleaning up expired cache entries")

	startCacheSize := c.byteSize

	keysToRemove := make([]CacheKey, 0)

	for key, meta := range c.entries {
		expired := meta.Expires.Before(time.Now())

		if !expired {
			continue
		}

		slog.Info("Found expired cache entry for key", "key", key.Hex)
		keysToRemove = append(keysToRemove, key)
	}

	for _, key := range keysToRemove {
		slog.Info("Removing expired cache entry for key", "key", key.Hex)

		lock := c.getLock(key)
		locked := lock.TryLock()
		if !locked {
			slog.Info("Failed to acquire lock for key", "key", key.Hex)
			continue
		}

		if err := c.ensureRemove(key); err != nil {
			slog.Info("Failed to remove expired cache entry for key", "key", key.Hex, "error", err)
			lock.Unlock()
			continue
		}
		lock.Unlock()

		slog.Info("Removed expired cache entry for key", "key", key.Hex)
	}

	endCacheSize := c.byteSize
	metrics.Global.Cache.BytesCached.Set(endCacheSize)
	metrics.Global.Cache.BytesCleaned.Add(startCacheSize - endCacheSize)

	slog.Info("Cache cleanup complete", "new_size", endCacheSize)
}

func (c *FileCache[ObjectData]) startCleanupTask(interval time.Duration, ctx context.Context) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		slog.Info("Cache cleanup task started")
		for {
			select {
			case <-ticker.C:
				slog.Info("Running cache cleanup cycle...")
				c.cleanExpiredEntries()
				c.ensureCacheSize()
				metrics.Global.Cache.CleanupRuns.Increment()
				slog.Info("Cache cleanup cycle complete")
			case <-ctx.Done():
				slog.Info("Cache cleanup task stopped")
				return
			}
		}
	}()
}

func (c *FileCache[ObjectData]) addCacheSize(delta int64) {
	c.byteSize += delta
	metrics.Global.Cache.BytesCached.Add(delta)
}

func (c *FileCache[ObjectData]) ensureRemoveFile(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// We only want to return critical errors, not if the file doesn't exist
			return nil
		}
		return fmt.Errorf("failed to stat file '%s': %v", path, err)
	}

	if err := os.Remove(path); err != nil {
		// We checked earlier that the file exists, so if we get an error here, it is unexpected.
		return fmt.Errorf("failed to remove file '%s': %v", path, err)
	}

	slog.Info("Removed file", "path", path)
	size := stat.Size()

	metrics.Global.Cache.CacheEntries.Add(-1)
	c.addCacheSize(-size)

	return nil
}

// Ensures that both the cached file, its metadata and lock are removed without acquiring a lock.
func (c *FileCache[ObjectData]) ensureRemove(key CacheKey) error {
	filePath := filepath.Join(c.rootDir.Path, key.Hex)

	if fileErr := c.ensureRemoveFile(filePath); fileErr != nil {
		return fmt.Errorf("failed to remove cached file '%s': %v", filePath, fileErr)
	}

	delete(c.entries, key)  // Remove the entry from the map
	c.locks.Delete(key.Hex) // Remove the lock for this key

	return nil
}

func (c *FileCache[ObjectData]) Get(key CacheKey) (*Entry[ObjectData], error) {
	lock := c.getLock(key)
	lock.RLock()
	defer lock.RUnlock()

	fileName := filepath.Join(c.rootDir.Path, key.Hex)
	entryMeta, exists := c.entries[key]
	if !exists {
		metrics.Global.Cache.CacheMisses.Increment()
		slog.Info("Cache miss", "key", key.Hex)
		return nil, ErrorCacheMiss
	}

	dataFile, err := os.Open(fileName)
	if err != nil {
		metrics.Global.Cache.CacheErrors.Increment()
		return nil, fmt.Errorf("failed to read cached data file '%s': %v", fileName, err)
	}
	// We don't close dataFile here since we are returning it in the Entry.

	stale := false
	if entryMeta.Expires.Before(time.Now()) {
		stale = true // The entry is stale if the expiration time is in the past
	}

	entryMeta.LastAccess = time.Now()

	metrics.Global.Cache.CacheHits.Increment()
	slog.Info("Successful cache hit", "key", key.Hex)
	return &Entry[ObjectData]{
		Data:     dataFile,
		Metadata: entryMeta,
		Stale:    stale,
	}, nil
}

func (c *FileCache[ObjectData]) Cache(key CacheKey, data io.Reader, expires time.Time, objectData ObjectData) (*Entry[ObjectData], error) {
	lock := c.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	fileName := filepath.Join(c.rootDir.Path, key.Hex)
	file, err := os.Create(fileName)
	if err != nil {
		metrics.Global.Cache.CacheErrors.Increment()
		return nil, fmt.Errorf("failed to create cache file '%s': %v", fileName, err)
	}
	// We don't close file here since we are returning it in the Entry.

	fileSize, err := io.Copy(file, data)
	if err != nil {
		metrics.Global.Cache.CacheErrors.Increment()
		return nil, fmt.Errorf("failed to write cache file '%s': %v", fileName, err)
	}

	if fileSize == 0 {
		metrics.Global.Cache.CacheErrors.Increment()
		return nil, fmt.Errorf("wrote 0 bytes to cache file '%s', treating as error", fileName)
	}

	meta := &EntryMetadata[ObjectData]{
		TimeWritten: time.Now(),
		LastAccess:  time.Now(),
		Expires:     expires,
		FileSize:    bytesize.ByteSize(fileSize),
		Object:      objectData,
	}
	c.entries[key] = meta

	metrics.Global.Cache.CacheEntries.Add(1)
	c.addCacheSize(fileSize)

	slog.Info("Successfully cached data", "key", key.Hex, "size", fileSize)

	// Check if cache size exceeded limit and evict if necessary
	c.ensureCacheSize()

	// Reset file stream to the beginning
	file.Seek(0, io.SeekStart)

	return &Entry[ObjectData]{
		Data:     file,
		Metadata: meta,
	}, nil
}

func (c *FileCache[ObjectData]) Delete(key CacheKey) error {
	lock := c.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	return c.ensureRemove(key)
}

func (c *FileCache[ObjectData]) UpdateMetadata(key CacheKey, modifier func(*EntryMetadata[ObjectData])) error {
	lock := c.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	meta, exists := c.entries[key]
	if !exists {
		metrics.Global.Cache.CacheErrors.Increment()
		return fmt.Errorf("cache entry for key '%s' does not exist", key.Hex)
	}

	modifier(meta)
	meta.LastAccess = time.Now()

	metrics.Global.Cache.CacheHits.Increment()

	slog.Info("Successfully updated metadata", "key", key.Hex)
	return nil
}
