package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/utils/assertedpath"
	"reservoir/utils/atomics"
	"reservoir/utils/bytesize"
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
	rootDir         assertedpath.AssertedPath
	entriesMetadata map[CacheKey]*EntryMetadata[MetadataT]
	mu              sync.RWMutex
	locks           []sync.RWMutex
	byteSize        atomics.Int64
	maxCacheSize    atomics.Int64
	janitor         *cacheJanitor[MetadataT]
	subs            config.ConfigSubscriber
}

// NewFileCache creates a new FileCache instance with the specified root directory.
func NewFileCache[MetadataT any](cfg *config.Config, rootDir string, maxCacheSize int64, cleanupInterval time.Duration, shardCount int, ctx context.Context) *FileCache[MetadataT] {
	c := &FileCache[MetadataT]{
		rootDir:         assertedpath.AssertDirectory(rootDir).EnsureCleared(),
		entriesMetadata: make(map[CacheKey]*EntryMetadata[MetadataT]),
		locks:           make([]sync.RWMutex, shardCount),
		byteSize:        atomics.NewInt64(0),
		maxCacheSize:    atomics.NewInt64(maxCacheSize),
	}

	c.subs.Add(cfg.Cache.MaxCacheSize.OnChange(func(newSize bytesize.ByteSize) {
		c.maxCacheSize.Set(newSize.Bytes())
	}))

	c.janitor = newCacheJanitor(cfg, cleanupInterval, cacheFunctions[MetadataT]{
		cacheIterator: func(yield func(key CacheKey, metadata *EntryMetadata[MetadataT]) bool) {
			c.mu.RLock()
			snapshot := maps.Clone(c.entriesMetadata)
			c.mu.RUnlock()

			for key, metadata := range snapshot {
				if !yield(key, metadata) {
					break
				}
			}
		},
		getCacheSize: func() int64 {
			return c.byteSize.Get()
		},
		getCacheLen: func() int {
			c.mu.RLock()
			defer c.mu.RUnlock()
			return len(c.entriesMetadata)
		},
		removeEntry: func(key CacheKey) error {
			return c.ensureRemove(key)
		},
		getLock: func(key CacheKey) *sync.RWMutex {
			return getLock(c.locks, key)
		},
	})
	c.janitor.start(ctx)
	return c
}

func (c *FileCache[MetadataT]) Destroy() {
	c.janitor.stop()
	c.subs.UnsubscribeAll()
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

	decrementCacheEntries()
	decrementCacheSize(&c.byteSize, size)

	return nil
}

// Ensures that both the cached file, its metadata and lock are removed without acquiring a lock.
func (c *FileCache[MetadataT]) ensureRemove(key CacheKey) error {
	filePath := filepath.Join(c.rootDir.Path, key.Hex)

	if fileErr := c.ensureRemoveFile(filePath); fileErr != nil {
		slog.Error("Failed to remove cached file", "key", key.Hex, "error", fileErr)
		return fmt.Errorf("%w: failed to remove cached file '%s'", fileErr, filePath)
	}

	c.mu.Lock()
	delete(c.entriesMetadata, key) // Remove the entry from the map
	c.mu.Unlock()

	return nil
}

func (c *FileCache[MetadataT]) Get(key CacheKey) (*Entry[MetadataT], error) {
	lock := getLock(c.locks, key)
	lock.Lock() // Upgraded from RLock to Lock to prevent data race on LastAccess
	defer lock.Unlock()

	c.mu.RLock()
	entryMeta, exists := c.entriesMetadata[key]
	c.mu.RUnlock()

	if !exists {
		metrics.Global.Cache.CacheMisses.Increment()
		slog.Debug("Cache miss", "key", key.Hex)
		return nil, ErrCacheEntryNotFound
	}

	fileName := filepath.Join(c.rootDir.Path, key.Hex)
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
	maxCacheSize := c.maxCacheSize.Get()
	if c.byteSize.Get() >= maxCacheSize {
		c.janitor.evict(maxCacheSize)
	}

	lock := getLock(c.locks, key)
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

	c.mu.Lock()
	c.entriesMetadata[key] = meta
	c.mu.Unlock()

	incrementCacheEntries()
	addCacheSize(&c.byteSize, fileSize)

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
	lock := getLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	return c.ensureRemove(key)
}

func (c *FileCache[MetadataT]) UpdateMetadata(key CacheKey, modifier func(*EntryMetadata[MetadataT])) error {
	lock := getLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	c.mu.RLock()
	meta, exists := c.entriesMetadata[key]
	c.mu.RUnlock()

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
	lock := getLock(c.locks, key)
	lock.Lock() // Upgraded from RLock to Lock to prevent data race on LastAccess
	defer lock.Unlock()

	c.mu.RLock()
	metaPtr, exists := c.entriesMetadata[key]
	c.mu.RUnlock()

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
