package file

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reservoir/cache"
	"reservoir/metrics"
	"strings"
	"time"
)

func metadataSnapshot[MetadataT any](meta *cache.EntryMetadata[MetadataT]) *cache.EntryMetadata[MetadataT] {
	if meta == nil {
		return nil
	}
	snapshot := *meta
	return &snapshot
}

func (c *Cache[MetadataT]) loadMetadataSidecars() {
	files, err := os.ReadDir(c.rootDir.Path)
	if err != nil {
		slog.Error("Failed to read file cache directory", "path", c.rootDir.Path, "error", err)
		return
	}

	loaded := make(map[string]struct{})
	now := time.Now()
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".meta.json") {
			continue
		}

		keyHex := strings.TrimSuffix(file.Name(), ".meta.json")
		if !isCacheDataFileName(keyHex) {
			_ = removeIfExists(filepath.Join(c.rootDir.Path, file.Name()))
			continue
		}

		key := cache.CacheKey{Hex: keyHex}
		if c.loadMetadataSidecar(key, now) {
			loaded[keyHex] = struct{}{}
		}
	}

	for _, file := range files {
		if file.IsDir() || !isCacheDataFileName(file.Name()) {
			continue
		}
		if _, ok := loaded[file.Name()]; ok {
			continue
		}
		if err := removeIfExists(filepath.Join(c.rootDir.Path, file.Name())); err != nil {
			slog.Error("Failed to remove orphaned cache data file", "file", file.Name(), "error", err)
		}
	}
}

func (c *Cache[MetadataT]) loadMetadataSidecar(key cache.CacheKey, now time.Time) bool {
	metaPath := c.metadataPath(key)
	dataPath := c.dataPath(key)

	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		slog.Error("Failed to read cache metadata sidecar", "key", key.Hex, "error", err)
		_ = removeIfExists(dataPath)
		_ = removeIfExists(metaPath)
		return false
	}

	var meta cache.EntryMetadata[MetadataT]
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		slog.Error("Failed to decode cache metadata sidecar", "key", key.Hex, "error", err)
		_ = removeIfExists(dataPath)
		_ = removeIfExists(metaPath)
		return false
	}

	dataStat, err := os.Stat(dataPath)
	if err != nil || dataStat.Size() == 0 || meta.Expires.Before(now) {
		_ = removeIfExists(dataPath)
		_ = removeIfExists(metaPath)
		return false
	}

	meta.Size = dataStat.Size()
	c.entriesMetadata[key] = &meta
	cache.AddCacheSize(&c.byteSize, meta.Size)
	cache.IncrementCacheEntries()
	return true
}

func (c *Cache[MetadataT]) writeMetadataSidecar(key cache.CacheKey, meta *cache.EntryMetadata[MetadataT]) {
	metaFile, err := os.Create(c.metadataPath(key))
	if err != nil {
		slog.Error("Failed to create cache metadata sidecar", "key", key.Hex, "error", err)
		return
	}
	defer metaFile.Close()

	if err := json.NewEncoder(metaFile).Encode(meta); err != nil {
		slog.Error("Failed to write cache metadata sidecar", "key", key.Hex, "error", err)
	}
}

func (c *Cache[MetadataT]) removeMetadataSidecar(key cache.CacheKey) error {
	return removeIfExists(c.metadataPath(key))
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func isCacheDataFileName(name string) bool {
	if len(name) != 64 {
		return false
	}
	_, err := hex.DecodeString(name)
	return err == nil
}

func (c *Cache[MetadataT]) updateMetadata(key cache.CacheKey, modifier func(*cache.EntryMetadata[MetadataT]), recordMetrics bool) error {
	lock := cache.GetLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	c.mu.RLock()
	meta, exists := c.entriesMetadata[key]
	c.mu.RUnlock()

	if !exists {
		if recordMetrics {
			metrics.Global.Cache.CacheErrors.Increment()
		}
		slog.Error("Failed to update metadata for cache entry", "key", key.Hex, "error", cache.ErrCacheEntryNotFound)
		return fmt.Errorf("%w: cache entry for key '%s' does not exist", cache.ErrCacheEntryNotFound, key.Hex)
	}

	modifier(meta)
	meta.LastAccess = time.Now()
	c.writeMetadataSidecar(key, meta)

	if recordMetrics {
		metrics.Global.Cache.CacheHits.Increment()
	}

	slog.Debug("Successfully updated metadata", "key", key.Hex)
	return nil
}

func (c *Cache[MetadataT]) UpdateMetadata(key cache.CacheKey, modifier func(*cache.EntryMetadata[MetadataT])) error {
	return c.updateMetadata(key, modifier, true)
}

func (c *Cache[MetadataT]) UpdateMetadataQuiet(key cache.CacheKey, modifier func(*cache.EntryMetadata[MetadataT])) error {
	return c.updateMetadata(key, modifier, false)
}

func (c *Cache[MetadataT]) getMetadata(key cache.CacheKey, recordMetrics bool) (meta *cache.EntryMetadata[MetadataT], stale bool, err error) {
	lock := cache.GetLock(c.locks, key)
	lock.Lock() // Upgraded from RLock to Lock to prevent data race on LastAccess
	defer lock.Unlock()

	c.mu.RLock()
	metaPtr, exists := c.entriesMetadata[key]
	c.mu.RUnlock()

	if !exists {
		if recordMetrics {
			metrics.Global.Cache.CacheMisses.Increment()
		}
		slog.Debug("Cache miss", "key", key.Hex)
		return nil, false, cache.ErrCacheEntryNotFound
	}

	if recordMetrics {
		metrics.Global.Cache.CacheHits.Increment()
	}

	stale = false
	if metaPtr.Expires.Before(time.Now()) {
		stale = true // The entry is stale if the expiration time is in the past
	}

	metaPtr.LastAccess = time.Now() // Now safe because we have a full Lock
	metaSnapshot := metadataSnapshot(metaPtr)

	slog.Debug("Successfully retrieved metadata", "key", key.Hex)
	return metaSnapshot, stale, nil
}

func (c *Cache[MetadataT]) GetMetadata(key cache.CacheKey) (meta *cache.EntryMetadata[MetadataT], stale bool, err error) {
	return c.getMetadata(key, true)
}

func (c *Cache[MetadataT]) GetMetadataQuiet(key cache.CacheKey) (meta *cache.EntryMetadata[MetadataT], stale bool, err error) {
	return c.getMetadata(key, false)
}
