package cache

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/utils/assertedpath"
	"reservoir/utils/bytesize"
	"reservoir/utils/syncmap"
	"slices"
	"sync"
	"time"
)

var (
	ErrCacheFileCreate = errors.New("cache file create failed")
	ErrCacheFileWrite  = errors.New("cache file write failed")
	ErrCacheFileRead   = errors.New("cache file read failed")
	ErrCacheFileRemove = errors.New("cache file remove failed")
	ErrCacheFileEmpty  = errors.New("cache file empty")
	ErrCacheFileStat   = errors.New("cache file stat failed")
)

type FileCache[MetadataT any] struct {
	rootDir  assertedpath.AssertedPath
	locks    *syncmap.SyncMap[CacheKey, *sync.RWMutex]
	entries  map[CacheKey]*EntryMetadata[MetadataT]
	byteSize int64
	stopChan chan struct{}
}

// NewFileCache creates a new FileCache instance with the specified root directory.
func NewFileCache[MetadataT any](rootDir string, cleanupInterval time.Duration, ctx context.Context) *FileCache[MetadataT] {
	c := &FileCache[MetadataT]{
		rootDir:  assertedpath.AssertDirectory(rootDir).EnsureCleared(),
		locks:    syncmap.New[CacheKey, *sync.RWMutex](),
		entries:  make(map[CacheKey]*EntryMetadata[MetadataT]),
		byteSize: 0,
		stopChan: make(chan struct{}),
	}
	//TODO: replace with the new janitor
	c.startCleanupTask(cleanupInterval, ctx)
	return c
}

func (c *FileCache[MetadataT]) Destroy() {
	close(c.stopChan)
}

// Gets or creates a lock for the given key.
func (c *FileCache[MetadataT]) getLock(key CacheKey) *sync.RWMutex {
	return c.locks.GetOrSet(key, &sync.RWMutex{})
}

func (c *FileCache[MetadataT]) ensureCacheSize() {
	maxCacheSize := config.Global.MaxCacheSize.Read().Bytes()
	if c.byteSize < maxCacheSize {
		return
	}

	slog.Info("Cache size exceeds limit, starting eviction", "byte_size", c.byteSize, "max_cache_size", maxCacheSize)

	type entryForEviction struct {
		key      CacheKey
		meta     *EntryMetadata[MetadataT]
		priority int64 // Lower = evict first
	}

	candidates := make([]entryForEviction, 0, len(c.entries))
	now := time.Now()
	startCacheSize := c.byteSize

	for key, meta := range c.entries {
		timeSinceAccess := now.Sub(meta.LastAccess).Milliseconds()
		sizeWeight := meta.Size / bytesize.UnitM

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
	targetSize := int64(float64(maxCacheSize) * 0.8) // Evict to 80% to avoid thrashing

	slog.Info("Target size for eviction", "target_size", bytesize.ByteSize(targetSize))
	evictions := 0
	for _, candidate := range candidates {
		if c.byteSize <= targetSize {
			break
		}

		lock := c.getLock(candidate.key)
		if lock.TryLock() {
			slog.Info("Evicting cache entry", "key", candidate.key.Hex, "size", candidate.meta.Size, "last_access", candidate.meta.LastAccess)

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

func (c *FileCache[MetadataT]) cleanExpiredEntries() {
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
			lock.Unlock()
			slog.Info("Failed to remove expired cache entry for key", "key", key.Hex, "error", err)
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

func (c *FileCache[MetadataT]) startCleanupTask(interval time.Duration, ctx context.Context) {
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
			case <-c.stopChan:
				slog.Info("Cache cleanup task stopped via stopChan")
				return
			case <-ctx.Done():
				slog.Info("Cache cleanup task stopped via context")
				return
			}
		}
	}()
}

func (c *FileCache[MetadataT]) addCacheSize(delta int64) {
	c.byteSize += delta
	metrics.Global.Cache.BytesCached.Add(delta)
}

func (c *FileCache[MetadataT]) ensureRemoveFile(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// We only want to return critical errors, not if the file doesn't exist
			return nil
		}
		slog.Error("Failed to stat cache file", "path", path, "error", err)
		return fmt.Errorf("%w: failed to stat file '%s'", ErrCacheFileStat, path)
	}

	if err := os.Remove(path); err != nil {
		// We checked earlier that the file exists, so if we get an error here, it is unexpected.
		slog.Error("Failed to remove cache file", "path", path, "error", err)
		return fmt.Errorf("%w: failed to remove file '%s'", ErrCacheFileRemove, path)
	}

	slog.Info("Removed file", "path", path)
	size := stat.Size()

	metrics.Global.Cache.CacheEntries.Add(-1)
	c.addCacheSize(-size)

	return nil
}

// Ensures that both the cached file, its metadata and lock are removed without acquiring a lock.
func (c *FileCache[MetadataT]) ensureRemove(key CacheKey) error {
	filePath := filepath.Join(c.rootDir.Path, key.Hex)

	if fileErr := c.ensureRemoveFile(filePath); fileErr != nil {
		slog.Error("Failed to remove cached file", "key", key.Hex, "error", fileErr)
		return fmt.Errorf("%w: failed to remove cached file '%s'", fileErr, filePath)
	}

	delete(c.entries, key) // Remove the entry from the map
	c.locks.Delete(key)    // Remove the lock for this key

	return nil
}

func (c *FileCache[MetadataT]) Get(key CacheKey) (*Entry[MetadataT], error) {
	lock := c.getLock(key)
	lock.Lock() // Upgraded from RLock to Lock to prevent data race on LastAccess
	defer lock.Unlock()

	fileName := filepath.Join(c.rootDir.Path, key.Hex)
	entryMeta, exists := c.entries[key]
	if !exists {
		metrics.Global.Cache.CacheMisses.Increment()
		slog.Debug("Cache miss", "key", key.Hex)
		return nil, ErrCacheEntryNotFound
	}

	dataFile, err := os.Open(fileName)
	if err != nil {
		metrics.Global.Cache.CacheErrors.Increment()
		slog.Error("Failed to open cached data file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to open cached data file '%s'", ErrCacheFileRead, fileName)
	}
	// We don't close dataFile here since we are returning it in the Entry.

	stale := false
	if entryMeta.Expires.Before(time.Now()) {
		stale = true // The entry is stale if the expiration time is in the past
	}

	entryMeta.LastAccess = time.Now()

	metrics.Global.Cache.CacheHits.Increment()
	slog.Debug("Successful cache hit", "key", key.Hex)
	return &Entry[MetadataT]{
		Data:     dataFile,
		Metadata: entryMeta,
		Stale:    stale,
	}, nil
}

func (c *FileCache[MetadataT]) Cache(key CacheKey, data io.Reader, expires time.Time, metadata MetadataT) (*Entry[MetadataT], error) {
	lock := c.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	fileName := filepath.Join(c.rootDir.Path, key.Hex)
	file, err := os.Create(fileName)
	if err != nil {
		metrics.Global.Cache.CacheErrors.Increment()
		slog.Error("Failed to create cache file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to create cache file '%s'", ErrCacheFileCreate, fileName)
	}

	fileSize, err := io.Copy(file, data)
	if err != nil {
		file.Close()
		os.Remove(fileName)
		metrics.Global.Cache.CacheErrors.Increment()
		slog.Error("Failed to write cache file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to write cache file '%s'", ErrCacheFileWrite, fileName)
	}

	if fileSize == 0 {
		file.Close()
		os.Remove(fileName)
		metrics.Global.Cache.CacheErrors.Increment()
		slog.Error("Cache file is empty", "key", key.Hex, "file_size", fileSize)
		return nil, fmt.Errorf("%w: wrote 0 bytes to cache file '%s'", ErrCacheFileEmpty, fileName)
	}

	meta := &EntryMetadata[MetadataT]{
		TimeWritten: time.Now(),
		LastAccess:  time.Now(),
		Expires:     expires,
		Size:        fileSize,
		Object:      metadata,
	}
	c.entries[key] = meta

	metrics.Global.Cache.CacheEntries.Add(1)
	c.addCacheSize(fileSize)

	slog.Debug("Successfully cached data", "key", key.Hex, "size", fileSize)

	slog.Debug("Seeking to the beginning of written cache file...", "key", key.Hex)
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		metrics.Global.Cache.CacheErrors.Increment()
		slog.Error("Failed to seek to start of cache file", "key", key.Hex, "error", err)
		return nil, fmt.Errorf("%w: failed to seek to start of cache file '%s'", ErrCacheFileRead, fileName)
	}

	return &Entry[MetadataT]{
		Data:     file,
		Metadata: meta,
	}, nil
}

func (c *FileCache[MetadataT]) Delete(key CacheKey) error {
	lock := c.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	return c.ensureRemove(key)
}

func (c *FileCache[MetadataT]) UpdateMetadata(key CacheKey, modifier func(*EntryMetadata[MetadataT])) error {
	lock := c.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	meta, exists := c.entries[key]
	if !exists {
		metrics.Global.Cache.CacheErrors.Increment()
		slog.Error("Failed to update metadata for cache entry", "key", key.Hex, "error", ErrCacheEntryNotFound)
		return fmt.Errorf("%w: cache entry for key '%s' does not exist", ErrCacheEntryNotFound, key.Hex)
	}

	modifier(meta)
	meta.LastAccess = time.Now()

	metrics.Global.Cache.CacheHits.Increment()

	slog.Debug("Successfully updated metadata", "key", key.Hex)
	return nil
}

func (c *FileCache[MetadataT]) GetMetadata(key CacheKey) (meta *EntryMetadata[MetadataT], stale bool, err error) {
	lock := c.getLock(key)
	lock.Lock() // Upgraded from RLock to Lock to prevent data race on LastAccess
	defer lock.Unlock()

	metaPtr, exists := c.entries[key]
	if !exists {
		metrics.Global.Cache.CacheMisses.Increment()
		slog.Debug("Cache miss", "key", key.Hex)
		return nil, false, ErrCacheEntryNotFound
	}

	metrics.Global.Cache.CacheHits.Increment()

	stale = false
	if metaPtr.Expires.Before(time.Now()) {
		stale = true // The entry is stale if the expiration time is in the past
	}

	metaPtr.LastAccess = time.Now() // Now safe because we have a full Lock

	slog.Debug("Successfully retrieved metadata", "key", key.Hex)
	return metaPtr, stale, nil
}
